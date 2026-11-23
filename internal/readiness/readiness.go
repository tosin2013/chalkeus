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

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/gate"
	"github.com/tosin2013/chalkeus/internal/slice"
)

// Deps are readiness's outside world, injected so tests need no network.
type Deps struct {
	Src gate.StatusSource
	// Clone makes a depth-1 clone of repo@branch and returns its dir and commit.
	Clone func(repo, branch string) (dir, commit string, err error)
	// OpenPRHeads lists the head branches of the target repository's open pull requests.
	OpenPRHeads func(repo string) ([]string, error)
	// AllRules is `./chalkeus forge doctor`: evaluate every rule and record every failure instead of stopping
	// at the first (the decision is still refuse if any rule fails).
	AllRules bool
	// SkipHidden is the test author's readiness (018 `author start`): an empty companion folder
	// doesn't refuse; HiddenFiles reports how many files are already there.
	SkipHidden bool
	// Slice is the slice to check (026 `run --slice`); "" checks the whole item.
	Slice string
	// PriorSlices is the slicing local slice runs of this item recorded (slice id -> phases and
	// the slice count then), for SLICE_MAP_CHANGED. Nil when there are none.
	PriorSlices map[string]PriorSlice
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
	// 026: the slice a slice run builds, whether it is the item's last, the companion folders the
	// grader overlays (this slice or the whole item) and the earlier slices' folders (G8), and the
	// item-level acceptance checks the spec declares.
	Slice      *SliceInfo `json:"slice,omitempty"`
	SlicesLast bool       `json:"slices_last,omitempty"`
	Overlay    []string   `json:"overlay,omitempty"`
	Earlier    []string   `json:"earlier,omitempty"`
	Acceptance []string   `json:"acceptance,omitempty"`
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
	failed := false
	// fail records a refusal. It reports whether to stop: always for a run (fail closed at the
	// first problem), never with AllRules (doctor), which collects every problem.
	fail := func(code, detail string) bool {
		failed = true
		if d.AllRules {
			res.Reasons = append(res.Reasons, code)
		} else {
			res.Reasons = append([]string{code}, res.Reasons...)
		}
		rule(code, false, detail)
		return !d.AllRules
	}
	finish := func() (Result, error) {
		if !failed {
			res.Decision, res.Reasons = "allow", []string{"OK"}
		}
		return res, writeRecord(outDir, seq, item, res)
	}
	skip := func(id, why string) { res.Rules = append(res.Rules, Rule{ID: id, Result: "skipped", Detail: why}) }
	if !itemIDRE.MatchString(item) {
		fail("ITEM_NOT_ON_ROADMAP", fmt.Sprintf("item %q is not a roadmap id (NNN)", item))
		return finish()
	}

	// 0. The gate, on a fresh clone of the target's branch.
	dir, commit, err := d.Clone(t.Repo, t.Branch)
	if err != nil {
		res.Gate = gate.Refused(d.Src, item, item, seq, t.Repo, "UNEVALUABLE:clone_failed")
		res.Reasons = res.Gate.Reasons
		rule("gate", false, "clone of "+t.Repo+"@"+t.Branch+" failed")
		failed = true
		return finish()
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
		failed = true
		res.Reasons = append(res.Reasons, g.Reasons...)
		rule("gate", false, strings.Join(g.Reasons, ","))
		for _, r := range g.Reasons {
			if r == "ITEM_MISSING" || r == "INACTIVE" {
				fail("ITEM_NOT_ON_ROADMAP", "item "+item+" is not an active item of "+t.Repo+"'s roadmap")
				return finish()
			}
		}
		if !d.AllRules {
			return finish()
		}
	} else {
		rule("gate", true, "allow")
	}
	doc, itemDoc := statusItem(stdout, item)
	var heads []string
	var headsErr error
	headsRead := false
	openHeads := func() ([]string, error) {
		if !headsRead {
			heads, headsErr = d.OpenPRHeads(t.Repo)
			headsRead = true
		}
		return heads, headsErr
	}

	// 2. Spec and tasks.
	specDir, _ := itemDoc["spec_dir"].(string)
	specDir = strings.TrimSuffix(strings.TrimPrefix(specDir, "./"), "/")
	var spec []byte
	specOK := false
	switch specDir {
	case "":
		if fail("ITEM_SPEC_MISSING", "item "+item+" has no Spec dir on the roadmap") {
			return finish()
		}
	default:
		res.SpecDir = specDir
		res.Title, _ = itemDoc["title"].(string)
		if spec, err = os.ReadFile(filepath.Join(dir, specDir, "spec.md")); err != nil {
			if fail("ITEM_SPEC_MISSING", specDir+"/spec.md is missing") {
				return finish()
			}
		} else {
			specOK = true
			rule("spec", true, specDir+"/spec.md")
			res.SpecSHA256 = sha256Hex(spec)
		}
		if _, err := os.Stat(filepath.Join(dir, specDir, "tasks.md")); err != nil {
			if fail("ITEM_TASKS_MISSING", specDir+"/tasks.md is missing") {
				return finish()
			}
		} else {
			rule("tasks", true, specDir+"/tasks.md")
		}
	}

	// 2b. Slices (026, contracts/slices.md): resolved here because the allowed paths and the
	// hidden tests below depend on the slice.
	ss := readSlices(itemDoc)
	var sl *slice.Slice
	var earlier []slice.Slice
	switch {
	case d.Slice != "":
		switch found, ok := slice.Find(ss.slices, d.Slice); {
		case !ss.known:
			if fail("SLICE_UNEVALUABLE", "spec status reports no tasks.phases for item "+item+" (needs spec-status 0.4.0+ and a readable tasks.md)") {
				return finish()
			}
		case ss.err != nil:
			if fail("SLICE_UNKNOWN", ss.err.Error()) {
				return finish()
			}
		case !ok:
			if fail("SLICE_UNKNOWN", fmt.Sprintf("%q is not a slice of item %s; its slices are %s (run the whole item without --slice)", d.Slice, item, strings.Join(slice.IDs(ss.slices), ", "))) {
				return finish()
			}
		default:
			sl, earlier = &found, slice.Earlier(ss.slices, found)
			res.SlicesLast = slice.Last(ss.slices, found)
			info := &SliceInfo{ID: found.ID, Index: found.Index, Of: len(ss.slices), Phases: found.Phases,
				Earlier: slice.IDs(earlier), EarlierPhases: [][]string{}, DefaultRevision: commit, All: slice.IDs(ss.slices)}
			for _, e := range earlier {
				info.EarlierPhases = append(info.EarlierPhases, e.Phases)
			}
			res.Slice = info
			ok := true
			if why := mapChanged(d.PriorSlices, ss.slices); why != "" {
				ok = false
				if fail("SLICE_MAP_CHANGED", why+"; the pilot never re-maps slices: fix tasks.md or plan a new item") {
					return finish()
				}
			}
			hs, herr := openHeads()
			open := openSlices(hs, item, slice.IDs(earlier))
			var notMerged []string
			for _, e := range earlier {
				if slice.State(e) != slice.Merged || open[e.ID] {
					notMerged = append(notMerged, e.ID)
				}
			}
			if herr != nil {
				ok = false
				if fail("SLICE_EARLIER_OPEN", "could not list open PRs on "+t.Repo+": "+herr.Error()) {
					return finish()
				}
			} else if len(notMerged) > 0 {
				ok = false
				if fail("SLICE_EARLIER_OPEN", fmt.Sprintf("earlier slice(s) %s of item %s are not merged on %s@%s; build and merge them first", strings.Join(notMerged, ", "), item, t.Repo, short(commit))) {
					return finish()
				}
			}
			if st := slice.State(found); st != slice.Untouched {
				ok = false
				if fail("SLICE_STARTED", fmt.Sprintf("slice %s already has %d of %d tasks ticked on %s@%s", found.ID, found.Checked, found.Total, t.Repo, short(commit))) {
					return finish()
				}
			}
			if ok {
				rule("slice", true, fmt.Sprintf("%s (%d of %d); earlier %s merged", found.ID, found.Index+1, len(ss.slices), strings.Join(slice.IDs(earlier), ", ")))
			}
		}
	case ss.known && ss.err == nil && slice.Sliced(ss.slices):
		var begun []string
		for _, x := range ss.slices {
			if slice.State(x) != slice.Untouched {
				begun = append(begun, x.ID)
			}
		}
		if len(begun) > 0 {
			if fail("ITEM_PARTLY_SLICED", fmt.Sprintf("item %s has slices %s, and %s already have ticked tasks; run the remaining slices with --slice", item, strings.Join(slice.IDs(ss.slices), ", "), strings.Join(begun, ", "))) {
				return finish()
			}
		}
	}

	// 3. Open questions: spec markers, and unresolved roadmap open questions naming the item.
	roadmap := ".specify/memory/roadmap.md"
	if src, ok := doc["source"].(map[string]any); ok {
		if p, ok := src["path"].(string); ok && p != "" {
			roadmap = p
		}
	}
	questionsOK := true
	if specOK && strings.Contains(string(spec), clarifyMarker) {
		questionsOK = false
		if fail("ITEM_OPEN_QUESTIONS", specDir+"/spec.md still has [NEEDS CLARIFICATION] markers") {
			return finish()
		}
	}
	if q := openQuestionsNaming(filepath.Join(dir, roadmap), item); q != "" {
		questionsOK = false
		if fail("ITEM_OPEN_QUESTIONS", "roadmap open question names "+item+": "+q) {
			return finish()
		}
	}
	if questionsOK {
		rule("open_questions", true, "none")
	}
	if specOK {
		res.Acceptance = Acceptance(string(spec))
	}

	// 4. Allowed paths.
	if !specOK {
		skip("allowed_paths", "needs the spec")
	} else if allowed, source, err := allowedFor(string(spec), specDir, sl, otherSpecDirs(doc, item)); err != nil {
		r := err.(*Refusal)
		if fail(r.Code, r.Detail) {
			return finish()
		}
	} else {
		if res.Slice != nil {
			res.Slice.AllowedPathsSource = source
		}
		res.AllowedPaths = allowed
		res.TestsMayChange = TestsMayChange(string(spec))
		rule("allowed_paths", true, strings.Join(allowed, ", "))
	}

	// 5. Hidden tests in the companion repository (read with the operator's own access).
	if cdir, ccommit, err := d.Clone(t.CompanionRepo, ""); err != nil {
		if fail("ITEM_NO_HIDDEN_TESTS", "companion repository "+t.CompanionRepo+" could not be read with your git access") {
			return finish()
		}
	} else {
		defer func() { _ = os.RemoveAll(cdir) }()
		itemDir := filepath.Join(cdir, filepath.FromSlash(t.CompanionItemDir(item)))
		overlay, earlierDirs, stray := overlayDirs(t, item, ss, sl, earlier, companionLayout(itemDir))
		res.Overlay, res.Earlier = overlay, earlierDirs
		n := countDirs(cdir, overlay)
		var emptyEarlier []string
		for i, e := range earlierDirs {
			if countDirs(cdir, []string{e}) == 0 {
				emptyEarlier = append(emptyEarlier, earlier[i].ID)
			}
		}
		res.CompanionCommit, res.HiddenFiles = ccommit, n
		where := strings.Join(overlay, ", ")
		switch {
		case len(stray) > 0 && !d.SkipHidden:
			if fail("ITEM_NO_HIDDEN_TESTS", t.CompanionRepo+": "+t.CompanionItemDir(item)+"/ mixes slice folders with other entries ("+strings.Join(stray, ", ")+"); keep only one folder per slice") {
				return finish()
			}
		case d.SkipHidden:
			rule("hidden_tests", true, fmt.Sprintf("skipped (test author): %d file(s) already at %s", n, where))
		case n == 0:
			if fail("ITEM_NO_HIDDEN_TESTS", t.CompanionRepo+" has no files under "+where+"/") {
				return finish()
			}
		case len(emptyEarlier) > 0:
			if fail("ITEM_NO_HIDDEN_TESTS", "earlier slice(s) "+strings.Join(emptyEarlier, ", ")+" have no hidden tests in "+t.CompanionRepo+"; the grader re-runs them (G8)") {
				return finish()
			}
		default:
			rule("hidden_tests", true, fmt.Sprintf("%d file(s) at %s@%s", n, t.CompanionRepo, short(ccommit)))
		}
		// The one realistic way a worker could learn hidden tests is finding them in the target
		// repository (018, roadmap C-14): refuse if any hidden test's content is already there.
		if leaked := hiddenInTarget(itemDir, dir); len(leaked) > 0 {
			if fail("HIDDEN_TESTS_IN_TARGET", "the target already contains hidden test content: "+strings.Join(leaked, ", ")) {
				return finish()
			}
		}
		res.Mapping = mappingReport(itemDir, res.SpecSHA256)
	}

	// 6. Input size: the snapshot the pods receive.
	if size, err := treeBytes(dir); err != nil {
		if fail("INPUT_TOO_LARGE", "could not measure the snapshot: "+err.Error()) {
			return finish()
		}
	} else {
		res.InputBytes = size
		if limit := int64(t.MaxInputMB) << 20; size > limit {
			if fail("INPUT_TOO_LARGE", fmt.Sprintf("snapshot is %d MB, over FORGE_MAX_INPUT_MB=%d", size>>20, t.MaxInputMB)) {
				return finish()
			}
		} else {
			rule("input_size", true, fmt.Sprintf("%d bytes", size))
		}
	}

	// 7. No open pilot PR for this item (or, 026, a slice that blocks this run) on the target.
	hs, err := openHeads()
	openPR := ""
	if err != nil {
		openPR = "could not list open PRs on " + t.Repo + ": " + err.Error()
	} else if blocking := openHeadsFor(hs, item, sl, earlier); len(blocking) > 0 {
		openPR = "an open PR from " + blocking[0] + " exists; merge or close it first"
	}
	if openPR != "" {
		fail("OPEN_ITEM_PR", openPR)
	} else {
		rule("open_pr", true, "none")
	}
	return finish()
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

// allowedFor is the run's allowed paths: the item's, or for a slice run the slice's own list when
// the spec has one (026). source is "item" or "slice".
func allowedFor(specText, specDir string, sl *slice.Slice, others []string) ([]string, string, error) {
	if sl == nil {
		g, err := AllowedPaths(specText, specDir, others)
		return g, "item", err
	}
	return SliceAllowedPaths(specText, specDir, sl.ID, others)
}
