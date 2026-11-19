package config

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/tosin2013/specfoundry/internal/pyrepr"
)

// LoginSource turns an OpenCode login (`/connect` or `opencode auth login`, stored in OpenCode's
// auth.json) into a temporary OpenCode config dir that PlanImport accepts: one provider block
// with settings.baseURL and an apiKey file reference, plus that key file (mode 0600). The import
// then runs exactly as for a hand-written config. The key is never printed.
//
// The API endpoint comes from, in order: baseURL (the --base-url flag), the provider's
// settings.baseURL in cfgDir/opencode.json, or OpenCode's own provider catalog cache
// (cacheDir/models.json, field "api"). Without one, the import refuses: the worker's egress
// allowlist needs the host.
func LoginSource(authFile, cfgDir, cacheDir, model, baseURL string) (string, func(), error) {
	noop := func() {}
	if model == "" || !strings.Contains(model, "/") {
		return "", noop, cerr("--from-login needs --model <provider>/<model>, for example the model you use in OpenCode")
	}
	provName, modelID, _ := strings.Cut(model, "/")
	raw, err := os.ReadFile(authFile)
	if err != nil {
		return "", noop, cerr("no OpenCode login at %s: log in with OpenCode first (/connect in the TUI, or `opencode auth login`). "+
			"OpenCode 2.2 and later keep logins in their database: pipe them in with "+
			"`opencode auth export | pilot/pilot import-opencode --from-login --auth-file - --model <provider>/<model>`", authFile)
	}
	auth, err := parseLogins(raw)
	if err != nil {
		return "", noop, cerr("%s is not an OpenCode login file (auth.json) or `opencode auth export` output", authFile)
	}
	entry, ok := auth[provName]
	if !ok {
		return "", noop, cerr("you are not logged in to provider %s in OpenCode (%s)", pyrepr.Str(provName), authFile)
	}
	if typ, _ := entry["type"].(string); typ != "api" && typ != "key" {
		return "", noop, cerr("your OpenCode login for %s is a %s login, not an API key. The pilot runs unattended, so it uses API keys only "+
			"(subscription logins are meant for interactive use, and vendors' terms may not allow automated use). "+
			"Create an API key for %s, connect it in OpenCode, and import again", pyrepr.Str(provName), pyrepr.Str(typ), provName)
	}
	key, _ := entry["key"].(string)
	if strings.TrimSpace(key) == "" {
		return "", noop, cerr("your OpenCode login for %s has no API key", pyrepr.Str(provName))
	}

	// The provider block: the user's own (if any) for its models and settings, else a minimal one.
	prov := map[string]any{}
	if cfgRaw, err := os.ReadFile(filepath.Join(cfgDir, "opencode.json")); err == nil {
		var doc map[string]any
		if json.Unmarshal(cfgRaw, &doc) == nil {
			provs, _ := doc["providers"].(map[string]any)
			if p, ok := provs[provName].(map[string]any); ok {
				prov = p
			}
		}
	}
	settings, _ := prov["settings"].(map[string]any)
	if settings == nil {
		settings = map[string]any{}
	}
	if baseURL == "" {
		baseURL, _ = settings["baseURL"].(string)
	}
	if baseURL == "" {
		baseURL = catalogAPI(cacheDir, provName)
	}
	u, err := url.Parse(baseURL)
	if baseURL == "" || err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return "", noop, cerr("no https API endpoint known for provider %s: pass --base-url https://… (from the provider's docs; "+
			"`opencode models --verbose` may show it)", pyrepr.Str(provName))
	}
	keyName := provName + "_api_key"
	settings["baseURL"] = baseURL
	settings["apiKey"] = "{file:" + workerDir + keyName + "}"
	for k := range settings { // drop any other secret-looking setting: the key file is the only secret
		if k != "apiKey" && secretFieldRE.MatchString(k) {
			delete(settings, k)
		}
	}
	prov["settings"] = settings
	delete(prov, "apiKey")
	models, _ := prov["models"].(map[string]any)
	if models == nil {
		models = map[string]any{}
	}
	if _, ok := models[modelID]; !ok {
		models[modelID] = map[string]any{}
	}
	prov["models"] = models

	dir, err := os.MkdirTemp("", "pilot-login-")
	if err != nil {
		return "", noop, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := os.WriteFile(filepath.Join(dir, keyName), []byte(strings.TrimSpace(key)+"\n"), 0o600); err != nil {
		cleanup()
		return "", noop, err
	}
	doc := map[string]any{"$schema": "https://opencode.ai/config.json", "providers": map[string]any{provName: prov}, "model": model}
	out, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "opencode.json"), append(out, '\n'), 0o600); err != nil {
		cleanup()
		return "", noop, err
	}
	return dir, cleanup, nil
}

// parseLogins reads OpenCode's logins in either format, keyed by provider: auth.json (OpenCode
// before 2.2: {"<provider>": {"type": "api", "key": …}}) or `opencode auth export` (2.2 keeps
// logins in its database: [{"integrationID": "<provider>", "active": true, "value": {"type":
// "key", "key": …}}]). From the export, only active entries count.
func parseLogins(raw []byte) (map[string]map[string]any, error) {
	var auth map[string]map[string]any
	if err := json.Unmarshal(raw, &auth); err == nil {
		return auth, nil
	}
	var list []struct {
		IntegrationID string         `json:"integrationID"`
		Active        bool           `json:"active"`
		Value         map[string]any `json:"value"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	auth = map[string]map[string]any{}
	for _, e := range list {
		if e.Active && e.IntegrationID != "" && e.Value != nil {
			auth[e.IntegrationID] = e.Value
		}
	}
	return auth, nil
}

// catalogAPI is the provider's API URL from OpenCode's cached provider catalog, or "".
func catalogAPI(cacheDir, prov string) string {
	raw, err := os.ReadFile(filepath.Join(cacheDir, "models.json"))
	if err != nil {
		return ""
	}
	var cat map[string]map[string]any
	if json.Unmarshal(raw, &cat) != nil {
		return ""
	}
	api, _ := cat[prov]["api"].(string)
	return api
}
