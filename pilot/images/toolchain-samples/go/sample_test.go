package pilotsample

import "testing"

func TestSame(t *testing.T) {
	if !Same([]string{"a"}, []string{"a"}) || Same([]string{"a"}, []string{"b"}) {
		t.Fatal("cmp")
	}
}
