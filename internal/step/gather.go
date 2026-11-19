package step

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tosin2013/specfoundry/internal/author"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/readiness"
)

// RunInfo is the newest item run of a target and item, as the evidence records it.
type RunInfo struct {
	ID, Dir, State string    // State: run.json "state"
	Watching       bool      // its GitHub watcher is alive
	EndedAt        time.Time // its last event
}

// Source is where the gatherer reads (research R4); every field is replaceable in tests.
type Source struct {
	Status    func() (string, error)                                        // spec status --json of the target branch
	Issues    func(ctx context.Context) (map[string]int, error)             // item -> issue number
	Proposal  func(ctx context.Context, issue int) (ghloop.Proposal, error) // the label and its last adder
	Ready     func(item string) (readiness.Result, error)                   // the author's readiness (SkipHidden)
	TestsPR   func(ctx context.Context, item string) (bool, error)          // an author/<NNN>- PR is open
	Run       func(item string) (RunInfo, bool)                             // the newest item run
	Approvers []string
	AutoDraft bool
	Log       *Log
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
		var labelledAt time.Time
		if n, ok := issues[c.ID]; !ok {
			s.NotWhy = "the item has no issue yet"
		} else if p, err := src.Proposal(ctx, n); err != nil {
			return nil, err
		} else {
			switch {
			case !p.Labelled:
				s.NotWhy = "no proposal label"
			case !ghloop.IsApprover(p.AddedBy, src.Approvers):
				s.NotWhy = "the label was last added by @" + p.AddedBy + ", who is not in PILOT_APPROVERS"
			default:
				s.Proposed = true
			}
			labelledAt = p.AddedAt
		}
		if r, ok := src.Run(c.ID); ok {
			s.RunID = r.ID
			s.Run = runState(r)
			s.RelabelledAfterRun = !labelledAt.IsZero() && labelledAt.After(r.EndedAt)
		}
		if !s.Proposed && s.Run != RunActive {
			out = append(out, s) // nothing else matters for an unproposed item
			continue
		}
		res, err := src.Ready(c.ID)
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
		}
		if src.Log != nil {
			s.DoctorFailedAt = src.Log.DoctorFailedAt(c.ID)
		}
		out = append(out, s)
	}
	return out, nil
}

// runState maps a run's recorded state: a paused run whose watcher is gone will never resume
// (for example its pull request was closed), so it counts as failed, not active.
func runState(r RunInfo) string {
	switch {
	case r.State == "succeeded":
		return RunSucceeded
	case strings.HasPrefix(r.State, "failed"), strings.HasPrefix(r.State, "declined"):
		return RunFailed
	case strings.HasPrefix(r.State, "paused") && !r.Watching:
		return RunFailed
	default:
		return RunActive
	}
}

// NewestRun finds the newest item run of target and item among the run folders under root and
// the extra folders (the decisions log's), reading run.json and events.jsonl.
func NewestRun(root string, extra []string, target, item string, watching func(dir string) bool) (RunInfo, bool) {
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
			Target map[string]any `json:"target"`
		}
		if json.Unmarshal(raw, &r) != nil || r.Kind != "item" || r.Item != item || r.Target["name"] != target {
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
