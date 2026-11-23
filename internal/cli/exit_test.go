package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func run(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Execute(append([]string{"forge"}, args...), "test", &out, &errb)
	return code, out.String(), errb.String()
}

func TestExitMapping(t *testing.T) {
	var b bytes.Buffer
	cases := []struct {
		err  error
		code int
		msg  string
	}{
		{nil, 0, ""},
		{Code(4), 4, ""},
		{Exit(3, "repo guard: %s", "x"), 3, "chalkeus: repo guard: x\n"},
		{Exit(0, "ignored"), 0, ""},
		{errors.New("boom"), 2, "chalkeus: boom\n"},
	}
	for _, c := range cases {
		b.Reset()
		if got := exitCode(c.err, &b); got != c.code || b.String() != c.msg {
			t.Errorf("%v: got %d %q, want %d %q", c.err, got, b.String(), c.code, c.msg)
		}
	}
}

func TestCodesMatchReference(t *testing.T) {
	want := map[string]int{"ok": 0, "fail": 1, "error": 2, "guard": 3, "merge": 4, "declined": 5, "retry": 6, "refuse": 10}
	got := map[string]int{"ok": ExitOK, "fail": ExitFail, "error": ExitError, "guard": ExitGuard, "merge": ExitAwaitMerge,
		"declined": ExitDeclined, "retry": ExitAwaitRetry, "refuse": ExitGateRefusal}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %d, want %d", k, got[k], v)
		}
	}
}

func TestUsageErrorsExit2(t *testing.T) {
	for _, args := range [][]string{{"nope"}, {"gate", "C"}, {"approve", "merge", "--run", "x"}, {"approve", "mode"}, {"audit", "--bogus"}} {
		if code, _, _ := run(args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}

func TestVersionAndHelp(t *testing.T) {
	code, out, _ := run("version")
	if code != 0 || !strings.Contains(out, "chalkeus test") {
		t.Fatalf("version: %d %q", code, out)
	}
	_, out, _ = run("--help")
	for _, sub := range []string{"up", "down", "run", "resume", "approve", "gate", "audit", "status", "github",
		"publisher-token", "import-opencode", "reset-fixture", "app-key", "app-token"} {
		if !strings.Contains(out, "\n  "+sub+" ") {
			t.Errorf("help lacks %s", sub)
		}
	}
}
