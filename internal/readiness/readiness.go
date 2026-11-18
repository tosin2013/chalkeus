package readiness

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/gate"
)

// Deps are readiness's outside world, injected so tests need no network.
type Deps struct {
	Src gate.StatusSource
	// Clone makes a depth-1 clone of repo@branch and returns its dir and commit.
	Clone func(repo, branch string) (dir, commit string, err error)
	// OpenPRHeads lists the head branches of the target repository's open pull requests.
	OpenPRHeads func(repo string) ([]string, error)
	// SkipHidden is the test author's readiness (018 `author start`): an empty companion folder
	// doesn't refuse; HiddenFiles reports how many files are already there.
	SkipHidden bool
}

// Rule is one readiness rule's outcome, recorded in gate/<seq>-<NNN>.readiness.json.
type Rule struct {
	ID     string `json:"id"`
	Result string `json:"result"` // pass | fail | skipped
	Detail string `json:"detail"`
}

// Result is the readiness decision. On allow it carries what the run needs next.
type Result struct {
	Decision        string         `json:"decision"` // allow | refuse
	Reasons         []string       `json:"reasons"`
	Rules           []Rule         `json:"rules"`
	Gate            gate.Decision  `json:"-"`
	Commit          string         `json:"commit,omitempty"`
	SpecDir         string         `json:"spec_dir,omitempty"`
	Title           string         `json:"title,omitempty"` // the item's roadmap title
	AllowedPaths    []string       `json:"allowed_paths,omitempty"`
	TestsMayChange  []string       `json:"tests_may_change,omitempty"`
	SpecSHA256      string         `json:"spec_sha256,omitempty"`
	Mapping         map[string]any `json:"mapping,omitempty"` // the test author's coverage map (018): present, uncovered, stale
	CompanionCommit string         `json:"companion_commit,omitempty"`
	HiddenFiles     int            `json:"hidden_files,omitempty"`
	InputBytes      int64          `json:"input_bytes,omitempty"`
}

var (
	itemIDRE      = regexp.MustCompile(`^[0-9]{3,4}$`)
	clarifyMarker = "[NEEDS CLARIFICATION"
	openQHeading  = regexp.MustCompile(`^##\s+Open Questions\s*$`)
)

// cachedSource answers the gate from one status run, so the tool runs once per check.
type cachedSource struct {
	exit    *int
	stdout  string
	version string
}

func (c cachedSource) Status(string) (*int, string) { return c.exit, c.stdout }
func (c cachedSource) Version() string              { return c.version }

// Check runs the readiness rules for item of target, in contract order; the first failing rule
// refuses. seq numbers the gate record; outDir (may be "") receives the gate decision and
// readiness record.
func Check(d Deps, t config.Target, item string, seq int, outDir string) (Result, error) {
	res := Result{Decision: "refuse"}
	rule := func(id string, ok bool, detail string) bool {
		r := "pass"
		if !ok {
			r = "fail"
		}
		res.Rules = append(res.Rules, Rule{ID: id, Result: r, Detail: detail})
		return ok
	}
	done := func(code, detail string) (Result, error) {
		if code != "" {
			res.Reasons = append([]string{code}, res.Reasons...)
			rule(code, false, detail)
		}
		return res, writeRecord(outDir, seq, item, res)
	}
	if !itemIDRE.MatchString(item) {
		return done("ITEM_NOT_ON_ROADMAP", fmt.Sprintf("item %q is not a roadmap id (NNN)", item))
	}

	// 0. The gate, on a fresh clone of the target's branch.
	dir, commit, err := d.Clone(t.Repo, t.Branch)
	if err != nil {
		res.Gate = gate.Refused(d.Src, item, item, seq, t.Repo, "UNEVALUABLE:clone_failed")
		res.Reasons = res.Gate.Reasons
		rule("gate", false, "clone of "+t.Repo+"@"+t.Branch+" failed")
		return res, writeRecord(outDir, seq, item, res)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	res.Commit = commit
	exit, stdout := d.Src.Status(dir)
	src := cachedSource{exit: exit, stdout: stdout, version: d.Src.Version()}
	g, err := gate.Evaluate(src, item, item, dir, seq, outDir, t.Repo, &commit)
	if err != nil {
		return res, err
	}
	res.Gate = g
	if g.Decision != "allow" {
		res.Reasons = append(res.Reasons, g.Reasons...)
		rule("gate", false, strings.Join(g.Reasons, ","))
		for _, r := range g.Reasons {
			if r == "ITEM_MISSING" || r == "INACTIVE" {
				return done("ITEM_NOT_ON_ROADMAP", "item "+item+" is not an active item of "+t.Repo+"'s roadmap")
			}
		}
		return done("", "")
	}
	rule("gate", true, "allow")
	doc, itemDoc := statusItem(stdout, item)

	// 2. Spec and tasks.
	specDir, _ := itemDoc["spec_dir"].(string)
	specDir = strings.TrimSuffix(strings.TrimPrefix(specDir, "./"), "/")
	if specDir == "" {
		return done("ITEM_SPEC_MISSING", "item "+item+" has no Spec dir on the roadmap")
	}
	res.SpecDir = specDir
	res.Title, _ = itemDoc["title"].(string)
	spec, err := os.ReadFile(filepath.Join(dir, specDir, "spec.md"))
	if err != nil {
		return done("ITEM_SPEC_MISSING", specDir+"/spec.md is missing")
	}
	rule("spec", true, specDir+"/spec.md")
	if _, err := os.Stat(filepath.Join(dir, specDir, "tasks.md")); err != nil {
		return done("ITEM_TASKS_MISSING", specDir+"/tasks.md is missing")
	}
	rule("tasks", true, specDir+"/tasks.md")
	res.SpecSHA256 = sha256Hex(spec)

	// 3. Open questions: spec markers, and unresolved roadmap open questions naming the item.
	if strings.Contains(string(spec), clarifyMarker) {
		return done("ITEM_OPEN_QUESTIONS", specDir+"/spec.md still has [NEEDS CLARIFICATION] markers")
	}
	roadmap := ".specify/memory/roadmap.md"
	if src, ok := doc["source"].(map[string]any); ok {
		if p, ok := src["path"].(string); ok && p != "" {
			roadmap = p
		}
	}
	if q := openQuestionsNaming(filepath.Join(dir, roadmap), item); q != "" {
		return done("ITEM_OPEN_QUESTIONS", "roadmap open question names "+item+": "+q)
	}
	rule("open_questions", true, "none")

	// 4. Allowed paths.
	allowed, err := AllowedPaths(string(spec), specDir, otherSpecDirs(doc, item))
	if err != nil {
		r := err.(*Refusal)
		return done(r.Code, r.Detail)
	}
	res.AllowedPaths = allowed
	res.TestsMayChange = TestsMayChange(string(spec))
	rule("allowed_paths", true, strings.Join(allowed, ", "))

	// 5. Hidden tests in the companion repository (read with the operator's own access).
	cdir, ccommit, err := d.Clone(t.CompanionRepo, "")
	if err != nil {
		return done("ITEM_NO_HIDDEN_TESTS", "companion repository "+t.CompanionRepo+" could not be read with your git access")
	}
	defer func() { _ = os.RemoveAll(cdir) }()
	itemDir := filepath.Join(cdir, filepath.FromSlash(t.CompanionItemDir(item)))
	n := countFiles(itemDir)
	res.CompanionCommit, res.HiddenFiles = ccommit, n
	if d.SkipHidden {
		rule("hidden_tests", true, fmt.Sprintf("skipped (test author): %d file(s) already at %s", n, t.CompanionItemDir(item)))
	} else {
		if n == 0 {
			return done("ITEM_NO_HIDDEN_TESTS", t.CompanionRepo+" has no files under "+t.CompanionItemDir(item)+"/")
		}
		rule("hidden_tests", true, fmt.Sprintf("%d file(s) at %s@%s", n, t.CompanionRepo, short(ccommit)))
	}
	// The one realistic way a worker could learn hidden tests is finding them in the target
	// repository (018, roadmap C-14): refuse if any hidden test's content is already there.
	if leaked := hiddenInTarget(itemDir, dir); len(leaked) > 0 {
		return done("HIDDEN_TESTS_IN_TARGET", "the target already contains hidden test content: "+strings.Join(leaked, ", "))
	}
	res.Mapping = mappingReport(itemDir, res.SpecSHA256)

	// 6. Input size: the snapshot the pods receive.
	size, err := treeBytes(dir)
	if err != nil {
		return done("INPUT_TOO_LARGE", "could not measure the snapshot: "+err.Error())
	}
	res.InputBytes = size
	if limit := int64(t.MaxInputMB) << 20; size > limit {
		return done("INPUT_TOO_LARGE", fmt.Sprintf("snapshot is %d MB, over PILOT_MAX_INPUT_MB=%d", size>>20, t.MaxInputMB))
	}
	rule("input_size", true, fmt.Sprintf("%d bytes", size))

	// 7. No open pilot PR for this item on the target.
	heads, err := d.OpenPRHeads(t.Repo)
	if err != nil {
		return done("OPEN_ITEM_PR", "could not list open PRs on "+t.Repo+": "+err.Error())
	}
	itemHead := regexp.MustCompile(`^pilot/[^/]+/` + item + `$`)
	for _, h := range heads {
		if itemHead.MatchString(h) {
			return done("OPEN_ITEM_PR", "an open PR from "+h+" exists; merge or close it first")
		}
	}
	rule("open_pr", true, "none")

	res.Decision, res.Reasons = "allow", []string{"OK"}
	return res, writeRecord(outDir, seq, item, res)
}

func statusItem(stdout, item string) (map[string]any, map[string]any) {
	var doc map[string]any
	dec := json.NewDecoder(strings.NewReader(stdout))
	dec.UseNumber()
	_ = dec.Decode(&doc)
	items, _ := doc["items"].([]any)
	for _, x := range items {
		if m, ok := x.(map[string]any); ok && m["id"] == item {
			return doc, m
		}
	}
	return doc, map[string]any{}
}

func otherSpecDirs(doc map[string]any, item string) []string {
	var out []string
	items, _ := doc["items"].([]any)
	for _, x := range items {
		m, ok := x.(map[string]any)
		if !ok || m["id"] == item {
			continue
		}
		if s, ok := m["spec_dir"].(string); ok && s != "" {
			out = append(out, strings.TrimSuffix(strings.TrimPrefix(s, "./"), "/"))
		}
	}
	return out
}

// openQuestionsNaming returns the first unresolved open question that names item: a list
// entry under "## Open Questions" containing the id as a word, not struck through (~~) and not
// marked resolved.
func openQuestionsNaming(roadmap, item string) string {
	raw, err := os.ReadFile(roadmap)
	if err != nil {
		return ""
	}
	word := regexp.MustCompile(`\b` + item + `\b`)
	in := false
	var entry []string
	check := func() string {
		e := strings.Join(entry, " ")
		entry = nil
		low := strings.ToLower(e)
		if e == "" || strings.HasPrefix(strings.TrimSpace(e), "- ~~") || strings.Contains(low, "resolved") || !word.MatchString(e) {
			return ""
		}
		return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(e), "- "))
	}
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case openQHeading.MatchString(line):
			in = true
		case in && strings.HasPrefix(line, "## "):
			if q := check(); q != "" {
				return q
			}
			in = false
		case in && strings.HasPrefix(line, "- "):
			if q := check(); q != "" {
				return q
			}
			entry = []string{line}
		case in && len(entry) > 0:
			entry = append(entry, strings.TrimSpace(line))
		}
	}
	if in {
		return check()
	}
	return ""
}

// MapFile is the test author's coverage map (018, grader.MapFile): not a hidden test.
const MapFile = ".acceptance-map.json"

func countFiles(dir string) int {
	n := 0
	_ = filepath.WalkDir(dir, func(_ string, e os.DirEntry, err error) error {
		if err == nil && !e.IsDir() && e.Name() != MapFile {
			n++
		}
		return nil
	})
	return n
}

// treeBytes is the size of the HEAD tree as a tar, which is what the snapshot carries.
func treeBytes(dir string) (int64, error) {
	cmd := exec.Command("git", "-C", dir, "archive", "--format=tar", "HEAD")
	var errb bytes.Buffer
	cmd.Stderr = &errb
	out, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	var n int64
	buf := make([]byte, 1<<16)
	for {
		k, rerr := out.Read(buf)
		n += int64(k)
		if rerr != nil {
			break
		}
	}
	if err := cmd.Wait(); err != nil {
		return 0, fmt.Errorf("git archive: %s", strings.TrimSpace(errb.String()))
	}
	return n, nil
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func writeRecord(outDir string, seq int, item string, res Result) error {
	if outDir == "" {
		return nil
	}
	if seq < 1 {
		seq = 1
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	return evidence.WriteJSON(filepath.Join(outDir, fmt.Sprintf("%d-%s.readiness.json", seq, item)), res)
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// hiddenInTarget lists hidden-test files (map excluded) whose content appears in any file of the
// target checkout (.git excluded).
func hiddenInTarget(itemDir, targetDir string) []string {
	hidden := map[string]string{}
	_ = filepath.WalkDir(itemDir, func(p string, e os.DirEntry, err error) error {
		if err == nil && e.Type().IsRegular() && e.Name() != MapFile {
			if raw, err := os.ReadFile(p); err == nil && len(raw) > 0 {
				rel, _ := filepath.Rel(itemDir, p)
				hidden[sha256Hex(raw)] = filepath.ToSlash(rel)
			}
		}
		return nil
	})
	if len(hidden) == 0 {
		return nil
	}
	var leaked []string
	_ = filepath.WalkDir(targetDir, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if e.IsDir() && e.Name() == ".git" {
			return filepath.SkipDir
		}
		if e.Type().IsRegular() {
			if raw, err := os.ReadFile(p); err == nil {
				if name, ok := hidden[sha256Hex(raw)]; ok {
					rel, _ := filepath.Rel(targetDir, p)
					leaked = append(leaked, name+" (as "+filepath.ToSlash(rel)+")")
				}
			}
		}
		return nil
	})
	sort.Strings(leaked)
	return leaked
}

// mappingReport reads the test author's coverage map (018), if any: whether it is present, the
// uncovered criteria it lists, and whether it was written for another version of the spec. It
// never refuses.
func mappingReport(itemDir, specSHA string) map[string]any {
	raw, err := os.ReadFile(filepath.Join(itemDir, MapFile))
	if err != nil {
		return map[string]any{"present": false}
	}
	var m struct {
		SpecSHA256 string   `json:"spec_sha256"`
		Uncovered  []string `json:"uncovered"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return map[string]any{"present": true, "valid": false}
	}
	unc := make([]any, 0, len(m.Uncovered))
	for _, u := range m.Uncovered {
		unc = append(unc, u)
	}
	return map[string]any{"present": true, "valid": true, "uncovered": unc, "stale": m.SpecSHA256 != specSHA}
}
