// Package diagnose builds the deterministic diagnosis of a failed attempt (033): the signals in
// the attempt's kept evidence, the triggers they point to (unknown work, an environment need) and
// one recommended choice (retry, raise the budget, back to planning). It reads files and plain
// values only: no clock, network or model (033 FR-006), so a diagnosis replays on kept evidence.
package diagnose

// KnownReasons are every attempt failure reason and worker report error the code can produce
// (033 research R4). Each has an entry in docs/troubleshooting.md (internal/cli docs test); a
// reason outside this list is "not recognised" and points to unknown work.
var KnownReasons = []string{
	// pipeline and worker
	"deadline", "spend_cap", "grade_fail", "harness_error",
	// result frames
	"no_result", "bad_frame", "oversize",
	// grader
	"grader_error", "grader_timeout",
	// worker report errors (pilot/images/worker/entrypoint.sh)
	"clone_failed", "checkout_failed", "workspace_failed", "bundle_failed", "bad_meta",
	"induction_failed",
}

// Known reports whether reason is a catalogued failure reason.
func Known(reason string) bool {
	for _, r := range KnownReasons {
		if r == reason {
			return true
		}
	}
	return false
}
