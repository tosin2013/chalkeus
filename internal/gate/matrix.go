package gate

import (
	"embed"
	"fmt"
	"reflect"
)

//go:embed matrix/*
var matrixFS embed.FS

// MatrixCase is one fail-closed matrix case: a recorded `spec status --json` output and the
// decision the gate must reach (reference: the former Python pilot's tests/test_gate.py CASES).
type MatrixCase struct {
	File, Item, Want string
	Reasons          []string
}

// MatrixCases are the gate's fail-closed matrix.
var MatrixCases = []MatrixCase{
	{"ready_A.json", "001", "allow", []string{"OK"}},
	{"blocked_B.json", "002", "refuse", []string{"DEP_UNSATISFIED:001"}},
	{"ready_B_after_verify.json", "002", "allow", []string{"OK"}},
	{"schema_1_1.json", "001", "refuse", []string{"SCHEMA_MISMATCH"}},
	{"incomplete.json", "001", "refuse", []string{"INCOMPLETE"}},
	{"error_set.json", "001", "refuse", []string{"UNEVALUABLE:tool_error"}},
	{"item_missing.json", "001", "refuse", []string{"ITEM_MISSING"}},
	{"item_duplicate.json", "001", "refuse", []string{"ITEM_DUPLICATE"}},
	{"status_unknown.json", "001", "refuse", []string{"STATUS_UNKNOWN"}},
	{"inactive.json", "001", "refuse", []string{"INACTIVE"}},
	{"already_verified.json", "001", "refuse", []string{"ALREADY_VERIFIED"}},
	{"already_implemented.json", "001", "refuse", []string{"ALREADY_IMPLEMENTED"}},
	{"item_error_diag.json", "001", "refuse", []string{"ITEM_ERROR_DIAGNOSTIC:SS_TASK_FILE_MISSING"}},
	{"inconsistent_count.json", "002", "refuse", []string{"DEP_UNSATISFIED:001", "INCONSISTENT_BLOCKED_COUNT"}},
	{"not_startable.json", "001", "refuse", []string{"STATUS_NOT_STARTABLE:deferred"}},
	{"malformed.txt", "001", "refuse", []string{"UNEVALUABLE:bad_json"}},
}

// Fixture returns an embedded status fixture.
func Fixture(name string) (string, error) {
	raw, err := matrixFS.ReadFile("matrix/" + name)
	return string(raw), err
}

// RunMatrix evaluates every matrix case plus the exit-code cases (GM01). It returns whether all
// cases held and a one-line detail ("OK", like the reference's unittest tail).
func RunMatrix() (bool, string) {
	zero, one, two, other := 0, 1, 2, 127
	var failed []string
	for _, c := range MatrixCases {
		out, err := Fixture(c.File)
		if err != nil {
			failed = append(failed, c.File)
			continue
		}
		if d, r := Decide(c.Item, &zero, out); d != c.Want || !reflect.DeepEqual(r, c.Reasons) {
			failed = append(failed, c.File)
		}
	}
	ready, _ := Fixture("ready_A.json")
	extra := []struct {
		exit *int
		out  string
		want []string
	}{{&two, ready, []string{"UNEVALUABLE:exit_2"}}, {&other, "", []string{"UNEVALUABLE:exit_127"}},
		{nil, "", []string{"UNEVALUABLE:spawn_failed"}}, {&zero, "", []string{"UNEVALUABLE:empty"}}, {&zero, "[]", []string{"SCHEMA_MISMATCH"}}}
	for i, x := range extra {
		if d, r := Decide("001", x.exit, x.out); d != "refuse" || !reflect.DeepEqual(r, x.want) {
			failed = append(failed, fmt.Sprintf("exit-case-%d", i))
		}
	}
	if d, _ := Decide("001", &one, ready); d != "allow" {
		failed = append(failed, "exit-1-evaluable")
	}
	if len(failed) > 0 {
		return false, fmt.Sprintf("FAILED (%d cases: %v)", len(failed), failed)
	}
	return true, "OK"
}
