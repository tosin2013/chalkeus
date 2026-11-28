package author

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/prompt"
	"github.com/tosin2013/chalkeus/internal/readiness"
	"github.com/tosin2013/chalkeus/internal/scm"
)

// The test author's eval set (041 US2, data model "Eval case" and "Eval result"). Cases live in
// the private companion under evals/<case>/; results go to the owner's machine, never to committed
// evidence, because they name the drafted tests' outcomes.

// EvalCase is evals/<case>/case.json.
type EvalCase struct {
	ID             string   `json:"id"`
	Target         string   `json:"target"`        // a configured target (its profile: toolchain, test command, model)
	BaseRevision   string   `json:"base_revision"` // a target commit with the spec but not the change
	Item           string   `json:"item"`
	Slice          string   `json:"slice,omitempty"`
	SpecDir        string   `json:"spec_dir"`
	Title          string   `json:"title"`
	AllowedPaths   []string `json:"allowed_paths,omitempty"`
	TestsMayChange []string `json:"tests_may_change,omitempty"`
	ManifestCommit string   `json:"manifest_commit,omitempty"` // 029: the approved manifest change
	ManifestFiles  []string `json:"manifest_files,omitempty"`
	KnownFailure   string   `json:"known_failure"`
	Status         string   `json:"status"` // ready | pending (no reference yet)
	reference      []byte
}

// EvalCaseResult is one case's outcome.
type EvalCaseResult struct {
	ID      string        `json:"id"`
	Outcome string        `json:"outcome"`          // proven | unproven | error | pending
	Reason  string        `json:"reason,omitempty"` // the check's code, or the error
	Files   []FileVerdict `json:"files,omitempty"`
	Draft   string        `json:"draft,omitempty"` // the draft's folder (owner's machine)
	Seconds int           `json:"seconds"`
}

// EvalResult is one eval run (~/.local/state/chalkeus/evals/<timestamp>.json).
type EvalResult struct {
	At           string           `json:"at"`
	Model        string           `json:"model"`
	PromptSHA256 string           `json:"prompt_sha256"`
	ForgeVersion string           `json:"forge_version"`
	Cases        []EvalCaseResult `json:"cases"`
	Score        int              `json:"score"`
	Of           int              `json:"of"`
	Previous     *int             `json:"previous,omitempty"`
	PreviousAt   string           `json:"previous_at,omitempty"`
	Dropped      bool             `json:"dropped"`
	Changes      []string         `json:"changes,omitempty"`
}

// EvalEnv is what `author eval` needs.
type EvalEnv struct {
	StateDir  string   // ~/.local/state/chalkeus/evals
	Companion string   // the companion repository holding evals/
	Host      scm.Host // the companion's git host (config.Target.Host())
	Out       io.Writer
	// For a case's target: its Env (Ready is replaced per case), author Job and baseline.
	Setup func(target string) (Env, func(ctx context.Context, d *Draft, attempt int, prompt string) JobRecord, Baseline, string, error)
	// Busy reports whether the author namespace is in use (a watcher drafting the same item).
	Busy func(ns string) bool
}

// LoadEvalCases reads every evals/<case>/case.json (and reference.patch) from a companion clone.
// only limits it to one case id.
func LoadEvalCases(cdir, only string) ([]EvalCase, error) {
	entries, err := os.ReadDir(filepath.Join(cdir, "evals"))
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("the companion has no evals/ folder; add evals/<case>/case.json (docs/targets.md)")
	}
	if err != nil {
		return nil, err
	}
	var cases []EvalCase
	for _, e := range entries {
		if !e.IsDir() || (only != "" && e.Name() != only) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(cdir, "evals", e.Name(), "case.json"))
		if err != nil {
			return nil, fmt.Errorf("evals/%s: %v", e.Name(), err)
		}
		var c EvalCase
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("evals/%s/case.json: %v", e.Name(), err)
		}
		if c.ID == "" {
			c.ID = e.Name()
		}
		if c.Status == "" {
			c.Status = "ready"
		}
		if c.Status != "ready" && c.Status != "pending" {
			return nil, fmt.Errorf("evals/%s/case.json: status %q (want ready or pending)", e.Name(), c.Status)
		}
		if c.Status == "ready" && (c.Target == "" || c.BaseRevision == "" || c.Item == "" || c.SpecDir == "") {
			return nil, fmt.Errorf("evals/%s/case.json: a ready case needs target, base_revision, item and spec_dir", e.Name())
		}
		if b, err := os.ReadFile(filepath.Join(cdir, "evals", e.Name(), "reference.patch")); err == nil {
			c.reference = b
		} else if c.Status == "ready" {
			return nil, fmt.Errorf("evals/%s: a ready case needs reference.patch", e.Name())
		}
		cases = append(cases, c)
	}
	if only != "" && len(cases) == 0 {
		return nil, fmt.Errorf("no eval case %q in the companion's evals/", only)
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	return cases, nil
}

// caseReadiness is the readiness a case drafts against: fixed by the case, not the target's
// current branch (the item may have been built since).
func caseReadiness(t config.Target, c EvalCase) (readiness.Result, error) {
	dir, _, err := fixture.FreshClone(t.Host(), t.Repo, "", 0)
	if err != nil {
		return readiness.Result{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if _, err := fixture.Git(t.Host(), dir, "checkout", "--quiet", c.BaseRevision); err != nil {
		return readiness.Result{}, fmt.Errorf("case %s: base %s: %v", c.ID, c.BaseRevision, err)
	}
	spec, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(c.SpecDir), "spec.md"))
	if err != nil {
		return readiness.Result{}, fmt.Errorf("case %s: no %s/spec.md at %s", c.ID, c.SpecDir, c.BaseRevision)
	}
	h := sha256.Sum256(spec)
	res := readiness.Result{Decision: "allow", Commit: c.BaseRevision, SpecDir: c.SpecDir, Title: c.Title,
		AllowedPaths: c.AllowedPaths, TestsMayChange: c.TestsMayChange, SpecSHA256: hex.EncodeToString(h[:])}
	if c.ManifestCommit != "" {
		res.ManifestChange = &readiness.ManifestChange{Commit: c.ManifestCommit, Files: c.ManifestFiles}
	}
	return res, nil
}

// PromptSHA256 fingerprints the author prompt template (a fixed input), so a result says which
// prompt it scored.
func PromptSHA256() string {
	p, _ := prompt.BuildAuthor(prompt.Author{Item: "000", SpecSHA256: "eval", Toolchain: "eval"})
	h := sha256.Sum256([]byte(p))
	return hex.EncodeToString(h[:])
}

// Eval is `./chalkeus forge author eval` (041 T018): draft each ready case with the current
// settings, check it (baseline, then the reference check), score, compare with the previous
// result. It never opens a pull request.
func Eval(ctx context.Context, ee EvalEnv, only string) (*EvalResult, error) {
	cdir, _, err := fixture.FreshClone(ee.Host, ee.Companion, "", 1)
	if err != nil {
		return nil, fmt.Errorf("companion repository %s: %v", ee.Companion, err)
	}
	cases, err := LoadEvalCases(cdir, only)
	_ = os.RemoveAll(cdir)
	if err != nil {
		return nil, err
	}
	at := time.Now().UTC().Format("20060102T150405Z")
	runDir := filepath.Join(ee.StateDir, at)
	r := &EvalResult{At: at, PromptSHA256: PromptSHA256(), ForgeVersion: ForgeVersion}
	for _, c := range cases {
		if c.Status != "ready" {
			r.Cases = append(r.Cases, EvalCaseResult{ID: c.ID, Outcome: "pending", Reason: "no reference yet"})
			fmt.Fprintf(ee.Out, "%s: pending (no reference yet)\n", c.ID)
			continue
		}
		started := time.Now()
		cr := ee.runCase(ctx, c, runDir, r)
		cr.Seconds = int(time.Since(started).Seconds())
		r.Cases = append(r.Cases, cr)
		fmt.Fprintf(ee.Out, "%s: %s %s\n", c.ID, cr.Outcome, cr.Reason)
		if ctx.Err() != nil {
			break
		}
	}
	for _, c := range r.Cases {
		if c.Outcome != "pending" {
			r.Of++
		}
		if c.Outcome == "proven" {
			r.Score++
		}
	}
	if prev, prevAt := latestEval(ee.StateDir); prev != nil {
		r.Previous, r.PreviousAt = &prev.Score, prevAt
		r.Dropped = r.Score < prev.Score
		r.Changes = compareEval(prev, r)
	}
	if err := os.MkdirAll(ee.StateDir, 0o700); err != nil {
		return r, err
	}
	if err := evidence.WriteJSON(filepath.Join(ee.StateDir, at+".json"), r); err != nil {
		return r, err
	}
	fmt.Fprintf(ee.Out, "score: %d of %d proven (model %s, prompt %s)\n", r.Score, r.Of, r.Model, short(r.PromptSHA256))
	if r.Previous != nil {
		fmt.Fprintf(ee.Out, "previous (%s): %d\n", r.PreviousAt, *r.Previous)
		for _, ch := range r.Changes {
			fmt.Fprintf(ee.Out, "  %s\n", ch)
		}
		if r.Dropped {
			fmt.Fprintln(ee.Out, "SCORE DROPPED: keep the change only if you accept that")
		}
	}
	fmt.Fprintf(ee.Out, "result: %s\n", filepath.Join(ee.StateDir, at+".json"))
	return r, nil
}

// runCase drafts and checks one case like `author watch` (attempts with feedback), without
// submitting.
func (ee EvalEnv) runCase(ctx context.Context, c EvalCase, runDir string, r *EvalResult) EvalCaseResult {
	cr := EvalCaseResult{ID: c.ID, Outcome: "error"}
	e, job, baseline, model, err := ee.Setup(c.Target)
	if err != nil {
		cr.Reason = err.Error()
		return cr
	}
	if r.Model == "" {
		r.Model = model
	} else if r.Model != model {
		r.Model = "mixed"
	}
	if ee.Busy != nil && ee.Busy(AuthorNS(e.Target.Name, c.Item)) {
		cr.Reason = fmt.Sprintf("namespace %s is in use (is author watch drafting %s %s?); stop it and run the eval again", AuthorNS(e.Target.Name, c.Item), c.Target, c.Item)
		return cr
	}
	res, err := caseReadiness(e.Target, c)
	if err != nil {
		cr.Reason = err.Error()
		return cr
	}
	e.EvidenceRoot = filepath.Join(runDir, c.ID)
	e.BuildDir = filepath.Join(runDir, c.ID, "build")
	e.Slice = c.Slice
	e.Out = io.Discard
	e.Ready = func(config.Target, string, string) (readiness.Result, error) { return res, nil }
	e.OpenAuthorPRs = func(context.Context, *Identity, string, string) ([]string, error) { return nil, nil }
	ref := c.reference
	e.LoadReference = func(t config.Target, item, slice string) (ReferenceFile, error) {
		return ReferenceFile{Path: "evals/" + c.ID + "/reference.patch", Content: ref}, nil
	}
	d, err := Start(ctx, e, c.Item, false)
	if err != nil {
		cr.Reason = err.Error()
		return cr
	}
	cr.Draft = d.Dir()
	d.Trigger = "eval"
	_ = d.Save()
	attempts := e.Target.AuthorAttempts
	if attempts < 1 {
		attempts = 1
	}
	feedbackCode, feedbackOut := "", ""
	for n := 1; n <= attempts; n++ {
		p, err := prompt.BuildAuthor(prompt.Author{Item: c.Item, SpecSHA256: res.SpecSHA256, Toolchain: e.Target.Toolchain,
			TestPaths: e.Target.TestPaths, FeedbackCode: feedbackCode, FeedbackOutput: feedbackOut})
		if err != nil {
			cr.Reason = err.Error()
			return cr
		}
		rec := job(ctx, d, n, p)
		d.Jobs = append(d.Jobs, rec)
		_ = d.Save()
		if ctx.Err() != nil {
			cr.Reason = "cancelled"
			return cr
		}
		if rec.DraftFiles == 0 {
			cr.Outcome, cr.Reason = "unproven", "no draft ("+rec.Outcome+" "+rec.Error+")"
			feedbackCode, feedbackOut = "", ""
			continue
		}
		var out bytes.Buffer
		chk := e
		chk.Out = &out
		d2, cerr := Check(ctx, chk, c.Item, baseline)
		if d2 != nil {
			d = d2
		}
		if last := lastReference(d); last != nil {
			cr.Files = last.Files
		}
		if cerr == nil {
			cr.Outcome, cr.Reason = "proven", ""
			return cr
		}
		code := "error"
		if ae, ok := cerr.(*Error); ok {
			code = ae.Code
		}
		cr.Outcome, cr.Reason = "unproven", code
		if code == CodeDraftGraderError || code == CodeReferenceBroken || code == CodeReferenceDoesntApply || code == "error" {
			cr.Outcome = "error"
			if code == "error" {
				cr.Reason = cerr.Error()
			}
			return cr
		}
		if !redraftable[code] {
			return cr
		}
		feedbackCode, feedbackOut = code, d.lastOutput
	}
	return cr
}

func lastReference(d *Draft) *ReferenceResult {
	if len(d.Checks) == 0 {
		return nil
	}
	return d.Checks[len(d.Checks)-1].Reference
}

// latestEval is the newest earlier result in dir, or nil.
func latestEval(dir string) (*EvalResult, string) {
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, n := range names {
		raw, err := os.ReadFile(filepath.Join(dir, n))
		var r EvalResult
		if err == nil && json.Unmarshal(raw, &r) == nil {
			return &r, r.At
		}
	}
	return nil, ""
}

// compareEval lists per-case changes from prev to cur.
func compareEval(prev, cur *EvalResult) []string {
	was := map[string]string{}
	for _, c := range prev.Cases {
		was[c.ID] = c.Outcome
	}
	var out []string
	for _, c := range cur.Cases {
		p, ok := was[c.ID]
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("%s: new, %s", c.ID, c.Outcome))
		case p != c.Outcome:
			out = append(out, fmt.Sprintf("%s: %s -> %s", c.ID, p, c.Outcome))
		}
	}
	return out
}
