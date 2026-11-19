package cli

import (
	"io"
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
