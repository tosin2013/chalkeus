// Package author is the test author's side of the pilot (specs/018-test-author): the draft
// workspace, the checks a draft must pass, and the companion pull request. A separate agent
// session (the test-author skill) writes the test files; these helpers own every side effect,
// so that session never handles credentials. Nothing here ever records a test file's contents.
package author

import "fmt"

// Refusal and result codes (specs/018-test-author/contracts/cli-author.md).
const (
	CodeCompanionNotEmpty  = "COMPANION_NOT_EMPTY"
	CodeDraftPROpen        = "DRAFT_PR_OPEN"
	CodeCanWriteTarget     = "AUTHOR_CAN_WRITE_TARGET"
	CodeNeedsClusterMode   = "LEVEL_NEEDS_CLUSTER_MODE"
	CodeDraftEmpty         = "DRAFT_EMPTY"
	CodeDraftOutsideFolder = "DRAFT_OUTSIDE_FOLDER"
	CodeMapInvalid         = "MAP_INVALID"
	CodeDraftNotChecked    = "DRAFT_NOT_CHECKED"
	CodeDraftHiddenPasses  = "DRAFT_HIDDEN_PASSES"
	CodeDraftSuiteFails    = "DRAFT_SUITE_FAILS"
	CodeDraftOK            = "DRAFT_OK"
	CodeDraftGraderError   = "DRAFT_GRADER_ERROR"
	// CodeDraftPathInSpec: a drafted file reuses a path the item's spec, plan or tasks name. Those
	// are the build worker's own files: the worker would see the hidden test's path (P11), and
	// the grader's overlay would replace the worker's file of that name.
	CodeDraftPathInSpec = "DRAFT_PATH_IN_SPEC"
	// 041 reference check (specs/041-proven-hidden-tests/contracts/commands.md).
	CodeDraftFailsOnReference = "DRAFT_FAILS_ON_REFERENCE"
	CodeDraftPassesOnBase     = "DRAFT_PASSES_ON_BASE"
	CodeDraftFlaky            = "DRAFT_FLAKY"
	CodeReferenceBroken       = "REFERENCE_BROKEN"
	CodeReferenceDoesntApply  = "REFERENCE_DOESNT_APPLY"
	exitFail, exitError = 1, 2
	exitRefuse          = 10
)

// failCodes are results (exit 1); every other code is a refusal (exit 10).
var failCodes = map[string]bool{CodeDraftHiddenPasses: true, CodeDraftSuiteFails: true, CodeDraftGraderError: true,
	CodeDraftFailsOnReference: true, CodeDraftPassesOnBase: true, CodeDraftFlaky: true}

// Error is a refusal or a failed result: one code the operator can act on (SC-005).
type Error struct {
	Code   string
	Detail string
}

func (e *Error) Error() string { return e.Code + ": " + e.Detail }

// ExitCode follows the pilot's table: 1 for a failed check, 10 for a refusal.
func (e *Error) ExitCode() int {
	if failCodes[e.Code] {
		return exitFail
	}
	return exitRefuse
}

func refuse(code, format string, a ...any) *Error {
	return &Error{Code: code, Detail: fmt.Sprintf(format, a...)}
}
