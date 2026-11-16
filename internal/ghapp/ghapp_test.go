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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/ghloop"
)

// Cases ported from the former Python pilot's tests/test_ghapp.py (004 US4), with a throwaway key; no network.

const repo = "o/x-pilot-fixture"

func writeKey(t *testing.T, dir string) (string, *rsa.PrivateKey) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "k.pem")
	raw := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return p, k
}

func unb64(t *testing.T, s string) []byte {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestJWTClaimsAndSignature(t *testing.T) {
	path, k := writeKey(t, t.TempDir())
	key, err := LoadKey(path)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := JWT(key, "123", time.Unix(1_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(tok, ".")
	if string(unb64(t, parts[0])) != `{"alg":"RS256","typ":"JWT"}` {
		t.Fatal(string(unb64(t, parts[0])))
	}
	if got := string(unb64(t, parts[1])); got != `{"iat":999940,"exp":1000540,"iss":"123"}` {
		t.Fatal(got)
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&k.PublicKey, crypto.SHA256, sum[:], unb64(t, parts[2])); err != nil {
		t.Fatal(err)
	}
}

func TestTokenRequestIsRepoScopedWithThreePermissions(t *testing.T) {
	got, _ := json.Marshal(TokenRequest(repo))
	if string(got) != `{"permissions":{"contents":"write","issues":"write","pull_requests":"write"},"repositories":["x-pilot-fixture"]}` {
		t.Fatal(string(got))
	}
}

type fakeAPI struct {
	perms      map[string]any
	repos      []any
	mintStatus int
	calls      []string
}

func (f *fakeAPI) serve(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		switch {
		case r.URL.Path == "/app":
			_ = json.NewEncoder(w).Encode(map[string]any{"slug": "specfoundry-pilot"})
		case strings.HasSuffix(r.URL.Path, "/access_tokens"):
			if f.mintStatus != 0 {
				w.WriteHeader(f.mintStatus)
				return
			}
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "ghs_SECRETSECRETSECRET1234", "expires_at": "2026-11-16T11:00:00Z",
				"permissions": f.perms, "repositories": f.repos})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	old := ghloop.API
	ghloop.API = srv.URL
	t.Cleanup(func() { ghloop.API = old })
}

func goodAPI() *fakeAPI {
	return &fakeAPI{perms: map[string]any{"contents": "write", "pull_requests": "write", "issues": "write", "metadata": "read"},
		repos: []any{map[string]any{"full_name": repo}}}
}

func TestMintWritesMode600AndHoldsNoSecret(t *testing.T) {
	dir := t.TempDir()
	key, _ := writeKey(t, dir)
	goodAPI().serve(t)
	out := filepath.Join(dir, "t.env")
	res := MintToken(key, "123", "456", repo, out, time.Now())
	if !res.OK || res.Login != "specfoundry-pilot[bot]" {
		t.Fatalf("%+v", res)
	}
	if raw, _ := json.Marshal(res); bytes.Contains(raw, []byte("ghs_")) {
		t.Fatal("token in result")
	}
	st, _ := os.Stat(out)
	if st.Mode().Perm() != 0o600 {
		t.Fatal(st.Mode())
	}
	if raw, _ := os.ReadFile(out); !strings.Contains(string(raw), "PILOT_SCM_KIND=app") {
		t.Fatal(string(raw))
	}
}

func TestMintProblems(t *testing.T) {
	dir := t.TempDir()
	key, _ := writeKey(t, dir)
	out := filepath.Join(dir, "t2.env")
	(&fakeAPI{mintStatus: 404}).serve(t)
	if MintToken(key, "1", "2", repo, out, time.Now()).OK {
		t.Fatal("404 accepted")
	}
	f := goodAPI()
	f.perms = map[string]any{"contents": "write"}
	f.serve(t)
	r := MintToken(key, "1", "2", repo, out, time.Now())
	if r.OK || !strings.Contains(strings.Join(r.Problems, " "), "issues") {
		t.Fatalf("%+v", r)
	}
	f = goodAPI()
	f.repos = append(f.repos, map[string]any{"full_name": "o/other"})
	f.serve(t)
	if MintToken(key, "1", "2", repo, out, time.Now()).OK {
		t.Fatal("extra repository accepted")
	}
}

func TestBadKeyIsReported(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.pem")
	_ = os.WriteFile(bad, []byte("not a key"), 0o600)
	goodAPI().serve(t)
	if MintToken(bad, "1", "2", repo, filepath.Join(dir, "x.env"), time.Now()).OK {
		t.Fatal("bad key accepted")
	}
}

func app(t *testing.T) App {
	t.Helper()
	d := t.TempDir()
	v, err := config.ValidateValues(map[string]string{"PILOT_FIXTURE_REPO": repo, "PILOT_MODEL": "p/m", "PILOT_MODEL_EGRESS_HOSTS": "h",
		"PILOT_PUBLISHER_CREDENTIAL": "app", "PILOT_APP_ID": "1", "PILOT_APP_INSTALLATION_ID": "2"})
	if err != nil {
		t.Fatal(err)
	}
	return App{Cfg: &config.Config{Values: v, ConfigDir: d, ModelDir: filepath.Join(d, "opencode"), PublisherEnv: filepath.Join(d, "publisher.env")}}
}

func TestFreshMargin(t *testing.T) {
	now := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	if !Fresh(map[string]any{"expires_at": "2026-11-16T10:30:00Z"}, now) || Fresh(map[string]any{"expires_at": "2026-11-16T10:04:00Z"}, now) || Fresh(map[string]any{}, now) {
		t.Fatal("margin")
	}
}

func TestCurrentEnvMintsWhenMissingAndReusesWhenFresh(t *testing.T) {
	a := app(t)
	writeKey(t, a.Cfg.ConfigDir)
	_ = os.Rename(filepath.Join(a.Cfg.ConfigDir, "k.pem"), a.Cfg.AppKey())
	f := goodAPI()
	f.serve(t)
	a.Now = func() time.Time { return time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC) }
	p, err := a.CurrentEnv()
	if err != nil || p != a.EnvPath() {
		t.Fatal(p, err)
	}
	if _, err := a.CurrentEnv(); err != nil {
		t.Fatal(err)
	}
	mints := 0
	for _, c := range f.calls {
		if strings.HasSuffix(c, "/access_tokens") {
			mints++
		}
	}
	if mints != 1 {
		t.Fatalf("minted %d times", mints)
	}
	if login, _ := a.BotLogin(); login != "specfoundry-pilot[bot]" {
		t.Fatal(login)
	}
	raw, _ := os.ReadFile(a.InfoPath())
	if string(raw) != `{"login": "specfoundry-pilot[bot]", "expires_at": "2026-11-16T11:00:00Z", "repositories": ["o/x-pilot-fixture"]}` {
		t.Fatal(string(raw))
	}
}

func TestMintFailureFailsClosed(t *testing.T) {
	a := app(t)
	writeKey(t, a.Cfg.ConfigDir)
	_ = os.Rename(filepath.Join(a.Cfg.ConfigDir, "k.pem"), a.Cfg.AppKey())
	(&fakeAPI{mintStatus: 403}).serve(t)
	_, err := a.CurrentEnv()
	var le *ghloop.Error
	if !errors.As(err, &le) || !strings.HasPrefix(le.Msg, "GitHub App token: ") {
		t.Fatal(err)
	}
}

func TestMissingKeyIsAConfigError(t *testing.T) {
	_, err := app(t).Mint("", "")
	var ce *config.Error
	if !errors.As(err, &ce) || ce.Code != 2 || !strings.Contains(ce.Msg, "pilot/pilot app-key") {
		t.Fatal(err)
	}
}

func TestDiscard(t *testing.T) {
	a := app(t)
	_ = os.WriteFile(a.EnvPath(), []byte("x"), 0o600)
	_ = os.WriteFile(a.InfoPath(), []byte("{}"), 0o600)
	a.Discard()
	if _, err := os.Stat(a.EnvPath()); err == nil {
		t.Fatal("env kept")
	}
	if _, err := os.Stat(a.InfoPath()); err == nil {
		t.Fatal("info kept")
	}
}

func TestStoreKeyRefusesEmptyAndFailedCheck(t *testing.T) {
	a := app(t)
	var out bytes.Buffer
	if a.StoreKey(strings.NewReader(""), &out, "") != 1 || !strings.Contains(out.String(), "not saved: no key read from stdin") {
		t.Fatal(out.String())
	}
	(&fakeAPI{mintStatus: 404}).serve(t)
	dir := t.TempDir()
	key, _ := writeKey(t, dir)
	raw, _ := os.ReadFile(key)
	out.Reset()
	if a.StoreKey(bytes.NewReader(raw), &out, "") != 1 {
		t.Fatal(out.String())
	}
	if !strings.Contains(out.String(), "GitHub App: NOT OK") || !strings.Contains(out.String(), "not saved: no key stored. Fix the App settings and run this again.") {
		t.Fatal(out.String())
	}
	if _, err := os.Stat(a.Cfg.AppKey()); err == nil {
		t.Fatal("key stored after a failed check")
	}
	if _, err := os.Stat(a.Cfg.AppKey() + ".new"); err == nil {
		t.Fatal("staging kept")
	}
	if strings.Contains(out.String(), "PRIVATE KEY") {
		t.Fatal("key printed")
	}
}
