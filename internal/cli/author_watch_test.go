package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/verbose"
)

// 024 contracts/author-watch.md: `pilot author watch --target <name> [--once]`.
func TestAuthorWatchCommand(t *testing.T) {
	root := NewRoot("test")
	c, _, err := root.Find([]string{"author", "watch"})
	if err != nil || c.Name() != "watch" {
		t.Fatalf("author watch: %v", err)
	}
	for _, f := range []string{"target", "once"} {
		if c.Flags().Lookup(f) == nil {
			t.Errorf("missing --%s", f)
		}
	}
}

func TestWatchVerbosityDefaultsToVV(t *testing.T) {
	if watchVerbosity(0, false) != 2 {
		t.Fatal("author watch defaults to -vv")
	}
	verbose.Set(1, io.Discard)
	defer verbose.Set(0, nil)
	if watchVerbosity(1, false) != 1 || watchVerbosity(0, true) != 1 {
		t.Fatal("-v or PILOT_VERBOSE keep the level they chose")
	}
}

// 021 contracts/step.md: `pilot step --target <name> [--once]`.
func TestStepCommand(t *testing.T) {
	c, _, err := NewRoot("test").Find([]string{"step"})
	if err != nil || c.Name() != "step" || c.Flags().Lookup("target") == nil || c.Flags().Lookup("once") == nil {
		t.Fatalf("step: %v", err)
	}
}

// runPilot runs a program with its output in the log, records step mode, and passes the exit code.
func TestRunPilot(t *testing.T) {
	log := filepath.Join(t.TempDir(), "x", "run.log")
	var echo strings.Builder
	code, err := runPilot(context.Background(), "/bin/sh", t.TempDir(),
		[]string{"-c", `echo "by=$PILOT_STARTED_BY notty=$PILOT_NO_TTY"; exit 5`}, log, &echo)
	raw, _ := os.ReadFile(log)
	if err != nil || code != 5 || !strings.Contains(string(raw), "by=step notty=1") {
		t.Fatalf("%d %v %q", code, err, raw)
	}
	if !strings.Contains(echo.String(), "  | by=step") {
		t.Fatalf("the output is echoed, indented: %q", echo.String())
	}

}

// Step mode follows the runs' rule: approvals on means issues on (found live, 2026-11-19).
func TestStepGitHubOn(t *testing.T) {
	if !stepGitHubOn(config.Target{GithubIssues: "off", GithubApprovals: "on"}) || stepGitHubOn(config.Target{GithubIssues: "on", GithubApprovals: "off"}) {
		t.Fatal("approvals decide")
	}
}

func TestStepWait(t *testing.T) {
	if stepWait(true, 120).Seconds() != 15 || stepWait(false, 120).Seconds() != 120 || stepWait(true, 10).Seconds() != 10 {
		t.Fatal("15 s while busy, the interval otherwise")
	}
}
