package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tosin2013/chalkeus/internal/pyrepr"
)

// `./chalkeus forge import-opencode`: reuse an existing host OpenCode setup as the worker's config
// (reference: the former Python pilot's import_opencode.py). Only the chosen model's provider and
// the key files it references are copied; key files are copied with `install -m 600` and never
// read by the pilot. The model host is never printed.

const workerDir = "~/.config/opencode/"

var (
	fileRefRE = regexp.MustCompile(`^\{file:(.+)\}$`)
	envRefRE  = regexp.MustCompile(`^\{env:(.+)\}$`)
	keepTop   = map[string]bool{"$schema": true, "providers": true, "model": true}
)

// ImportPlan is what import-opencode will copy.
type ImportPlan struct {
	Doc              map[string]any
	Keys             []string
	Model, Host      string
	DroppedProviders []string
	DroppedSettings  []string
}

type secretRef struct{ where, val string }

func walkSecrets(o any, path string, out *[]secretRef) {
	switch x := o.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if s, ok := x[k].(string); ok && secretFieldRE.MatchString(k) {
				*out = append(*out, secretRef{path + k, s})
			}
			walkSecrets(x[k], path+k+".", out)
		}
	case []any:
		for _, e := range x {
			walkSecrets(e, path, out)
		}
	}
}

func cerr(format string, a ...any) error { return errf(2, format, a...) }

// PlanImport validates the host config and decides what to copy.
func PlanImport(src, model string) (*ImportPlan, error) {
	cfgFile := filepath.Join(src, "opencode.json")
	raw, err := os.ReadFile(cfgFile)
	if err != nil {
		extra := ""
		if exists(filepath.Join(src, "opencode.jsonc")) {
			extra = " (opencode.jsonc is not supported: save it as plain JSON)"
		}
		return nil, cerr("no OpenCode config at %s%s", cfgFile, extra)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, cerr("%s is not valid JSON: %v", cfgFile, err)
	}
	if model == "" {
		model, _ = doc["model"].(string)
	}
	if model == "" || !strings.Contains(model, "/") {
		return nil, cerr("no default model: set \"model\": \"<provider>/<model>\" in opencode.json, or pass --model")
	}
	provName, modelID, _ := strings.Cut(model, "/")
	provs, _ := doc["providers"].(map[string]any)
	prov, ok := provs[provName].(map[string]any)
	if !ok {
		return nil, cerr("provider %s (from model %s) is not defined under \"providers\"", pyrepr.Str(provName), pyrepr.Str(model))
	}
	models, _ := prov["models"].(map[string]any)
	if _, ok := models[modelID]; !ok {
		return nil, cerr("model %s is not listed under providers.%s.models", pyrepr.Str(modelID), provName)
	}
	settings, _ := prov["settings"].(map[string]any)
	base, _ := settings["baseURL"].(string)
	u, _ := url.Parse(base)
	host := ""
	if u != nil {
		host = u.Hostname()
	}
	if host == "" {
		return nil, cerr("providers.%s.settings.baseURL must be a full https:// URL", provName)
	}
	var refs []secretRef
	walkSecrets(prov, "providers."+provName+".", &refs)
	keySet := map[string]bool{}
	for _, r := range refs {
		if envRefRE.MatchString(r.val) {
			return nil, cerr("%s uses an {env:...} reference; the worker can't see your environment. Put the key in %s%s_api_key and use \"{file:%s%s_api_key}\"",
				r.where, workerDir, provName, workerDir, provName)
		}
		m := fileRefRE.FindStringSubmatch(r.val)
		if m == nil {
			return nil, cerr("%s holds a literal secret; move it to %s%s_api_key and use \"{file:%s%s_api_key}\" (value not shown)",
				r.where, workerDir, provName, workerDir, provName)
		}
		ref := m[1]
		name := filepath.Base(ref)
		if !strings.HasPrefix(ref, workerDir) || strings.Contains(ref[len(workerDir):], "/") || !strings.HasSuffix(name, "_api_key") {
			return nil, cerr("%s must reference %s<name>_api_key (got %s); the worker mounts only those files", r.where, workerDir, pyrepr.Str(ref))
		}
		if st, err := os.Stat(filepath.Join(src, name)); err != nil || !st.Mode().IsRegular() {
			return nil, cerr("%s references %s, which is missing from %s", r.where, name, src)
		}
		keySet[name] = true
	}
	if len(keySet) == 0 {
		return nil, cerr("providers.%s has no apiKey; add \"apiKey\": \"{file:%s%s_api_key}\"", provName, workerDir, provName)
	}
	p := &ImportPlan{Model: model, Host: host}
	for k := range keySet {
		p.Keys = append(p.Keys, k)
	}
	sort.Strings(p.Keys)
	for k := range provs {
		if k != provName {
			p.DroppedProviders = append(p.DroppedProviders, k)
		}
	}
	sort.Strings(p.DroppedProviders)
	for k := range doc {
		if !keepTop[k] {
			p.DroppedSettings = append(p.DroppedSettings, k)
		}
	}
	sort.Strings(p.DroppedSettings)
	p.Doc = map[string]any{"providers": map[string]any{provName: prov}, "model": model}
	if s, ok := doc["$schema"]; ok {
		p.Doc["$schema"] = s
	}
	return p, nil
}

func install(args ...string) error {
	if out, err := exec.Command("install", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("install %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}

// RunImport copies the plan into ~/.config/specfoundry/opencode and writes or checks pilot.env.
func RunImport(src, model, fixtureRepo, root string, force bool, out io.Writer) error {
	p, err := PlanImport(src, model)
	if err != nil {
		return err
	}
	d := Dir()
	dest := filepath.Join(d, "opencode")
	if exists(filepath.Join(dest, "opencode.json")) && !force {
		return cerr("%s already has an opencode.json; pass --force to replace it", dest)
	}
	if fixtureRepo != "" {
		if err := RepoGuard(fixtureRepo, OriginRepo(root)); err != nil {
			return err
		}
	}
	if err := install("-d", "-m", "700", d, dest); err != nil {
		return err
	}
	for _, name := range p.Keys {
		if err := install("-m", "600", filepath.Join(src, name), filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	target := filepath.Join(dest, "opencode.json")
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(p.Doc) // ends with a newline, like the reference
	if err := os.WriteFile(target, []byte(buf.String()), 0o600); err != nil {
		return err
	}
	_ = os.Chmod(target, 0o600)
	if err := CheckModelDir(dest); err != nil {
		return err
	}
	fmt.Fprintf(out, "imported OpenCode config: %s -> %s\n", src, dest)
	fmt.Fprintf(out, "  model: %s\n", p.Model)
	fmt.Fprintf(out, "  key files: %s\n", strings.Join(p.Keys, ", "))
	if len(p.DroppedProviders) > 0 {
		fmt.Fprintf(out, "  left out other providers: %s\n", strings.Join(p.DroppedProviders, ", "))
	}
	if len(p.DroppedSettings) > 0 {
		fmt.Fprintf(out, "  left out settings the worker can't use: %s\n", strings.Join(p.DroppedSettings, ", "))
	}
	pilotEnv := SettingsFile(d) // forge.env, or an existing pilot.env (027)
	if !exists(pilotEnv) {
		body := fmt.Sprintf("FORGE_FIXTURE_REPO=%s\nFORGE_FIXTURE_BRANCH=main\nFORGE_MODEL=%s\nFORGE_MODEL_EGRESS_HOSTS=%s\n"+
			"FORGE_SPEND_CAP_USD=1.00\nFORGE_TOKEN_CAP=2000000\nFORGE_PROVIDER_CAP_DECLARED=false\n", fixtureRepo, p.Model, p.Host)
		if err := os.WriteFile(pilotEnv, []byte(body), 0o600); err != nil {
			return err
		}
		fmt.Fprintf(out, "wrote %s (model, its egress host, default caps)\n", pilotEnv)
		if fixtureRepo == "" {
			fmt.Fprintln(out, "  next: set FORGE_FIXTURE_REPO=<you>/<you>-pilot-fixture in it")
		}
		return nil
	}
	envRaw, err := os.ReadFile(pilotEnv)
	if err != nil {
		return err
	}
	values := ParseSettings(string(envRaw), pilotEnv)
	base := filepath.Base(pilotEnv)
	ok := true
	if values["PILOT_MODEL"] != p.Model {
		cur := values["PILOT_MODEL"]
		if cur == "" {
			cur = "unset"
		}
		fmt.Fprintf(out, "  %s: set FORGE_MODEL=%s (it is %s)\n", base, p.Model, cur)
		ok = false
	}
	found := false
	for _, h := range splitList(values["PILOT_MODEL_EGRESS_HOSTS"], false) {
		found = found || h == p.Host
	}
	if !found {
		fmt.Fprintf(out, "  %s: FORGE_MODEL_EGRESS_HOSTS doesn't include your provider's host; set it to the host part of the provider's baseURL\n", base)
		ok = false
	}
	verdict := "needs the changes above"
	if ok {
		verdict = "matches"
	}
	fmt.Fprintf(out, "%s: %s (%s left unchanged)\n", base, verdict, pilotEnv)
	return nil
}
