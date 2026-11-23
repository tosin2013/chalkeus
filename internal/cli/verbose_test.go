package cli

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/verbose"
)

// FR-018: -v changes only stderr; stdout and the exit code stay as they are.
func TestVerboseFlagLeavesStdoutAlone(t *testing.T) {
	defer verbose.Set(0, nil)
	var quiet, loud, errQ, errL strings.Builder
	c0 := Execute([]string{"version"}, "test", &quiet, &errQ)
	c1 := Execute([]string{"-vvv", "version"}, "test", &loud, &errL)
	if c0 != c1 || quiet.String() != loud.String() {
		t.Fatalf("stdout or exit code changed: %d %q vs %d %q", c0, quiet.String(), c1, loud.String())
	}
	if verbose.Level() != 3 {
		t.Fatalf("level %d", verbose.Level())
	}
}

func TestPilotVerboseEnv(t *testing.T) {
	defer verbose.Set(0, nil)
	t.Setenv("PILOT_VERBOSE", "2")
	var o, e strings.Builder
	Execute([]string{"version"}, "test", &o, &e)
	if verbose.Level() != 2 {
		t.Fatalf("level %d", verbose.Level())
	}
}
