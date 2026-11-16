package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// T036 (contracts/cli.md): the operator guide, the troubleshooting page and the pilot skill
// quote what the pilot prints. Every quoted message of the reference pilot must also be in the
// Go sources, so the docs stay true after the port.
//
// A "message" is a literal chunk (15+ characters, with a space) of a backtick span or code-block
// line in those docs, split at placeholders (<run-id>, {x}, ..., run ids, numbers), that the
// Python reference contains. That filter keeps commands and paths out. Once the reference is
// retired (US5), the frozen list in testdata/wording.txt is used instead.

var (
	wordingDocs = []string{"docs/operator-guide.md", "docs/troubleshooting.md", "docs/github-loop.md", "docs/github-app.md",
		".claude/skills/pilot/SKILL.md"}
	fenceRE     = regexp.MustCompile("(?s)```[^\\n]*\\n(.*?)```")
	spanRE      = regexp.MustCompile("`([^`\\n]+)`")
	holeRE      = regexp.MustCompile(`<[^>]+>|\{[^}]+\}|\.\.\.|…|\d{8}T\d{6}Z|#\d+|\b\d+\b`)
	// Messages of commands not ported yet; each must leave this list when its command lands.
	wordingPending = map[string]string{}
)

func readTree(t *testing.T, dir, ext string, skip func(string) bool) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ext) || (skip != nil && skip(p)) {
			return err
		}
		raw, err := os.ReadFile(p)
		b.Write(raw)
		return err
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return b.String()
}

func docChunks(t *testing.T, root string) []string {
	t.Helper()
	var spans []string
	for _, d := range wordingDocs {
		raw, err := os.ReadFile(filepath.Join(root, d))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		for _, m := range fenceRE.FindAllStringSubmatch(text, -1) {
			for _, l := range strings.Split(m[1], "\n") {
				spans = append(spans, strings.TrimSpace(l))
			}
		}
		for _, m := range spanRE.FindAllStringSubmatch(fenceRE.ReplaceAllString(text, ""), -1) {
			spans = append(spans, m[1])
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, s := range spans {
		for _, c := range holeRE.Split(strings.TrimPrefix(s, "== "), -1) {
			c = strings.Trim(c, " .:;,()'\"")
			if len(c) >= 15 && strings.Contains(c, " ") && !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	sort.Strings(out)
	return out
}

func TestDocsWordingInGoSources(t *testing.T) {
	root := "../.."
	goSrc := readTree(t, filepath.Join(root, "internal"), ".go", func(p string) bool { return strings.HasSuffix(p, "_test.go") })
	frozen := filepath.Join("testdata", "wording.txt")
	all := docChunks(t, root)

	var msgs []string
	if py := readTree(t, filepath.Join(root, "pilot/specfoundry_pilot"), ".py", nil); py != "" {
		for _, c := range all {
			if strings.Contains(py, c) {
				msgs = append(msgs, c)
			}
		}
		want := strings.Join(msgs, "\n") + "\n"
		if raw, _ := os.ReadFile(frozen); string(raw) != want {
			if os.Getenv("UPDATE_WORDING") == "1" {
				if err := os.WriteFile(frozen, []byte(want), 0o644); err != nil {
					t.Fatal(err)
				}
			} else {
				t.Errorf("%s is out of date with the docs; rerun with UPDATE_WORDING=1", frozen)
			}
		}
	} else { // the reference is retired: use the frozen list, and require each entry still quoted
		raw, err := os.ReadFile(frozen)
		if err != nil {
			t.Fatal(err)
		}
		quoted := map[string]bool{}
		for _, c := range all {
			quoted[c] = true
		}
		for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			if !quoted[l] {
				t.Errorf("%s: %q is no longer quoted in the docs; remove it", frozen, l)
			}
			msgs = append(msgs, l)
		}
	}
	for _, m := range msgs {
		in := strings.Contains(goSrc, m)
		why, pending := wordingPending[m]
		switch {
		case !in && !pending:
			t.Errorf("quoted in the docs but missing from the Go pilot: %q", m)
		case in && pending:
			t.Errorf("%q is now in the Go pilot; remove it from wordingPending (%s)", m, why)
		}
	}
	if len(msgs) < 20 {
		t.Errorf("only %d messages found; the extraction is probably broken", len(msgs))
	}
}
