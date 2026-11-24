package debt

import (
	"strconv"
	"strings"
)

// Semver levels of a bump (029 research R4).
const (
	LevelPatch   = "patch"
	LevelMinor   = "minor"
	LevelMajor   = "major"
	LevelUnknown = "unknown"
)

// version is a parsed dotted numeric version with an optional pre-release.
type version struct {
	nums []int
	pre  string
}

// parseVersion reads "1.2.3", "v1.2.3", "1.2.3-rc.1" or "1.2.3+build"; ok is false for anything
// else (ranges, dates with letters, git hashes).
func parseVersion(s string) (version, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var v version
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, v.pre = s[:i], s[i+1:]
	}
	if s == "" {
		return v, false
	}
	for _, p := range strings.Split(s, ".") {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v.nums = append(v.nums, n)
	}
	return v, true
}

func (v version) at(i int) int {
	if i < len(v.nums) {
		return v.nums[i]
	}
	return 0
}

// compareVersions returns -1, 0 or 1, and ok false when either side isn't a version.
func compareVersions(a, b string) (int, bool) {
	va, ok1 := parseVersion(a)
	vb, ok2 := parseVersion(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	n := max(len(va.nums), len(vb.nums))
	for i := 0; i < n; i++ {
		if x, y := va.at(i), vb.at(i); x != y {
			if x < y {
				return -1, true
			}
			return 1, true
		}
	}
	switch {
	case va.pre == vb.pre:
		return 0, true
	case va.pre == "": // a release is newer than its pre-releases
		return 1, true
	case vb.pre == "":
		return -1, true
	case va.pre < vb.pre:
		return -1, true
	default:
		return 1, true
	}
}

// Level is a bump's semver level. The first non-zero component is the major one, so 0.3 → 0.4
// is major (R4); anything that isn't a version is unknown.
func Level(from, to string) string {
	a, ok1 := parseVersion(from)
	b, ok2 := parseVersion(to)
	if !ok1 || !ok2 {
		return LevelUnknown
	}
	n := max(len(a.nums), len(b.nums), 3)
	lead := 0 // index of the first non-zero component of from
	for lead < n-1 && a.at(lead) == 0 && b.at(lead) == 0 {
		lead++
	}
	for i := 0; i < n; i++ {
		if a.at(i) == b.at(i) {
			continue
		}
		switch {
		case i <= lead:
			return LevelMajor
		case i == lead+1:
			return LevelMinor
		default:
			return LevelPatch
		}
	}
	return LevelPatch // only the pre-release differs (or nothing does)
}

// maxLevel is the highest of two levels; unknown counts as the highest (R4 rule 4).
func maxLevel(a, b string) string {
	rank := map[string]int{"": 0, LevelPatch: 1, LevelMinor: 2, LevelMajor: 3, LevelUnknown: 4}
	if rank[b] > rank[a] {
		return b
	}
	return a
}
