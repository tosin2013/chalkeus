package pyrepr

import "testing"

func TestRepr(t *testing.T) {
	cases := map[string]string{
		Value(nil):                                  "None",
		Value([]any{}):                              "[]",
		Value([]string{"A-1", "B-2"}):               "['A-1', 'B-2']",
		Value([]any{"harness", nil}):                "['harness', None]",
		Value(Tuple{"A", float64(1), "grade_fail"}): "('A', 1, 'grade_fail')",
		Value([]any{Tuple{"A", 2.0, "x"}}):          "[('A', 2, 'x')]",
		Str("it's"):                                 `"it's"`,
		Str(`it's "x"`):                             `'it\'s "x"'`,
		Str("a\nb\\"):                               `'a\nb\\'`,
		Value(true):                                 "True",
		Value(3.5):                                  "3.5",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %s want %s", got, want)
		}
	}
}

func TestJSONDumps(t *testing.T) {
	m := OrderedMap{Keys: []string{"task", "kind", "number", "draft", "author"},
		Values: map[string]any{"task": "A", "kind": nil, "number": float64(7), "draft": true, "author": "é"}}
	got := JSONDumps([]OrderedMap{m})
	want := "[{\"task\": \"A\", \"kind\": null, \"number\": 7, \"draft\": true, \"author\": \"\\u00e9\"}]"
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
	if JSONDumps([]OrderedMap{}) != "[]" {
		t.Error("empty list")
	}
}
