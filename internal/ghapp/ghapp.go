// Package ghapp is the GitHub App as the publisher credential (reference: ghapp.py and
// the former pilot/tools/app_token.py; 004 research R5). The App's private key
// (~/.config/specfoundry/github-app.pem, mode 600) never leaves this host and is never copied
// to the cluster. A 1-hour installation token is minted, limited to the target repository with
// contents, pull requests and issues write, and written to app-token.env (mode 600, next to the
// key), which is then used exactly like publisher.env. Only non-secret facts (bot login,
// expiry) are kept in app-token.json.
//
// Change from the reference, recorded in 016 R7: the key is read and the JWT signed in-process
// (crypto/rsa) instead of in an openssl child process.
package ghapp

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/pyrepr"
	"github.com/tosin2013/chalkeus/internal/secret"
)

// Margin: re-mint when less than 5 minutes are left.
const Margin = 300 * time.Second

// Permissions are the installation token's permissions.
var Permissions = []string{"contents", "pull_requests", "issues"}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// LoadKey parses an RSA private key (PKCS#1 or PKCS#8 PEM).
func LoadKey(path string) (*rsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(raw)
	if blk == nil {
		return nil, errors.New("no PEM block")
	}
	if k, err := x509.ParsePKCS1PrivateKey(blk.Bytes); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(blk.Bytes)
	if err != nil {
		return nil, err
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA key")
	}
	return rk, nil
}

// JWT is the App JWT: RS256, iat now-60 s, exp now+9 min, iss the App id (the reference's
// claims, in its key order).
func JWT(key *rsa.PrivateKey, appID string, now time.Time) (string, error) {
	header := b64url([]byte(`{"alg":"RS256","typ":"JWT"}`))
	iss, _ := json.Marshal(appID)
	t := now.Unix()
	payload := b64url([]byte(fmt.Sprintf(`{"iat":%d,"exp":%d,"iss":%s}`, t-60, t+540, iss)))
	sum := sha256.Sum256([]byte(header + "." + payload))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return header + "." + payload + "." + b64url(sig), nil
}

// TokenRequest is the installation token request body: one repository, three permissions.
func TokenRequest(repo string) map[string]any {
	_, name := ghloop.SplitRepo(repo)
	perms := map[string]string{}
	for _, p := range Permissions {
		perms[p] = "write"
	}
	return map[string]any{"repositories": []string{name}, "permissions": perms}
}

// Result is the mint outcome (no secret in it).
type Result struct {
	OK           bool
	Login        string
	ExpiresAt    string
	Repositories []string
	Permissions  map[string]any
	Problems     []string
}

func call(method, path, bearer string, body any) (int, map[string]any) {
	var r io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		r = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, ghloop.API+path, r)
	if err != nil {
		return 0, nil
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "specfoundry-pilot-app-token")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return 0, nil
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

// MintToken mints an installation token for repo and writes it to outEnv (mode 600) as
// PILOT_SCM_TOKEN / PILOT_SCM_KIND=app / PILOT_SCM_LOGIN=<slug>[bot].
func MintToken(keyPath, appID, installationID, repo, outEnv string, now time.Time) Result {
	key, err := LoadKey(keyPath)
	var j string
	if err == nil {
		j, err = JWT(key, appID, now)
	}
	if err != nil {
		return Result{Problems: []string{"cannot sign with the App key; is it the App's private key?"}}
	}
	st, app := call("GET", "/app", j, nil)
	if st != 200 {
		return Result{Problems: []string{fmt.Sprintf("GitHub rejected the App JWT (HTTP %d): check PILOT_APP_ID and the key", st)}}
	}
	slug, _ := app["slug"].(string)
	login := slug + "[bot]"
	inst, err := strconv.Atoi(installationID)
	if err != nil {
		return Result{Login: login, Problems: []string{"PILOT_APP_INSTALLATION_ID must be numeric"}}
	}
	st, tok := call("POST", fmt.Sprintf("/app/installations/%d/access_tokens", inst), j, TokenRequest(repo))
	if st != 201 {
		return Result{Login: login, Problems: []string{fmt.Sprintf("cannot mint an installation token for %s (HTTP %d): is the App installed on that repository "+
			"with Contents, Pull requests and Issues: read and write?", repo, st)}}
	}
	res := Result{Login: login, Permissions: map[string]any{}}
	res.ExpiresAt, _ = tok["expires_at"].(string)
	if p, ok := tok["permissions"].(map[string]any); ok {
		res.Permissions = p
	}
	for _, k := range Permissions {
		if res.Permissions[k] != "write" {
			res.Problems = append(res.Problems, fmt.Sprintf("the App lacks %s: write on %s", k, repo))
		}
	}
	repos, _ := tok["repositories"].([]any)
	for _, r := range repos {
		m, _ := r.(map[string]any)
		n, _ := m["full_name"].(string)
		res.Repositories = append(res.Repositories, n)
	}
	if len(res.Repositories) > 0 && (len(res.Repositories) != 1 || res.Repositories[0] != repo) {
		res.Problems = append(res.Problems, fmt.Sprintf("token covers %s, expected only %s", pyrepr.Strings(res.Repositories), repo))
	}
	value, _ := tok["token"].(string)
	t := secret.New(value)
	_ = os.Remove(outEnv) // created fresh, so the mode is always 600
	if err := os.WriteFile(outEnv, []byte("PILOT_SCM_TOKEN="+t.Reveal()+"\nPILOT_SCM_KIND=app\nPILOT_SCM_LOGIN="+login+"\n"), 0o600); err != nil {
		res.Problems = append(res.Problems, "cannot write the App token file")
	}
	res.OK = len(res.Problems) == 0
	return res
}

// App is the host side for one config.
type App struct {
	Cfg *config.Config
	Now func() time.Time
	// Repo is the repository tokens are minted for: an item run's target (014), else the fixture.
	Repo string
}

func (a App) repo() string {
	if a.Repo != "" {
		return a.Repo
	}
	return a.Cfg.Repo()
}

// suffix keeps a target's cached token apart from the fixture's (tokens are scoped to one repo).
func (a App) suffix() string {
	if a.Repo == "" || a.Repo == a.Cfg.Repo() {
		return ""
	}
	return "-" + strings.NewReplacer("/", "_", ".", "_").Replace(a.Repo)
}

func (a App) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now().UTC()
}

// EnvPath is app-token.env next to the key.
func (a App) EnvPath() string { return filepath.Join(a.Cfg.ConfigDir, "app-token"+a.suffix()+".env") }

// InfoPath is app-token.json (login, expiry, repositories; no secret).
func (a App) InfoPath() string { return filepath.Join(a.Cfg.ConfigDir, "app-token"+a.suffix()+".json") }

func (a App) info() map[string]any {
	var m map[string]any
	if raw, err := os.ReadFile(a.InfoPath()); err == nil {
		_ = json.Unmarshal(raw, &m)
	}
	return m
}

// Fresh reports whether info's token has more than Margin left at now.
func Fresh(info map[string]any, now time.Time) bool {
	exp, _ := info["expires_at"].(string)
	if exp == "" {
		return false
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", exp)
	if err != nil {
		return false
	}
	return t.Sub(now) > Margin
}

// Mint mints a token with key (default: the stored key) into out (default: EnvPath). Writing to
// EnvPath also records app-token.json.
func (a App) Mint(key, out string) (Result, error) {
	if key == "" {
		key = a.Cfg.AppKey()
	}
	if out == "" {
		out = a.EnvPath()
	}
	if _, err := os.Stat(key); err != nil {
		return Result{}, &config.Error{Msg: fmt.Sprintf("no GitHub App key at %s: store it with `./chalkeus forge app-key`", key), Code: 2}
	}
	if err := config.CheckMode(key); err != nil {
		return Result{}, err
	}
	res := MintToken(key, a.Cfg.Get("PILOT_APP_ID"), a.Cfg.Get("PILOT_APP_INSTALLATION_ID"), a.repo(), out, a.now())
	if res.OK && out == a.EnvPath() {
		raw := pyrepr.JSONDumps(pyrepr.NewOrdered("login", res.Login, "expires_at", res.ExpiresAt, "repositories", toAny(res.Repositories)))
		if err := os.WriteFile(a.InfoPath(), []byte(raw), 0o600); err != nil {
			return res, err
		}
	}
	return res, nil
}

func toAny(s []string) []any {
	out := make([]any, len(s))
	for i, x := range s {
		out[i] = x
	}
	return out
}

// CurrentEnv is the env file with a valid App token, minting one when missing or near expiry.
func (a App) CurrentEnv() (string, error) {
	if _, err := os.Stat(a.EnvPath()); err == nil && Fresh(a.info(), a.now()) {
		return a.EnvPath(), nil
	}
	res, err := a.Mint("", "")
	if err != nil {
		return "", err
	}
	if !res.OK {
		p := res.Problems
		if len(p) == 0 {
			p = []string{"unknown error"}
		}
		return "", &ghloop.Error{Msg: "GitHub App token: " + strings.Join(p, "; ")}
	}
	return a.EnvPath(), nil
}

// BotLogin is the App's bot login (minting if needed).
func (a App) BotLogin() (string, error) {
	if _, err := a.CurrentEnv(); err != nil {
		return "", err
	}
	s, _ := a.info()["login"].(string)
	return s, nil
}

// Discard removes the cached token and its facts.
func (a App) Discard() {
	_ = os.Remove(a.EnvPath())
	_ = os.Remove(a.InfoPath())
}

// Check is `./chalkeus forge app-token --check`: mint a token, run the write probe with it, discard it.
// canary is this repository (the probe's must-fail write target).
func (a App) Check(out io.Writer, key, canary string) int {
	tmp := filepath.Join(a.Cfg.ConfigDir, "app-token.check.env")
	defer func() { _ = os.Remove(tmp) }()
	res, err := a.Mint(key, tmp)
	if err != nil {
		fmt.Fprintln(out, "GitHub App: NOT OK")
		fmt.Fprintf(out, "  - %v\n", err)
		return 1
	}
	if !res.OK {
		fmt.Fprintln(out, "GitHub App: NOT OK")
		for _, p := range res.Problems {
			fmt.Fprintf(out, "  - %s\n", p)
		}
		return 1
	}
	probe := ghloop.VerifyFile(tmp, a.repo(), canary, true)
	fmt.Fprintf(out, "GitHub App %s: token expires %s, repositories %s\n", res.Login, res.ExpiresAt, pyrepr.Strings(res.Repositories))
	ghloop.Report(out, probe)
	if probe.OK {
		return 0
	}
	return 1
}

// StoreKey is `./chalkeus forge app-key`: copy the PEM from in to a staging file (mode 600) and keep it
// only if Check passes (the 006 F3 rule).
func (a App) StoreKey(in io.Reader, out io.Writer, canary string) int {
	key := a.Cfg.AppKey()
	staging := key + ".new"
	f, err := os.OpenFile(staging, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	var n int64
	if err == nil {
		n, err = io.Copy(f, in)
		_ = f.Close()
	}
	if err != nil || n == 0 {
		_ = os.Remove(staging)
		fmt.Fprintln(out, "  not saved: no key read from stdin")
		return 1
	}
	if a.Check(out, staging, canary) != 0 {
		_ = os.Remove(staging)
		kept := "no key stored"
		if _, err := os.Stat(key); err == nil {
			kept = "your previous key is unchanged"
		}
		fmt.Fprintf(out, "  not saved: %s. Fix the App settings and run this again.\n", kept)
		return 1
	}
	if err := os.Rename(staging, key); err != nil {
		fmt.Fprintf(out, "  not saved: %v\n", err)
		return 1
	}
	fmt.Fprintf(out, "  saved: %s\n", key)
	return 0
}
