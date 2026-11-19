package cli

import "testing"

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
