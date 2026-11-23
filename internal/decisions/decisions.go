// Package decisions handles human decisions with or without a terminal
// (reference: the former Python pilot's approvals.py; specs/012-agent-approvals, 004). It is not
// called "approvals" because the project's harness rules deny edits under any approvals/ path.
//
// With a terminal, the pilot asks there (channel tty) and any pending approval for the run is
// discarded: the terminal answer is the current decision. Without one, nothing is assumed: the
// run pauses, and a human approves the one pending step with `./chalkeus forge approve` (channel harness)
// or on GitHub (channel github, 004). An approval is accepted only for the step the run is
// paused at, after it was shown; it is bound to that step (task, attempt, merge commit) and
// used at most once; every decision is recorded with its channel.
package decisions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/names"
)

func file(run *evidence.Run) string { return filepath.Join(run.Dir, "approvals", "pending.json") }

// HasTTY reports whether a controlling terminal is available (replaced in tests).
var HasTTY = func() bool {
	if NoTTY() {
		return false
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	return term.IsTerminal(int(f.Fd()))
}

// AskTTY asks on the controlling terminal; no terminal or any error gives "" (no, fail closed).
var AskTTY = func(question string) string {
	if NoTTY() {
		return ""
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return ""
	}
	defer func() { _ = tty.Close() }()
	if _, err := tty.WriteString(question); err != nil {
		return ""
	}
	line, _ := bufio.NewReader(tty).ReadString('\n')
	return line
}

// Pending is the decision a run is paused at.
type Pending struct {
	Step        string // mode | retry | reconcile | merge
	Task        any    // nil, "A"/"B" (fixture), or the item id (014 item run)
	N           any    // nil or the attempt number
	Mode        any
	MergeCommit string
	Proposed    string
	ItemID      any
	Target      string // an item run's "target <name> (<repo>), item NNN, …" line (014)
}

// mergeTask is the task whose PR a paused run awaits: A, or an item run's item.
func mergeTask(run *evidence.Run) string {
	if run.Kind() == evidence.KindItem {
		item, _ := run.State["item"].(string)
		return item
	}
	return "A"
}

func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	case json.Number:
		i, err := x.Int64()
		return int(i), err == nil
	case string:
		var i int
		_, err := fmt.Sscanf(x, "%d", &i)
		return i, err == nil
	}
	return 0, false
}

func readJSON(path string) map[string]any {
	var m map[string]any
	if raw, err := os.ReadFile(path); err == nil {
		_ = evidence.Decode(raw, &m)
	}
	return m
}

// PendingStep returns the decision the run is paused at, or nil.
func PendingStep(run *evidence.Run) *Pending {
	st, _ := run.State["state"].(string)
	pend, _ := run.State["pending"].(map[string]any)
	switch st {
	case "paused(awaiting-mode)":
		target, _ := run.State["proposed_target"].(string)
		return &Pending{Step: "mode", Mode: run.State["proposed_mode"], Target: target}
	case "paused(awaiting-retry)":
		n, _ := toInt(pend["n"])
		return &Pending{Step: "retry", Task: pend["task"], N: n}
	case "paused(awaiting-acceptance)": // 026: the item-level acceptance check before `verified`
		return &Pending{Step: "acceptance", Task: pend["task"]}
	case "paused(awaiting-merge)":
		task := mergeTask(run)
		rec := readJSON(filepath.Join(run.Dir, "reconcile", task+".json"))
		mc, _ := rec["merge_commit"].(string)
		prop, _ := rec["proposed"].(string)
		if rec["reason"] == "DECLINED" && prop != "" && mc != "" {
			return &Pending{Step: "reconcile", Task: task, MergeCommit: mc, Proposed: prop, ItemID: rec["item_id"]}
		}
		return &Pending{Step: "merge", Task: task}
	}
	return nil
}

// PyJSON converts compact JSON to Python's json.dumps spacing (", " and ": ").
func PyJSON(raw string) string {
	var b strings.Builder
	inStr, esc := false, false
	for _, c := range raw {
		b.WriteRune(c)
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case ',', ':':
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func pyNone(v any) string {
	if v == nil {
		return "None"
	}
	return fmt.Sprint(v)
}

// Describe is the decision in plain words.
func Describe(p *Pending) string {
	switch p.Step {
	case "mode":
		mode, _ := json.Marshal(p.Mode) // maps marshal with sorted keys
		if p.Target != "" {
			return "start the run in this mode: " + PyJSON(string(mode)) + "; " + p.Target
		}
		return "start the run in this mode: " + PyJSON(string(mode))
	case "retry":
		return fmt.Sprintf("retry task %v (attempt %v)", p.Task, p.N)
	case "acceptance":
		return fmt.Sprintf("record that item %v's acceptance checks were done and passed (the status change then follows)", p.Task)
	case "reconcile":
		where := "the fixture roadmap"
		if t, ok := p.Task.(string); ok && t != "A" {
			where = "the target's roadmap"
		}
		return fmt.Sprintf("apply '%s: %s' to %s (merge commit %s)", pyNone(p.ItemID), p.Proposed, where, p.MergeCommit)
	}
	return "merge"
}

// Record approves exactly the pending step (`./chalkeus forge approve`: channel harness; 004: github with
// the approver's login and comment id). It returns (exit code, message).
func Record(run *evidence.Run, step, channel, approver string, commentID any) (int, string) {
	p := PendingStep(run)
	if p == nil {
		return 2, fmt.Sprintf("run %s is not paused at a decision (state=%s)", run.ID, pyNone(run.State["state"]))
	}
	if p.Step == "merge" {
		if p.Task != "A" {
			return 4, fmt.Sprintf("the run is waiting for you to merge %v's PR on GitHub; merging is never approved here. "+
				"Merge it, run `./chalkeus forge resume`, then approve the reconcile proposal it shows.", p.Task)
		}
		return 4, "the run is waiting for you to merge A's PR on GitHub; merging is never approved here. " +
			"Merge it, run `./chalkeus forge resume`, then approve the reconcile proposal it shows."
	}
	if p.Step != step {
		return 2, fmt.Sprintf("run %s is waiting for '%s', not '%s': %s", run.ID, p.Step, step, Describe(p))
	}
	if channel == "" {
		channel = "harness"
	}
	a := map[string]any{"step": p.Step, "task": p.Task, "n": p.N, "channel": channel, "at": evidence.Now()}
	if p.Step == "reconcile" {
		a["merge_commit"], a["proposed"], a["item_id"] = p.MergeCommit, p.Proposed, p.ItemID
	}
	extra := ""
	if channel == "github" {
		a["approver"], a["comment_id"] = approver, commentID
		extra = fmt.Sprintf(" approver=%s comment=%v", approver, commentID)
	}
	if err := evidence.WriteJSON(file(run), a); err != nil {
		return 2, err.Error()
	}
	_ = run.Event("approval", p.Task, p.N, "approval_recorded", fmt.Sprintf("step=%s channel=%s%s", step, channel, extra))
	by := ""
	if channel == "github" {
		by = " by @" + approver + " on GitHub"
	}
	return 0, fmt.Sprintf("approved%s: %s. Now run: ./chalkeus forge resume --run %s", by, Describe(p), run.ID)
}

func eq(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ai, aok := toInt(a)
	bi, bok := toInt(b)
	if aok && bok {
		return ai == bi
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

// Recorded reports whether an approval for exactly this step, task and attempt is already
// recorded and not yet used (it does not consume it).
func Recorded(run *evidence.Run, step string, task, n any) bool {
	a := readJSON(file(run))
	return a != nil && a["step"] == step && eq(a["task"], task) && eq(a["n"], n)
}

// TakeRecord uses the recorded approval for this exact step, once (deleted either way).
func TakeRecord(run *evidence.Run, step string, task, n any) map[string]any {
	raw, err := os.ReadFile(file(run))
	if err != nil {
		return nil
	}
	_ = os.Remove(file(run))
	var a map[string]any
	if evidence.Decode(raw, &a) != nil {
		return nil
	}
	if a["step"] != step || !eq(a["task"], task) || !eq(a["n"], n) {
		return nil
	}
	if step == "reconcile" && readJSON(filepath.Join(run.Dir, "reconcile", mergeTask(run)+".json"))["merge_commit"] != a["merge_commit"] {
		return nil
	}
	return a
}

// Decide asks on the terminal if there is one (discarding any pending approval: the terminal
// decides), otherwise uses a recorded approval for this step. It returns (approved, channel):
// tty, harness, github or none. The used record is kept on run.LastApproval.
func Decide(run *evidence.Run, step, question string, task, n any, say func(string)) (bool, string) {
	run.LastApproval = nil
	if !NoTTY() && HasTTY() { // PILOT_NO_TTY: never ask on an inherited terminal (021)
		defer func() { _ = os.Remove(file(run)) }()
		return strings.ToLower(strings.TrimSpace(AskTTY(question))) == "y", "tty"
	}
	if a := TakeRecord(run, step, task, n); a != nil {
		ch, _ := a["channel"].(string)
		if ch == "" {
			ch = "harness"
		}
		run.LastApproval = a
		how := "approved with `./chalkeus forge approve`"
		if ch == "github" {
			how = fmt.Sprintf("approved on GitHub by @%v", a["approver"])
		}
		say(fmt.Sprintf("%sy  (%s)", question, how))
		return true, ch
	}
	return false, "none"
}

// Note records an approval_channel event; approver and extra apply to the github channel.
func Note(run *evidence.Run, step string, task, n any, approved bool, channel, approver, extra string) {
	detail := fmt.Sprintf("step=%s approved=%t channel=%s", step, approved, channel)
	if channel == "github" {
		var a map[string]any
		if approver == "" {
			a = run.LastApproval
			approver, _ = a["approver"].(string)
		}
		detail += " approver=" + approver
		if cid, ok := a["comment_id"]; ok && cid != nil {
			detail += fmt.Sprintf(" comment=%v", cid)
		}
		if extra != "" {
			detail += " " + extra
		}
	}
	_ = run.Event("approval", task, n, "approval_channel", detail)
}

// Hint tells a no-terminal caller how to continue.
func Hint(run *evidence.Run, step string) string {
	return fmt.Sprintf("no terminal: approve with `./chalkeus forge approve --run %s %s` (your agent harness asks you first), then `./chalkeus forge resume --run %s`",
		run.ID, step, run.ID)
}

// NoTTY: PILOT_NO_TTY=1 means a process started on the operator's behalf (step mode, 021) must
// never ask on a terminal, even one it inherited: every decision goes to GitHub instead.
func NoTTY() bool { return names.Env("NO_TTY") == "1" }
