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
	"sort"
	"strings"

	"github.com/tosin2013/chalkeus/internal/evidence"
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

	// Busy: the last pass saw a run in flight; the loop then polls sooner, so short waits (a
	// status PR merged within seconds) are seen.
	Busy bool
	// Next: what happens next after the last pass, in words for the owner: what they do (with
	// the link) or that step mode is working and nothing is needed. The loop prints it when it
	// changes.
	Next string

	shown map[string]string // the last line printed per item: print only changes
	relay map[string]int64  // bytes of each run's watcher log already relayed
	// relayItem is the item of each relayed log, to finish relaying a run whose item has left the
	// candidates (it became verified when its status PR merged).
	relayItem map[string]string
}

// relayed reports whether this process has relayed the run's log before.
func (m *Manager) relayed(dir string) bool {
	if dir == "" || m.relay == nil {
		return false
	}
	_, ok := m.relay[filepath.Join(dir, "github", "watch.log")]
	return ok
}

// relayRun prints what the run's watcher logged since the last pass, so the owner follows the
// run in step mode's own terminal (approvals, attempts, grades, PR links).
func (m *Manager) relayRun(item, dir string) {
	if dir == "" {
		return
	}
	if m.relay == nil {
		m.relay, m.relayItem = map[string]int64{}, map[string]string{}
	}
	p := filepath.Join(dir, "github", "watch.log")
	m.relayItem[p] = item
	f, err := os.Open(p)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Seek(m.relay[p], io.SeekStart); err != nil {
		return
	}
	b, _ := io.ReadAll(f)
	m.relay[p] += int64(len(b))
	for _, l := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if strings.TrimSpace(l) != "" {
			fmt.Fprintf(m.Out, "%s run: %s\n", item, l)
		}
	}
}

// finishRelays keeps relaying each run this manager was following whose item is no longer a
// candidate, until the run has ended, then relays it once more and stops following it. Merging the
// status PR makes the item verified, which takes it out of the candidates while its run is still
// finishing (the audit, then "succeeded" or "failed"); found by the 025 step scenario, since #124
// fixed this only for items still proposed. While such a run is finishing the loop stays busy, so
// it polls every 15 s.
func (m *Manager) finishRelays(states []ItemState) {
	live := map[string]bool{}
	for _, s := range states {
		if s.RunDir != "" {
			live[filepath.Join(s.RunDir, "github", "watch.log")] = true
		}
	}
	var done []string
	for p := range m.relay {
		if !live[p] {
			done = append(done, p)
		}
	}
	sort.Strings(done)
	for _, p := range done {
		dir := filepath.Dir(filepath.Dir(p))
		ended := runEnded(dir) // read before relaying, so no line written at the end is missed
		m.relayRun(m.relayItem[p], dir)
		if ended {
			delete(m.relay, p)
			delete(m.relayItem, p)
		} else {
			m.Busy = true
		}
	}
}

// runEnded reports whether a run's recorded state is final (succeeded, failed or declined), or
// its record can't be read (nothing more will be written that step mode can follow).
func runEnded(dir string) bool {
	var st struct {
		State string `json:"state"`
	}
	raw, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil || json.Unmarshal(raw, &st) != nil {
		return true
	}
	return st.State == "succeeded" || strings.HasPrefix(st.State, "failed") || strings.HasPrefix(st.State, "declined")
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
	m.Busy = false
	m.Next = ""
	if len(states) == 0 { // nothing can be proposed: say why (once), and what the owner does next
		if m.shown[""] == "" {
			m.shown[""] = "none"
			fmt.Fprintln(m.Out, "step: no item can be proposed now: every active item is verified, or waits on a dependency.")
		}
		var open []string
		if m.Src.Planning != nil {
			open, _ = m.Src.Planning(ctx)
		}
		m.Next = "plan a new item: merge its spec and tasks, then label its issue"
		if len(open) > 0 {
			m.Next = "review and merge the planning PR " + open[0] + "; then label the new item's issue"
		}
	}
	if len(states) > 0 {
		delete(m.shown, "")
	}
	m.finishRelays(states)
	plan := Plan(states)
	m.Next = nextFor(plan, m.Next)
	for _, a := range plan {
		if ctx.Err() != nil { // stopped: start nothing new (T010)
			return failed, nil
		}
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
		// Relay while the run is active, and once more after it ends: lines written between
		// passes (a status PR merged within seconds, "succeeded") must still reach the owner.
		if st.Run == RunActive || m.relayed(st.RunDir) {
			m.relayRun(a.Item, st.RunDir)
		}
		if st.Run == RunActive {
			m.Busy = true
		}
		switch a.Kind {
		case Draft:
			res, err := m.Draft(ctx, a.Item)
			if err != nil {
				return failed, err
			}
			_ = m.Log.Add(Entry{Item: a.Item, Action: Draft, Reason: "drafting pass: " + res})
			m.Next = draftNext(a.Item, res)
		case NeedsTests:
			m.comment(ctx, a.Item, "needs-tests:"+a.Item, "Hidden tests are needed before this item can run. Auto-draft is off for this target: write them in the companion repository, or turn on `FORGE_AUTHOR_AUTO`.")
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
	code, err := m.Pilot(ctx, withSlice([]string{"doctor", "--target", m.Target, "--item", st.ID, "--baseline"}, st.Slice), dlog)
	if err != nil {
		return false, err
	}
	if code != 0 {
		_ = m.Log.Add(Entry{Item: st.ID, Action: DoctorFailed, Reason: fmt.Sprintf("doctor exit %d", code), Commits: st.Commits})
		m.Next = st.ID + ": waiting for you: doctor found problems (listed in the comment on its issue); fix them, and step mode tries again once the target or the companion changes"
		m.comment(ctx, st.ID, "doctor-failed@"+st.Commits, "`./chalkeus forge doctor --baseline` found problems, so this item didn't start. Step mode tries again when the target or the companion changes.\n\n```text\n"+findings(dlog)+"\n```")
		return true, nil
	}
	_ = m.Log.Add(Entry{Item: st.ID, Action: "doctor-ok", Reason: "doctor --baseline passed", Commits: st.Commits})
	rlog := filepath.Join(m.Dir, st.ID, fmt.Sprintf("%03d-run.log", n))
	code, err = m.Pilot(ctx, withSlice([]string{"run", "--target", m.Target, "--item", st.ID}, st.Slice), rlog)
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
	if code == 5 {
		m.Next = st.ID + ": " + waitingFor(runDir, "paused(awaiting-mode)", m.Src.Repo)
	} else {
		m.Next = fmt.Sprintf("%s: run %s ended with exit %d; its output is above", st.ID, id, code)
	}
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

// draftNext is the owner's next step after a drafting pass (the pass already printed the PR link
// or the reason).
func draftNext(item, res string) string {
	switch res {
	case "opened":
		return item + ": waiting for you: review and merge the hidden-tests pull request linked above (on the companion repository)"
	case "failed":
		return item + ": the automatic draft failed its check (reason above); write the hidden tests in the companion repository, or change the spec for a new draft"
	}
	return item + ": no draft was started (see above)"
}

// nextFor is the owner's next step before a pass acts (Pass replaces it with the outcome of a
// draft, a doctor or a run start, so the line printed after the pass is never about a step that
// already happened): the item in flight decides; otherwise the
// first item waiting for its label; otherwise fallback.
func nextFor(plan []Action, fallback string) string {
	for _, a := range plan {
		switch a.Kind {
		case Wait:
			if strings.Contains(a.Reason, "waiting for you") {
				return a.Item + ": " + a.Reason[strings.Index(a.Reason, "waiting for you"):]
			}
			return a.Item + ": " + a.Reason + "; nothing needed from you"
		case Draft:
			return a.Item + ": drafting its hidden tests in the sandbox (about 10 minutes); nothing needed from you"
		case Doctor:
			return a.Item + ": running doctor --baseline, then starting the run; nothing needed from you yet"
		case NeedsTests:
			return a.Item + ": waiting for you: " + a.Reason + " (see the comment on its issue)"
		case DoctorFailed:
			return a.Item + ": waiting for you: doctor found problems (listed in the comment on its issue); fix them, and step mode tries again once the target or the companion changes"
		}
	}
	for _, a := range plan {
		if i := strings.Index(a.Reason, "to start it, "); a.Kind == Skip && i >= 0 {
			return a.Item + ": " + a.Reason[i+len("to start it, "):]
		}
		if i := strings.Index(a.Reason, "an approver must "); a.Kind == Skip && i >= 0 {
			return a.Item + ": waiting for an approver: " + a.Reason[i+len("an approver must "):]
		}
	}
	if fallback != "" {
		return fallback
	}
	return "nothing is waiting for you; failures are commented on their items' issues"
}

// withSlice adds --slice to a doctor or run command for a sliced item (026 R9).
func withSlice(args []string, slice string) []string {
	if slice == "" {
		return args
	}
	return append(args, "--slice", slice)
}
