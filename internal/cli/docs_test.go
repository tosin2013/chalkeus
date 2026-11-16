package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/audit"
)

// The docs check (spec 006, contracts/docs-check.md), run by `go test ./internal/cli`:
//
//	D1  every `pilot/pilot <sub> [--flags]` in an sh/bash/shell code block (or inline code) is a
//	    real command and flag of this CLI (read from the command tree, not from --help)
//	D2  repo paths exist: relative Markdown links, and inline-code tokens that start with a
//	    top-level repo entry (pilot/, specs/, docs/, …). Domains, owner/repo slugs, ~ paths,
//	    <placeholders>, globs and URLs are never treated as paths (006 analysis H1)
//	D3  quoted versions of kind, kubectl, Calico, spec and OpenCode match pilot/pins.env
//	D4  nothing matches the evidence redaction patterns (token-like strings)
//
// To check other files: DOCS_CHECK_FILES="a.md b.md" go test ./internal/cli -run TestDocsCheck

var (
	topLevel   = []string{"pilot/", "specs/", "docs/", "test/", ".github/", ".specify/", "internal/", "cmd/"}
	topFiles   = map[string]bool{"README.md": true, "CONTRIBUTING.md": true, "CODE_OF_CONDUCT.md": true, "LICENSE": true, "AGENTS.md": true}
	shellLangs = map[string]bool{"sh": true, "bash": true, "shell": true, "console": true}
	fenceRe    = regexp.MustCompile("^\\s*```(\\S*)")
	linkRe     = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	inlineRe   = regexp.MustCompile("`([^`\n]+)`")
	commandRe  = regexp.MustCompile(`pilot/pilot(?:\s+([^\s;&|#]+))?((?:\s+[^\s;&|#]+)*)`)
	versionRes = []struct {
		re        *regexp.Regexp
		key, name string
	}{
		{regexp.MustCompile(`\bkind v(\d+\.\d+\.\d+)`), "KIND_VERSION", "kind"},
		{regexp.MustCompile(`\bkubectl v(\d+\.\d+\.\d+)`), "KUBECTL_VERSION", "kubectl"},
		{regexp.MustCompile(`(?i)\bCalico v(\d+\.\d+\.\d+)`), "CALICO_VERSION", "Calico"},
		{regexp.MustCompile(`\b(?:spec|spec-status)[ _]v?(\d+\.\d+\.\d+)`), "SPEC_VERSION", "spec"},
		{regexp.MustCompile(`(?i)\bOpenCode v?(\d+\.\d+\.\d+)`), "OPENCODE_VERSION", "OpenCode"},
	}
)

type docsChecker struct {
	root string
	cli  *cobra.Command
	pins map[string]string
}

func newDocsChecker(t *testing.T) *docsChecker {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	pins := map[string]string{}
	raw, err := os.ReadFile(filepath.Join(root, "pilot", "pins.env"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, "="); ok && line != "" && !strings.HasPrefix(line, "#") {
			pins[k] = v
		}
	}
	return &docsChecker{root: root, cli: NewRoot("test"), pins: pins}
}

func subcommand(c *cobra.Command, name string) *cobra.Command {
	for _, s := range c.Commands() {
		if s.Name() == name && name != "help" {
			return s
		}
	}
	return nil
}

func hasFlag(c *cobra.Command, flag string) bool {
	name := strings.TrimPrefix(flag, "--")
	return name == "help" || c.Flags().Lookup(name) != nil || c.InheritedFlags().Lookup(name) != nil ||
		c.PersistentFlags().Lookup(name) != nil || c.Root().PersistentFlags().Lookup(name) != nil
}

// checkCommand returns D1 problems for each `pilot/pilot …` in s.
func (d *docsChecker) checkCommand(s string) []string {
	var errs []string
	for _, m := range commandRe.FindAllStringSubmatch(s, -1) {
		sub, toks := m[1], strings.Fields(m[2])
		for strings.HasPrefix(sub, "-") && sub != "--help" { // global flags before the subcommand
			if strings.HasPrefix(sub, "--") && !hasFlag(d.cli, strings.SplitN(sub, "=", 2)[0]) {
				errs = append(errs, fmt.Sprintf("unknown flag %s", sub))
			}
			if len(toks) == 0 {
				sub = ""
				break
			}
			sub, toks = toks[0], toks[1:]
		}
		if sub == "" || strings.HasPrefix(sub, "<") || sub == "--help" || sub == "…" || sub == "..." {
			continue
		}
		cmd := subcommand(d.cli, sub)
		if cmd == nil {
			errs = append(errs, fmt.Sprintf("unknown subcommand %q", sub))
			continue
		}
		if cmd.HasSubCommands() && len(toks) > 0 && !strings.HasPrefix(toks[0], "-") && !strings.HasPrefix(toks[0], "<") {
			nested := subcommand(cmd, toks[0])
			if nested == nil {
				errs = append(errs, fmt.Sprintf("unknown subcommand %q", sub+" "+toks[0]))
				continue
			}
			sub, cmd, toks = sub+" "+toks[0], nested, toks[1:]
		}
		for _, tok := range toks {
			if strings.HasPrefix(tok, "--") {
				flag, _, _ := strings.Cut(tok, "=")
				if !hasFlag(cmd, flag) {
					errs = append(errs, fmt.Sprintf("unknown flag %s for %q", flag, sub))
				}
			}
		}
	}
	return errs
}

func cleanTok(tok string) string { return strings.TrimRight(strings.TrimSpace(tok), ".,;:)") }

func looksLikeRepoPath(tok string) bool {
	tok = cleanTok(tok)
	if tok == "" || strings.ContainsAny(tok, " <>*{}$") || strings.Contains(tok, "://") ||
		strings.HasPrefix(tok, "~") || strings.HasPrefix(tok, "/") || strings.HasPrefix(tok, "-") {
		return false
	}
	for _, p := range topLevel {
		if strings.HasPrefix(tok, p) {
			return true
		}
	}
	return topFiles[tok]
}

func (d *docsChecker) pathExists(tok, doc string) bool {
	tok, _, _ = strings.Cut(cleanTok(tok), "#")
	if tok == "" {
		return true
	}
	for _, base := range []string{filepath.Dir(doc), d.root} {
		if _, err := os.Stat(filepath.Join(base, tok)); err == nil {
			return true
		}
	}
	return false
}

func (d *docsChecker) checkFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	rel := path
	if r, err := filepath.Rel(d.root, path); err == nil && !strings.HasPrefix(r, "..") {
		rel = r
	}
	var errs []string
	inBlock, lang := false, ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		where := fmt.Sprintf("%s:%d:", rel, n)
		if m := fenceRe.FindStringSubmatch(line); m != nil {
			if inBlock {
				inBlock, lang = false, ""
			} else {
				inBlock, lang = true, strings.ToLower(m[1])
			}
			continue
		}
		if audit.Patterns.MatchString(line) {
			errs = append(errs, where+" token-like string (D4)")
		}
		for _, v := range versionRes {
			want := strings.TrimPrefix(d.pins[v.key], "v")
			for _, m := range v.re.FindAllStringSubmatch(line, -1) {
				if want != "" && m[1] != want {
					errs = append(errs, fmt.Sprintf("%s %s %s != pins %s (D3)", where, v.name, m[1], want))
				}
			}
		}
		if inBlock {
			if shellLangs[lang] && strings.Contains(line, "pilot/pilot") {
				for _, e := range d.checkCommand(line) {
					errs = append(errs, fmt.Sprintf("%s %s (D1)", where, e))
				}
			}
			continue
		}
		for _, m := range linkRe.FindAllStringSubmatch(line, -1) {
			target := m[1]
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") ||
				strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			if !d.pathExists(target, path) {
				errs = append(errs, fmt.Sprintf("%s missing link target %s (D2)", where, target))
			}
		}
		for _, m := range inlineRe.FindAllStringSubmatch(line, -1) {
			tok := m[1]
			if looksLikeRepoPath(tok) {
				if !d.pathExists(tok, path) {
					errs = append(errs, fmt.Sprintf("%s missing path %s (D2)", where, tok))
				}
			} else if strings.Contains(tok, "pilot/pilot") {
				for _, e := range d.checkCommand(tok) {
					errs = append(errs, fmt.Sprintf("%s %s (D1)", where, e))
				}
			}
		}
	}
	return errs, sc.Err()
}

func (d *docsChecker) defaultFiles(t *testing.T) []string {
	if env := strings.Fields(os.Getenv("DOCS_CHECK_FILES")); len(env) > 0 {
		var out []string
		for _, f := range env {
			a, _ := filepath.Abs(filepath.Join(d.root, f))
			if filepath.IsAbs(f) {
				a = f
			}
			out = append(out, a)
		}
		return out
	}
	files, err := filepath.Glob(filepath.Join(d.root, "docs", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	for _, f := range []string{"README.md", "CONTRIBUTING.md", "AGENTS.md", "pilot/README.md", ".claude/skills/pilot/SKILL.md"} {
		files = append(files, filepath.Join(d.root, f))
	}
	return files
}

// TestDocsCheck runs D1–D4 over the operator and contributor docs.
func TestDocsCheck(t *testing.T) {
	d := newDocsChecker(t)
	files := d.defaultFiles(t)
	var errs []string
	for _, f := range files {
		e, err := d.checkFile(f)
		if err != nil {
			t.Fatal(err)
		}
		errs = append(errs, e...)
	}
	if len(errs) > 0 {
		t.Fatalf("docs check: %d problem(s) in %d file(s):\n%s", len(errs), len(files), strings.Join(errs, "\n"))
	}
}

// Unit cases (ported from the former Python pilot's tests/test_docs_check.py), including 006 analysis H1.
func TestDocsCheckRules(t *testing.T) {
	d := newDocsChecker(t)
	check := func(text string) []string {
		p := filepath.Join(t.TempDir(), "doc.md")
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		e, err := d.checkFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	clean := []string{
		"```sh\npilot/pilot run\npilot/pilot up --local-build\npilot/pilot gate A --root x\n```\n",
		"```sh\npilot/pilot github watch --run x --detach\npilot/pilot -vv run\n```\n",
		"```text\npilot/pilot frobnicate\n```\n",
		"See `pilot/pins.env` and [the guide](docs/harnesses.md).\n",
		"`ghcr.io/tosin2013/specfoundry-pilot-worker@sha256:abc` `tosin2013/craz3-pilot-fixture` " +
			"`anomalyco/opencode` `kelos-dev/kelos` `~/.config/specfoundry/pilot.env` " +
			"`<owner>/<name>-pilot-fixture` `https://github.com/x/y` `refs/heads/main` " +
			"`localhost:5001/pilot-worker` `/etc/containerd/certs.d`\n",
		"Installs kind v0.27.0 and kubectl v1.32.2; Calico v3.29.3; spec 0.2.0; OpenCode v2.2.7.\n",
		"Paste the token into the hidden prompt; never into chat.\n",
	}
	for _, c := range clean {
		if e := check(c); len(e) != 0 {
			t.Errorf("%q: unexpected %v", c, e)
		}
	}
	flagged := []struct{ text, needle string }{
		{"```sh\npilot/pilot run --bogus\n```\n", "--bogus"},
		{"```bash\ncd ~/specfoundry && pilot/pilot frobnicate\n```\n", "frobnicate"},
		{"```sh\npilot/pilot github frob\n```\n", "github frob"},
		{"Edit `pilot/images/lock` first.\n", "pilot/images/lock"},
		{"[x](specs/999-nope/spec.md)\n", "specs/999-nope"},
		{"Installs kind v0.26.0.\n", "kind"},
		{"token: " + "gh" + "p_" + strings.Repeat("a", 30) + "\n", "token-like"},
	}
	for _, c := range flagged {
		e := check(c.text)
		found := false
		for _, x := range e {
			if strings.Contains(x, c.needle) && strings.Contains(strings.Fields(x)[0], ":") {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want a problem mentioning %q, got %v", c.text, c.needle, e)
		}
	}
}
