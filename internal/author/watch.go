package author

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/prompt"
)

// Candidate is a roadmap item the watcher may draft for: active, startable, every dependency
// satisfied, in roadmap order (024 R2).
type Candidate struct{ ID, Title string }

// Candidates reads `spec status --json` and returns the items worth checking.
func Candidates(statusJSON string) ([]Candidate, error) {
	var doc struct {
		Items []struct {
			ID           string `json:"id"`
			Title        string `json:"title"`
			Status       string `json:"status"`
			Active       bool   `json:"active"`
			Dependencies []struct {
				Satisfied bool   `json:"satisfied"`
				State     string `json:"state"`
			} `json:"dependencies"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(statusJSON), &doc); err != nil {
		return nil, fmt.Errorf("the target's roadmap could not be read: %v", err)
	}
	var out []Candidate
	for _, it := range doc.Items {
		if !it.Active || (it.Status != "planned" && it.Status != "specced" && it.Status != "in-progress") {
			continue
		}
		ok := true
		for _, d := range it.Dependencies {
			ok = ok && (d.Satisfied || d.State == "satisfied")
		}
		if ok {
			out = append(out, Candidate{ID: it.ID, Title: it.Title})
		}
	}
	return out, nil
}

// WatchEnv is what `pilot author watch` needs (024 FR-002).
type WatchEnv struct {
	Env
	Status   func() (string, error) // spec status --json of the target's branch (a fresh clone)
	Job      func(ctx context.Context, d *Draft, attempt int, prompt string) JobRecord
	Baseline Baseline
	// Only limits a pass to one item ("" = every candidate); step mode (021) drafts one item.
	Only string
}

// Outcome of one watcher pass.
const (
	PassIdle    = "idle"    // nothing eligible
	PassOpened  = "opened"  // a companion PR opened
	PassFailed  = "failed"  // a draft failed after its attempts
	PassStopped = "stopped" // cancelled
)

// Pass is one poll: it drafts the first eligible item, if any (one at a time, FR-002).
func Pass(ctx context.Context, w WatchEnv) (string, error) {
	js, err := w.Status()
	if err != nil {
		return "", err
	}
	cands, err := Candidates(js)
	if err != nil {
		return "", err
	}
	for _, c := range cands {
		if w.Only != "" && c.ID != w.Only {
			continue
		}
		if ctx.Err() != nil {
			return PassStopped, nil
		}
		why, specSHA, err := w.skip(ctx, c.ID)
		if err != nil {
			return "", err
		}
		if why != "" {
			fmt.Fprintf(w.Out, "skip %s: %s\n", c.ID, why)
			continue
		}
		return w.draft(ctx, c.ID, specSHA)
	}
	return PassIdle, nil
}

// skip says why an item isn't eligible ("" if it is) (024 R2).
func (w WatchEnv) skip(ctx context.Context, item string) (string, string, error) {
	res, err := w.Ready(w.Target, item)
	if err != nil {
		return "", "", err
	}
	if res.Decision != "allow" {
		if len(res.Reasons) > 0 {
			return "not ready (" + res.Reasons[0] + ")", "", nil
		}
		return "not ready", "", nil
	}
	if res.HiddenFiles > 0 {
		return "hidden tests already merged", "", nil
	}
	id, err := w.Identity(ctx, w.Target)
	if err != nil {
		return "", "", err
	}
	if open, err := w.OpenAuthorPRs(ctx, id, w.Target.CompanionRepo, branchPrefix(item)); err != nil {
		return "", "", err
	} else if len(open) > 0 {
		return "a draft pull request is open (" + open[0] + ")", "", nil
	}
	if prev := w.previousAuto(item, res.SpecSHA256); prev != "" {
		return "already drafted automatically for this spec version (" + prev + "); change the spec or draft by hand", "", nil
	}
	return "", res.SpecSHA256, nil
}

// AutoDrafted reports whether an automatic draft of this item for this spec version already got
// as far as a check (021: step mode then stops asking for a redraft).
func AutoDrafted(evidenceRoot, target, item, specSHA string) bool {
	w := WatchEnv{Env: Env{EvidenceRoot: evidenceRoot, Target: config.Target{Name: target}}}
	return w.previousAuto(item, specSHA) != ""
}

// previousAuto is the id of an earlier automatic draft of this item for the same spec that got as
// far as a check (or a PR), or "". A draft that never produced files (an infrastructure failure,
// such as a Job that couldn't start) doesn't count, so the next pass tries again.
func (w WatchEnv) previousAuto(item, specSHA string) string {
	entries, _ := os.ReadDir(Root(w.EvidenceRoot))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		d, err := LoadDraft(Root(w.EvidenceRoot), e.Name(), "", "")
		if err == nil && d.Trigger == "auto" && d.Target == w.Target.Name && d.Item == item && d.SpecSHA256 == specSHA &&
			(len(d.Checks) > 0 || d.State == "submitted") {
			return d.ID
		}
	}
	return ""
}

// draft runs one automatic draft: start, author Job(s), check, and submit only after a passing
// check (SC-003). A failing check is retried with feedback while attempts remain (R5).
func (w WatchEnv) draft(ctx context.Context, item, specSHA string) (string, error) {
	quiet := w.Env
	quiet.Out = io.Discard // start/check print instructions meant for a human author
	d, err := Start(ctx, quiet, item, false)
	if err != nil {
		return "", err
	}
	d.Trigger, d.WatchDecision = "auto", "eligible: ready except for hidden tests"
	_ = d.Save()
	fmt.Fprintf(w.Out, "draft %s: started %s\n", item, d.ID)
	attempts := w.Target.AuthorAttempts
	if attempts < 1 {
		attempts = 1
	}
	feedbackCode, feedbackOut := "", ""
	for n := 1; n <= attempts; n++ {
		p, err := prompt.BuildAuthor(prompt.Author{Item: item, SpecSHA256: specSHA, Toolchain: w.Target.Toolchain,
			TestPaths: w.Target.TestPaths, FeedbackCode: feedbackCode, FeedbackOutput: feedbackOut})
		if err != nil {
			return "", err
		}
		rec := w.Job(ctx, d, n, p)
		d.Jobs = append(d.Jobs, rec)
		_ = d.Save()
		_ = d.Event("job", fmt.Sprintf("attempt=%d outcome=%s files=%d error=%s", n, rec.Outcome, rec.DraftFiles, rec.Error))
		if rec.Error == "cancelled" || ctx.Err() != nil {
			d.WatchDecision = "stopped: the watcher was interrupted"
			_ = d.Save()
			_ = d.Event("watch_stopped", "")
			fmt.Fprintf(w.Out, "draft %s: stopped, no PR\n", item)
			return PassStopped, nil
		}
		for _, c := range rec.Checks {
			if c.Result != "pass" {
				d.WatchDecision = "refused: isolation check " + c.ID + " failed"
				_ = d.Save()
				fmt.Fprintf(w.Out, "draft %s: isolation check %s failed (%s), no PR\n", item, c.ID, c.Detail)
				return PassFailed, nil
			}
		}
		if rec.DraftFiles == 0 {
			feedbackCode, feedbackOut = "", ""
			fmt.Fprintf(w.Out, "draft %s: attempt %d produced no draft (%s)\n", item, n, rec.Error)
			continue
		}
		var out bytes.Buffer
		chk := w.Env
		chk.Out = &out
		d2, cerr := Check(ctx, chk, item, w.Baseline)
		if d2 != nil {
			d = d2
			d.Trigger, d.Jobs = "auto", append([]JobRecord(nil), d.Jobs...)
		}
		if cerr == nil {
			s := w.Env
			s.Out = io.Discard
			d3, serr := Submit(ctx, s, item)
			if serr != nil {
				return "", serr
			}
			fmt.Fprintf(w.Out, "draft %s: DRAFT_OK, opened %v\n", item, d3.PR["url"])
			return PassOpened, nil
		}
		code := "error"
		if e, ok := cerr.(*Error); ok {
			code = e.Code
		}
		fmt.Fprintf(w.Out, "draft %s: attempt %d %s\n", item, n, code)
		if code != CodeDraftHiddenPasses && code != CodeDraftSuiteFails && code != CodeDraftPathInSpec {
			break
		}
		feedbackCode, feedbackOut = code, d.lastOutput
	}
	d.WatchDecision = "failed after its attempts; waits for a spec change or a manual draft"
	_ = d.Save()
	fmt.Fprintf(w.Out, "draft %s: failed, no PR\n", item)
	return PassFailed, nil
}
