package debt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Baseline flag kinds (research R7).
const (
	FlagRedDefault     = "red default branch"
	FlagForceExit      = "force exit"
	FlagInstallScripts = "install scripts"
)

var (
	forceExitScript = regexp.MustCompile(`--forceExit\b|\bmocha\b[^&|;]*\s--exit\b`)
	forceExitConfig = regexp.MustCompile(`"?forceExit"?\s*:\s*true`)
	jestConfigs     = []string{"jest.config.js", "jest.config.ts", "jest.config.cjs", "jest.config.mjs", "jest.config.json"}
)

// flags reads the baseline flags statically: nothing is run (FR-006, FR-007).
func flags(ctx context.Context, rep *Report, files *fileCache) []Flag {
	var out []Flag
	for _, c := range rep.DefaultChecksFailing {
		out = append(out, Flag{Kind: FlagRedDefault, Where: c, Detail: "fails on " + rep.DefaultBranch + "'s head; a forge baseline on this code fails too"})
	}
	if pkg, err := files.get(ctx, "package.json"); err == nil && pkg != nil {
		var p struct {
			Scripts map[string]string `json:"scripts"`
		}
		_ = json.Unmarshal(pkg, &p)
		var names []string
		for n := range p.Scripts {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if m := forceExitScript.FindString(p.Scripts[n]); m != "" {
				out = append(out, Flag{Kind: FlagForceExit, Where: fmt.Sprintf("package.json:%d", lineOf(pkg, `"`+n+`"`)),
					Detail: fmt.Sprintf("script %q uses %s: tests that leave handles open are hidden, and the run's tests may hang or pass early", n, strings.TrimSpace(m))})
			}
		}
		if loc := forceExitConfig.FindIndex(pkg); loc != nil {
			out = append(out, Flag{Kind: FlagForceExit, Where: fmt.Sprintf("package.json:%d", bytes.Count(pkg[:loc[0]], []byte("\n"))+1),
				Detail: "jest forceExit is on"})
		}
	}
	for _, f := range jestConfigs {
		c, err := files.get(ctx, f)
		if err != nil || c == nil {
			continue
		}
		if loc := forceExitConfig.FindIndex(c); loc != nil {
			out = append(out, Flag{Kind: FlagForceExit, Where: fmt.Sprintf("%s:%d", f, bytes.Count(c[:loc[0]], []byte("\n"))+1), Detail: "jest forceExit is on"})
		}
	}
	if lock, err := files.get(ctx, "package-lock.json"); err == nil && lock != nil {
		if names := installScripts(lock); len(names) > 0 {
			shown := names
			if len(shown) > 5 {
				shown = append(append([]string{}, shown[:5]...), fmt.Sprintf("+%d more", len(names)-5))
			}
			out = append(out, Flag{Kind: FlagInstallScripts, Where: "package-lock.json",
				Detail: fmt.Sprintf("%d package%s with an install script (%s); a native build that downloads fails behind forge's proxy: see docs/targets.md (native builds, --ignore-scripts)",
					len(names), plural(len(names)), strings.Join(shown, ", "))})
		}
	}
	return out
}

// installScripts lists top-level lockfile packages marked hasInstallScript.
func installScripts(lock []byte) []string {
	var l struct {
		Packages map[string]struct {
			HasInstallScript bool `json:"hasInstallScript"`
		} `json:"packages"`
	}
	if json.Unmarshal(lock, &l) != nil {
		return nil
	}
	var out []string
	for k, v := range l.Packages {
		if !v.HasInstallScript {
			continue
		}
		name := k
		if i := strings.LastIndex(k, "node_modules/"); i >= 0 {
			name = k[i+len("node_modules/"):]
		}
		if name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return dedupe(out)
}

func dedupe(s []string) []string {
	var out []string
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}

// lineOf is the 1-based line of the first occurrence of sub, or 1.
func lineOf(b []byte, sub string) int {
	i := bytes.Index(b, []byte(sub))
	if i < 0 {
		return 1
	}
	return bytes.Count(b[:i], []byte("\n")) + 1
}
