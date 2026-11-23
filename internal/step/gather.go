package step

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/author"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/readiness"
	"github.com/tosin2013/chalkeus/internal/slice"
)

// RunInfo is the newest item run of a target and item, as the evidence records it.
type RunInfo struct {
	ID, Dir, State string    // State: run.json "state"
	Watching       bool      // its GitHub watcher is alive
	EndedAt        time.Time // its last event
	Waiting        string    // what the run waits for, in words, with the link to act on
}

// Source is where the gatherer reads (research R4); every field is replaceable in tests.
type Source struct {
	Status    func() (string, error)                                        // spec status --json of the target branch
	Issues    func(ctx context.Context) (map[string]int, error)             // item -> issue number
	Proposal  func(ctx context.Context, issue int) (ghloop.Proposal, error) // the label and its last adder
	Ready     func(item, slice string) (readiness.Result, error)            // the author's readiness (SkipHidden), of a slice (026) or the item
	TestsPR   func(ctx context.Context, item string) (bool, error)          // an author/<NNN>- PR is open
	Run       func(item, slice string) (RunInfo, bool)                      // the newest run of the item's slice ("" whole item, "*" any slice)
	Approvers []string
	AutoDraft bool
	// SliceRuns (026, PILOT_SLICE_RUNS=on): work each item slice by slice. Off: every item whole.
	SliceRuns bool
	Repo      string                                 // the target repository, for issue links
	Label     string                                 // the proposal label's name
	Drafted   func(item, slice, specSHA string) bool // an automatic draft (of the slice, 026) already reached its check
	// Planning lists the target's open planning PRs (roadmap or specs/), shown when no item can
	// be proposed: merging one is the owner's next step.
	Planning func(ctx context.Context) ([]string, error)
	Log      *Log
}

// Gather builds the state of every candidate item (active, startable, dependencies satisfied,
// in roadmap order).
func Gather(ctx context.Context, src Source) ([]ItemState, error) {
	js, err := src.Status()
	if err != nil {
		return nil, err
	}
	cands, err := author.Candidates(js)
	if err != nil {
		return nil, err
	}
	issues, err := src.Issues(ctx)
	if err != nil {
		return nil, err
	}
	var out []ItemState
	for _, c := range cands {
		s := ItemState{ID: c.ID, AutoDraft: src.AutoDraft}
		if src.SliceRuns {
			s.Slice = slice.CurrentOf(js, c.ID)
		}
		var labelledAt time.Time
		if n, ok := issues[c.ID]; !ok {
			s.NotWhy = "the item has no issue yet; the next issue sync creates it"
		} else if p, err := src.Proposal(ctx, n); err != nil {
			return nil, err
		} else {
			switch {
			case !p.Labelled:
				s.NotWhy = fmt.Sprintf("to start it, add the label %q to https://github.com/%s/issues/%d", src.Label, src.Repo, n)
			case !ghloop.IsApprover(p.AddedBy, src.Approvers):
				s.NotWhy = fmt.Sprintf("the label was last added by @%s, who is not in PILOT_APPROVERS; an approver must remove and re-add it on https://github.com/%s/issues/%d", p.AddedBy, src.Repo, n)
			default:
				s.Proposed = true
			}
			labelledAt = p.AddedAt
		}
		r, ok := src.Run(c.ID, "*") // 026: any slice's run still active holds the item
		if !ok || runState(r) != RunActive {
			r, ok = src.Run(c.ID, s.Slice)
		}
		if ok {
			s.RunID, s.RunDir, s.RunWaiting = r.ID, r.Dir, r.Waiting
			s.Run = runState(r)
			s.RelabelledAfterRun = !labelledAt.IsZero() && labelledAt.After(r.EndedAt)
		}
		if (!s.Proposed && s.Run != RunActive) || s.Run == RunActive {
			out = append(out, s) // an unproposed item, or a run in flight: nothing else is needed (and no clone every 15 s)
			continue
		}
		res, err := src.Ready(c.ID, s.Slice)
		if err != nil {
			return nil, err
		}
		s.Ready, s.Hidden = res.Decision == "allow", res.HiddenFiles
		if !s.Ready && len(res.Reasons) > 0 {
			s.ReadyWhy = res.Reasons[0]
		}
		s.Commits = res.Commit + ":" + res.CompanionCommit
		if s.Hidden == 0 {
			if s.TestsPR, err = src.TestsPR(ctx, c.ID); err != nil {
				return nil, err
			}
			s.AutoDrafted = src.Drafted != nil && src.Drafted(c.ID, s.Slice, res.SpecSHA256)
		}
		if src.Log != nil {
			s.DoctorFailedAt = src.Log.DoctorFailedAt(c.ID)
		}
		out = append(out, s)
	}
	return out, nil
}

// StaleAfter is how long an `in-progress` run may go without an event, and without a watcher,
// before step mode counts it as crashed (025 T014). The longest quiet stretch of a live run is one
// attempt (the worker deadline, 20 minutes) plus grading, well under an hour.
const StaleAfter = time.Hour

// now is the clock (replaced in tests).
var now = time.Now

// runState maps a run's recorded state:
//   - a paused run whose watcher is gone will never resume (for example its pull request was
//     closed), so it counts as failed, not active;
//   - an in-progress run with no watcher and no event for StaleAfter crashed (its process was
//     killed, or the machine restarted), so it counts as failed too instead of holding the item's
//     slot forever. Either way step mode then waits for the owner to relabel the item.
func runState(r RunInfo) string {
	switch {
	case r.State == "succeeded":
		return RunSucceeded
	case strings.HasPrefix(r.State, "failed"), strings.HasPrefix(r.State, "declined"):
		return RunFailed
	case strings.HasPrefix(r.State, "paused") && !r.Watching:
		return RunFailed
	case r.State == "in-progress" && !r.Watching && !r.EndedAt.IsZero() && now().Sub(r.EndedAt) > StaleAfter:
		return RunFailed
	default:
		return RunActive
	}
}

// NewestRun finds the newest item run of target and item among the run folders under root and
// the extra folders (the decisions log's), reading run.json and events.jsonl.
func NewestRun(root string, extra []string, target, item, slice string, watching func(dir string) bool) (RunInfo, bool) {
	dirs := map[string]bool{}
	if entries, err := os.ReadDir(root); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dirs[filepath.Join(root, e.Name())] = true
			}
		}
	}
	for _, d := range extra {
		dirs[d] = true
	}
	var ids []string
	byID := map[string]string{}
	for d := range dirs {
		raw, err := os.ReadFile(filepath.Join(d, "run.json"))
		if err != nil {
			continue
		}
		var r struct {
			ID     string         `json:"run_id"`
			Kind   string         `json:"kind"`
			Item   string         `json:"item"`
			Slice  string         `json:"slice"`
			Target map[string]any `json:"target"`
		}
		if json.Unmarshal(raw, &r) != nil || r.Kind != "item" || r.Item != item || r.Target["name"] != target {
			continue
		}
		if slice != "*" && r.Slice != slice { // 026: "" is a whole-item run, "*" any run of the item
			continue
		}
		ids, byID[r.ID] = append(ids, r.ID), d
	}
	if len(ids) == 0 {
		return RunInfo{}, false
	}
	sort.Strings(ids)
	id := ids[len(ids)-1]
	d := byID[id]
	var st map[string]any
	raw, _ := os.ReadFile(filepath.Join(d, "run.json"))
	_ = json.Unmarshal(raw, &st)
	ri := RunInfo{ID: id, Dir: d, Watching: watching != nil && watching(d)}
	ri.State, _ = st["state"].(string)
	repo, _ := (st["target"].(map[string]any))["repo"].(string)
	ri.Waiting = waitingFor(d, ri.State, repo)
	if ev, err := os.ReadFile(filepath.Join(d, "events.jsonl")); err == nil {
		lines := strings.Split(strings.TrimSpace(string(ev)), "\n")
		var last struct {
			At string `json:"at"`
		}
		if json.Unmarshal([]byte(lines[len(lines)-1]), &last) == nil {
			ri.EndedAt, _ = time.Parse(time.RFC3339, last.At)
		}
	}
	return ri, true
}

// waitingFor says what a run waits for and where the owner acts: the tracking issue for an
// approval, the PR for a merge (from the run's own tracking.json and prs.json).
func waitingFor(dir, state, repo string) string {
	issue := func() string {
		var t struct {
			Issue any `json:"issue"`
		}
		raw, _ := os.ReadFile(filepath.Join(dir, "github", "tracking.json"))
		if json.Unmarshal(raw, &t) == nil && t.Issue != nil && repo != "" {
			return fmt.Sprintf("https://github.com/%s/issues/%v", repo, t.Issue)
		}
		return "the run's tracking issue"
	}
	switch state {
	case "paused(awaiting-mode)":
		return "waiting for you: reply `/approve mode` on " + issue()
	case "paused(awaiting-retry)":
		return "waiting for you: an attempt failed; reply `/approve retry` (or not) on " + issue()
	case "paused(awaiting-merge)":
		var prs []struct {
			Kind string `json:"kind"`
			URL  string `json:"url"`
		}
		raw, _ := os.ReadFile(filepath.Join(dir, "prs.json"))
		if json.Unmarshal(raw, &prs) == nil && len(prs) > 0 {
			last := prs[len(prs)-1]
			if last.Kind == "status" {
				return "waiting for you: review and merge the status-change PR (it records the item verified): " + last.URL
			}
			return "waiting for you: review and merge the item's PR: " + last.URL
		}
		return "waiting for you to merge the run's PR"
	case "paused(awaiting-publish)":
		return "the run is publishing its PR; nothing needed from you"
	default:
		return "the run is working (build, grade or publish); nothing needed from you yet"
	}
}
