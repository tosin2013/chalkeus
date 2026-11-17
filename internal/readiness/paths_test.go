package readiness

import (
	"reflect"
	"testing"
)

const spec = "# Feature\n\n## Allowed paths\n\n- `src/main/java/com/example/*` (the service)\n* `src/test/java/com/example/*`\n- not a glob\n\n## Requirements\n- `README.md` is outside the section\n"

func TestAllowedPathsParsed(t *testing.T) {
	got, err := AllowedPaths(spec, "specs/003-x", []string{"specs/001-a", "specs/002-b"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"src/main/java/com/example/*", "src/test/java/com/example/*", "specs/003-x/tasks.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
}

func TestAllowedPathsRefusals(t *testing.T) {
	cases := map[string]string{
		"# F\n\nno section\n":                         "ITEM_NO_ALLOWED_PATHS",
		"## Allowed paths\n\nTBD\n":                   "ITEM_NO_ALLOWED_PATHS",
		"## Allowed paths\n- `*`\n":                   "ITEM_PATHS_TOO_WIDE",
		"## Allowed paths\n- `.github/*`\n":           "ITEM_PATHS_TOO_WIDE",
		"## Allowed paths\n- `.specify/memory/*`\n":   "ITEM_PATHS_TOO_WIDE",
		"## Allowed paths\n- `specs/*`\n":             "ITEM_PATHS_TOO_WIDE",
		"## Allowed paths\n- `specs/001-a/spec.md`\n": "ITEM_PATHS_TOO_WIDE",
		"## Allowed paths\n- `../outside/*`\n":        "ITEM_PATHS_TOO_WIDE",
		"## Allowed paths\n- `/etc/*`\n":              "ITEM_PATHS_TOO_WIDE",
	}
	for text, code := range cases {
		_, err := AllowedPaths(text, "specs/003-x", []string{"specs/001-a"})
		r, ok := err.(*Refusal)
		if !ok || r.Code != code {
			t.Errorf("%q: want %s, got %v", text, code, err)
		}
	}
}

// Match must agree with the grader's Python fnmatch (G2), where "*" crosses "/".
func TestMatchIsFnmatch(t *testing.T) {
	for _, c := range []struct {
		glob, name string
		want       bool
	}{
		{"src/slugkit/*", "src/slugkit/slug.py", true},
		{"src/slugkit/*", "src/slugkit/sub/deep.py", true},
		{"tests/*", "tests/test_x.py", true},
		{"tests/*", "src/tests/x.py", false},
		{"src/?.py", "src/a.py", true},
		{"src/[ab].py", "src/b.py", true},
		{"src/[!ab].py", "src/b.py", false},
		{"a.b", "axb", false},
		{"*.md", "docs/x.md", true},
	} {
		if got := Match(c.glob, c.name); got != c.want {
			t.Errorf("Match(%q, %q) = %v", c.glob, c.name, got)
		}
	}
}

func TestTestsMayChange(t *testing.T) {
	spec := "# X\n\n## Allowed paths\n- `src/*`\n\n## Tests that may change\n\n- `tests/test_old.py` (its expectation changes)\n\n## Notes\n- `not/this`\n"
	if got := TestsMayChange(spec); len(got) != 1 || got[0] != "tests/test_old.py" {
		t.Fatal(got)
	}
	if got := TestsMayChange("# X\n"); len(got) != 0 {
		t.Fatal(got)
	}
}
