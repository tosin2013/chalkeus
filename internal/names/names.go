// Package names is 027's rename layer: the new names (FORGE_*, forge.env, CHALKEUS_CONFIG) are
// read first, the old ones (PILOT_*, pilot.env, SPECFOUNDRY_CONFIG) still work, and each old name
// in use gets a notice once (specs/027-chalkeus-rename/contracts/name-map.md). Inside the pilot,
// settings keep their historical PILOT_* keys, so nothing else changes: Canonical maps FORGE_X to
// PILOT_X. It imports nothing from this module, so every package can use it.
package names

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	NewPrefix = "FORGE_"
	OldPrefix = "PILOT_"
)

// Out receives notices. Under `go test` they are discarded unless a test sets Out, so no test
// prints them or leaves a marker on the developer's machine.
var Out io.Writer = os.Stderr

// NoticeDir holds one marker per notice already shown, outside the config folder, so the tool
// never writes into it (027 FR-006). Under `go test` it is "" (no markers) unless a test sets it.
var NoticeDir = func() string {
	if testing.Testing() {
		return ""
	}
	d, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "chalkeus", "notices")
}

// shown remembers notices within one process when there's no marker folder.
var shown = map[string]bool{}

// Notice prints text once per id on this machine (when the marker can't be written, every time:
// a notice shown twice is better than one lost).
func Notice(id, text string) {
	dir := NoticeDir()
	sum := sha256.Sum256([]byte(id))
	marker := ""
	if dir != "" {
		marker = filepath.Join(dir, hex.EncodeToString(sum[:8]))
		if _, err := os.Stat(marker); err == nil {
			return
		}
	} else if shown[id] {
		return
	}
	shown[id] = true
	out := Out
	if testing.Testing() && out == os.Stderr {
		out = io.Discard
	}
	fmt.Fprintf(out, "chalkeus: %s\n", text)
	if marker != "" && os.MkdirAll(dir, 0o700) == nil {
		_ = os.WriteFile(marker, []byte(id+"\n"), 0o600)
	}
}

// Env reads the environment variable FORGE_<suffix>, else PILOT_<suffix> (with a notice).
func Env(suffix string) string {
	v, _ := LookupEnv(suffix)
	return v
}

// LookupEnv is Env that also reports whether either name is set.
func LookupEnv(suffix string) (string, bool) {
	if v, ok := os.LookupEnv(NewPrefix + suffix); ok {
		if _, old := os.LookupEnv(OldPrefix + suffix); old {
			Notice("env-both:"+suffix, fmt.Sprintf("%s%s is set and wins; %s%s is ignored (remove it)", NewPrefix, suffix, OldPrefix, suffix))
		}
		return v, true
	}
	if v, ok := os.LookupEnv(OldPrefix + suffix); ok {
		Notice("env-old:"+suffix, fmt.Sprintf("%s%s is the old name of %s%s; it still works, but rename it", OldPrefix, suffix, NewPrefix, suffix))
		return v, true
	}
	return "", false
}

// Display is the name an operator sees for an internal settings key: PILOT_X shows as FORGE_X.
// The publisher credential keys (PILOT_SCM_TOKEN, _KIND, _LOGIN) keep their names (name map);
// 037's host settings (PILOT_SCM, PILOT_SCM_HOST, PILOT_SCM_CA_BUNDLE) show as FORGE_; other
// text is unchanged.
func Display(key string) string {
	if strings.HasPrefix(key, OldPrefix) && !credentialKeys[key] {
		return NewPrefix + strings.TrimPrefix(key, OldPrefix)
	}
	return key
}

var credentialKeys = map[string]bool{"PILOT_SCM_TOKEN": true, "PILOT_SCM_KIND": true, "PILOT_SCM_LOGIN": true}

// Canonical maps a settings file's keys to the pilot's internal names: FORGE_X becomes PILOT_X;
// when both are given, FORGE_X wins. file names the source for the notices (a path), and a
// notice lists the old names in use, once per file.
func Canonical(values map[string]string, file string) map[string]string {
	out := map[string]string{}
	var old, ignored []string
	for k, v := range values {
		if strings.HasPrefix(k, NewPrefix) {
			out[OldPrefix+strings.TrimPrefix(k, NewPrefix)] = v
		}
	}
	for k, v := range values {
		switch {
		case strings.HasPrefix(k, NewPrefix):
		case strings.HasPrefix(k, OldPrefix):
			if _, ok := values[NewPrefix+strings.TrimPrefix(k, OldPrefix)]; ok {
				ignored = append(ignored, k)
				continue
			}
			old = append(old, k)
			out[k] = v
		default:
			out[k] = v
		}
	}
	sort.Strings(old)
	sort.Strings(ignored)
	if len(ignored) > 0 {
		Notice("file-both:"+file+":"+strings.Join(ignored, ","), fmt.Sprintf("%s gives both names for %s; the FORGE_ names win and these are ignored (remove them)", file, strings.Join(ignored, ", ")))
	}
	if len(old) > 0 {
		Notice("file-old:"+file, fmt.Sprintf("%s uses the old PILOT_ names (%s); they still work, but rename them to FORGE_ (same suffix)", file, strings.Join(old, ", ")))
	}
	return out
}
