package pilotsample

import "github.com/google/go-cmp/cmp"

// Same reports whether two string slices are equal.
func Same(a, b []string) bool { return cmp.Equal(a, b) }
