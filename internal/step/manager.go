package step

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tosin2013/specfoundry/internal/evidence"
)

// Entry is one line of decisions.jsonl (FR-006): names, steps and reasons only.
type Entry struct {
	At      string `json:"at"`
	Item    string `json:"item"`
	Action  string `json:"action"`
	Reason  string `json:"reason"`
	Run     string `json:"run,omitempty"`     // the run folder a run step started
	Comment string `json:"comment,omitempty"` // the marker of a comment posted
	Commits string `json:"commits,omitempty"` // target:companion commits, for doctor results
}

// Log is <evidence root>/step/<target>/decisions.jsonl.
type Log struct {
	Path    string
	entries []Entry
}

// OpenLog reads an existing log (or starts an empty one).
func OpenLog(dir string) (*Log, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	l := &Log{Path: filepath.Join(dir, "decisions.jsonl")}
	f, err := os.Open(l.Path)
	if err != nil {
		return l, nil
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e Entry
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			l.entries = append(l.entries, e)
		}
	}
	return l, nil
}

// Add appends an entry.
func (l *Log) Add(e Entry) error {
	e.At = evidence.Now()
	l.entries = append(l.entries, e)
	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	b, _ := json.Marshal(e)
	_, err = f.Write(append(b, '\n'))
	return err
}

// Posted reports whether a comment with this marker was already posted.
func (l *Log) Posted(marker string) bool {
	for _, e := range l.entries {
		if e.Comment == marker {
			return true
		}
	}
	return false
}

// DoctorFailedAt is the commits of the item's last doctor, if it failed ("" otherwise).
func (l *Log) DoctorFailedAt(item string) string {
	at := ""
	for _, e := range l.entries {
		if e.Item == item && e.Action == DoctorFailed {
			at = e.Commits
		}
		if e.Item == item && e.Action == "doctor-ok" {
			at = ""
		}
	}
	return at
}

// RunDirs are the run folders the manager started (so a changed evidence root can't hide them).
func (l *Log) RunDirs() []string {
	var out []string
	for _, e := range l.entries {
		if e.Run != "" {
			out = append(out, e.Run)
		}
	}
	return out
}

// Manager is one target's loop (FR-001). Its actions are delegated (research R3).
type Manager struct {
	Target string
	Src    Source
	Log    *Log
	Dir    string // <evidence root>/step/<target>: subprocess logs
	Out    io.Writer
	Sync   func(ctx context.Context) error                                       // 004's issue sync
	Draft  func(ctx context.Context, item string) (string, error)                // author.Pass for one item
	Pilot  func(ctx context.Context, args []string, logPath string) (int, error) // the pilot binary
	Post   func(ctx context.Context, issue int, body string) error               // a comment on an item's issue
	Issues func(ctx context.Context) (map[string]int, error)

	shown map[string]string // the last line printed per item: print only changes
}

var runIDRE = regexp.MustCompile(`== run (\d{8}T\d{6}Z)`)

// Pass is one reconcile pass. It returns true when an item's doctor or run failed in it.
func (m *Manager) Pass(ctx context.Context) (bool, error) {
	if m.Sync != nil {
		if err := m.Sync(ctx); err != nil {
			fmt.Fprintf(m.Out, "step: issue sync failed (%v); continuing with the issues as they are\n", err)
		}
	}
	states, err := Gather(ctx, m.Src)
	if err != nil {
		return false, err
	}
	failed := false
	if m.shown == nil {
		m.shown = map[string]string{}
	}
	for _, a := range Plan(states) {
		if line := a.String(); m.shown[a.Item] != line {
			m.shown[a.Item] = line
			fmt.Fprintln(m.Out, line)
		}
		var st ItemState
		for _, s := range states {
			if s.ID == a.Item {
				st = s
			}
		}
		switch a.Kind {
		case Draft:
			res, err := m.Draft(ctx, a.Item)
			if err != nil {
				return failed, err
			}
			_ = m.Log.Add(Entry{Item: a.Item, Action: Draft, Reason: "drafting pass: " + res})
		case NeedsTests:
			m.comment(ctx, a.Item, "needs-tests:"+a.Item, "Hidden tests are needed before this item can run. Auto-draft is off for this target: write them in the companion repository, or turn on `PILOT_AUTHOR_AUTO`.")
		case RunFailedAct:
			failed = failed || !m.Log.Posted("run-failed:"+st.RunID) // report a failure once
			m.comment(ctx, a.Item, "run-failed:"+st.RunID, fmt.Sprintf("Run %s ended without verifying this item. Step mode leaves it until the proposal label is added again.", st.RunID))
		case Doctor:
			if f, err := m.doctorThenRun(ctx, st); err != nil {
				return failed, err
			} else if f {
				failed = true
			}
		}
	}
	return failed, nil
}

func (m *Manager) doctorThenRun(ctx context.Context, st ItemState) (bool, error) {
	n := len(m.Log.entries) + 1
	dlog := filepath.Join(m.Dir, st.ID, fmt.Sprintf("%03d-doctor.log", n))
	code, err := m.Pilot(ctx, []string{"doctor", "--target", m.Target, "--item", st.ID, "--baseline"}, dlog)
	if err != nil {
		return false, err
	}
	if code != 0 {
		_ = m.Log.Add(Entry{Item: st.ID, Action: DoctorFailed, Reason: fmt.Sprintf("doctor exit %d", code), Commits: st.Commits})
		m.comment(ctx, st.ID, "doctor-failed@"+st.Commits, "`pilot doctor --baseline` found problems, so this item didn't start. Step mode tries again when the target or the companion changes.\n\n```text\n"+findings(dlog)+"\n```")
		return true, nil
	}
	_ = m.Log.Add(Entry{Item: st.ID, Action: "doctor-ok", Reason: "doctor --baseline passed", Commits: st.Commits})
	rlog := filepath.Join(m.Dir, st.ID, fmt.Sprintf("%03d-run.log", n))
	code, err = m.Pilot(ctx, []string{"run", "--target", m.Target, "--item", st.ID}, rlog)
	if err != nil {
		return false, err
	}
	raw, _ := os.ReadFile(rlog)
	id := ""
	if g := runIDRE.FindSubmatch(raw); g != nil {
		id = string(g[1])
	}
	runDir := ""
	if id != "" {
		runDir = filepath.Join(filepath.Dir(filepath.Dir(m.Dir)), id) // <evidence root>/<run id>
	}
	_ = m.Log.Add(Entry{Item: st.ID, Action: "run", Reason: fmt.Sprintf("run exit %d (5 = paused for /approve mode)", code), Run: runDir})
	fmt.Fprintf(m.Out, "%s: run %s started; it waits for /approve mode on its tracking issue\n", st.ID, id)
	return code != 0 && code != 5, nil
}

// comment posts once per marker on the item's issue.
func (m *Manager) comment(ctx context.Context, item, marker, body string) {
	if m.Log.Posted(marker) {
		return
	}
	issues, err := m.Issues(ctx)
	n, ok := issues[item]
	if err != nil || !ok {
		fmt.Fprintf(m.Out, "%s: could not find its issue to comment on\n", item)
		return
	}
	if err := m.Post(ctx, n, "<!-- pilot-step:"+marker+" -->\n"+body); err != nil {
		fmt.Fprintf(m.Out, "%s: comment failed: %v\n", item, err)
		return
	}
	_ = m.Log.Add(Entry{Item: item, Action: "comment", Reason: strings.SplitN(marker, ":", 2)[0], Comment: marker})
}

// findings are doctor's FAIL and WARN lines (they never carry secrets; R5).
func findings(logPath string) string {
	raw, _ := os.ReadFile(logPath)
	var keep []string
	for _, l := range strings.Split(string(raw), "\n") {
		if t := strings.TrimSpace(l); strings.HasPrefix(t, "FAIL ") || strings.HasPrefix(t, "WARN ") || strings.HasPrefix(t, "doctor:") {
			keep = append(keep, t)
		}
	}
	if len(keep) == 0 {
		return "(no findings printed; see the doctor log)"
	}
	return strings.Join(keep, "\n")
}
