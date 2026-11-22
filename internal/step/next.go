// Package step is 021's manager loop: each pass it gathers where every proposed item of a target
// stands and takes the one next automatic step for the item in flight, then waits for the owner's
// answers on GitHub. The decision is a pure function (Plan, Next) so the operator (011) can reuse
// it on a different host.
package step

import "fmt"

// Run states as the manager sees them.
const (
	RunNone      = ""
	RunActive    = "active" // created, paused for an approval or a merge, or attempting
	RunSucceeded = "succeeded"
	RunFailed    = "failed"
)

// ItemState is everything the decision needs about one candidate item (research R4).
type ItemState struct {
	ID string
	// Slice is the item's current slice (026: the first not merged), "" for an item built whole.
	Slice string
	// Proposed: the proposal label is on its issue and an approver added it last.
	Proposed  bool
	NotWhy    string // why it isn't proposed ("" when proposed)
	Ready     bool   // the author's readiness (every rule except hidden tests) allows it
	ReadyWhy  string // the first readiness reason when not ready
	Hidden    int    // hidden-test files merged in the companion
	TestsPR   bool   // an `author/<NNN>-` pull request is open on the companion
	AutoDraft bool   // the target has PILOT_AUTHOR_AUTO=on
	// AutoDrafted: an automatic draft for this spec version already reached its check (and
	// failed, or its PR was closed): step mode asks the owner instead of drafting again.
	AutoDrafted bool
	Run         string // RunNone | RunActive | RunSucceeded | RunFailed (the newest item run)
	RunID       string
	RunDir      string // its evidence folder (to relay its watcher's log)
	// RunWaiting: what the run waits for, with the link to act on (from its evidence).
	RunWaiting string
	// Commits is "<target commit>:<companion commit>"; DoctorFailedAt is the Commits value at the
	// last failed doctor, so doctor is retried only when either repository changes.
	Commits        string
	DoctorFailedAt string
	// RelabelledAfterRun: the label was added again after the newest run ended (a failed run's
	// item waits for that).
	RelabelledAfterRun bool
}

// Kinds of action.
const (
	Skip         = "skip"          // nothing to do for this item now
	Wait         = "wait"          // the item is in flight, waiting for the owner or a run
	Draft        = "draft"         // start a drafting pass (024) for this item
	NeedsTests   = "needs-tests"   // comment once: hidden tests are needed (auto-draft off)
	Doctor       = "doctor"        // run doctor --baseline, then (if clean) the run
	DoctorFailed = "doctor-failed" // doctor failed at these commits: wait for a change
	RunFailedAct = "run-failed"    // comment once: the run failed; wait for a relabel
	Done         = "done"          // verified through a succeeded run
)

// Action is the decision for one item.
type Action struct {
	Item, Kind, Reason string
}

func (a Action) String() string { return fmt.Sprintf("%s: %s (%s)", a.Item, a.Kind, a.Reason) }

// Next is FR-004 for one item, ignoring the other items.
func Next(s ItemState) Action {
	a := func(kind, why string) Action { return Action{Item: s.ID, Kind: kind, Reason: why} }
	switch {
	case s.Run == RunActive && s.RunWaiting != "":
		return a(Wait, "run "+s.RunID+": "+s.RunWaiting)
	case s.Run == RunActive:
		return a(Wait, "run "+s.RunID+" is in progress")
	case !s.Proposed:
		return a(Skip, "not proposed: "+s.NotWhy)
	case s.Run == RunSucceeded:
		return a(Done, "run "+s.RunID+" succeeded")
	case s.Run == RunFailed && !s.RelabelledAfterRun:
		return a(RunFailedAct, "run "+s.RunID+" failed; relabel the issue to try again")
	case !s.Ready:
		return a(Skip, "not ready: "+s.ReadyWhy)
	case s.Hidden == 0 && s.TestsPR:
		return a(Wait, "waiting for you: review and merge the hidden-tests PR on the companion repository")
	case s.Hidden == 0 && s.AutoDraft && s.AutoDrafted:
		return a(NeedsTests, "the automatic draft for this spec version didn't make it; draft the hidden tests by hand, or change the spec")
	case s.Hidden == 0 && s.AutoDraft:
		return a(Draft, "no hidden tests yet")
	case s.Hidden == 0:
		return a(NeedsTests, "no hidden tests, and auto-draft is off")
	case s.DoctorFailedAt != "" && s.DoctorFailedAt == s.Commits:
		return a(DoctorFailed, "doctor failed at these commits; change the target or the companion")
	default:
		return a(Doctor, "hidden tests merged; doctor --baseline, then the run")
	}
}

// inFlight: the actions that hold the target's one slot. Failures (doctor-failed, run-failed)
// don't: their item waits for the owner, and the next proposed item goes on (US3).
func inFlight(kind string) bool {
	return kind == Wait || kind == Draft || kind == NeedsTests || kind == Doctor
}

// Plan decides a whole pass: one item in flight per target (FR-003). An item with an active run
// (proposed or started by hand) is in flight; otherwise the first item in roadmap order whose
// action holds the slot is. Every other such item waits behind it.
func Plan(items []ItemState) []Action {
	out := make([]Action, len(items))
	flight := -1
	for i, s := range items {
		if s.Run == RunActive {
			flight = i
			break
		}
	}
	for i, s := range items {
		out[i] = Next(s)
		if flight < 0 && inFlight(out[i].Kind) {
			flight = i
		}
	}
	for i := range out {
		if flight >= 0 && i != flight && inFlight(out[i].Kind) {
			out[i] = Action{Item: items[i].ID, Kind: Skip, Reason: "item " + items[flight].ID + " is in flight"}
		}
	}
	return out
}
