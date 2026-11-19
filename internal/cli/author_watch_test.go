package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	code, err := runPilot(context.Background(), "/bin/sh", t.TempDir(), []string{"-c", `echo "by=$PILOT_STARTED_BY"; exit 5`}, log)
	raw, _ := os.ReadFile(log)
	if err != nil || code != 5 || !strings.Contains(string(raw), "by=step") {
		t.Fatalf("%d %v %q", code, err, raw)
	}
}
