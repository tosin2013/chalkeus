package reconcile

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Port of pilot/tests/test_reconcile.py, plus the shared status-PR vectors (016 FR-011).

func roadmap(t *testing.T) string {
	raw, err := os.ReadFile("../../pilot/fixture/.specify/memory/roadmap.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

var paths2 = []string{"src/slugkit/slug.py", "specs/001-slugify/tasks.md"}

func make1(t *testing.T, merged bool, percent any, equal bool, answer string) (Reconciler, *int, *string) {
	pushed, text := 0, ""
	head := map[string][]byte{"src/slugkit/slug.py": []byte("v1"), "specs/001-slugify/tasks.md": []byte("done")}
	rm := roadmap(t)
	return Reconciler{
		PRView: func(int) (map[string]any, error) {
			if merged {
				return map[string]any{"state": "MERGED", "mergedAt": "2026-11-13T00:00:00Z", "mergeCommit": map[string]any{"oid": "abc"}}, nil
			}
			return map[string]any{"state": "OPEN", "mergedAt": nil, "mergeCommit": nil}, nil
		},
		TasksPercent: func(string) any { return percent },
		DefaultFile: func(p string) []byte {
			if equal {
				return head[p]
			}
			return []byte("edited after grading")
		},
		GradedFile:  func(p string) []byte { return head[p] },
		RoadmapText: func() string { return rm },
		Push:        func(s string) (string, error) { pushed++; text = s; return "def456", nil },
		Ask:         func(string) string { return answer },
		Say:         func(string) {},
	}, &pushed, &text
}

func TestHappyPathAppliesOneLine(t *testing.T) {
	r, pushed, text := make1(t, true, 100, true, "y")
	code, rec := r.Run("A", "001", 7, paths2)
	if code != 0 || *pushed != 1 || rec["applied_commit"] != "def456" || rec["approved"] != true {
		t.Fatalf("%d %v", code, rec)
	}
	old, nl := strings.Split(roadmap(t), "\n"), strings.Split(*text, "\n")
	diff := 0
	for i := range old {
		if old[i] != nl[i] {
			diff++
			if !strings.Contains(nl[i], "### 001") || !strings.Contains(nl[i], "[status: verified]") {
				t.Errorf("wrong line changed: %s", nl[i])
			}
		}
	}
	if diff != 1 {
		t.Fatalf("%d lines changed", diff)
	}
}

func TestRefusals(t *testing.T) {
	cases := []struct {
		merged  bool
		percent any
		equal   bool
		answer  string
		code    int
		reason  string
	}{
		{false, 100, true, "y", 4, "NOT_MERGED"},
		{true, 50, true, "y", 1, "TASKS_INCOMPLETE"},
		{true, 100, false, "y", 1, "TREE_MISMATCH"},
		{true, 100, true, "n", 5, "DECLINED"},
	}
	for _, c := range cases {
		r, pushed, _ := make1(t, c.merged, c.percent, c.equal, c.answer)
		code, rec := r.Run("A", "001", 7, paths2)
		if code != c.code || rec["reason"] != c.reason || *pushed != 0 {
			t.Errorf("%+v: %d %v pushed=%d", c, code, rec["reason"], *pushed)
		}
	}
}

func TestStatusEditNeedsExactlyOnePlannedHeading(t *testing.T) {
	rm := roadmap(t)
	if _, err := StatusEdit(strings.Replace(rm, "[status: planned]", "[status: specced]", 1), "001"); err == nil {
		t.Error("no planned heading must fail")
	}
	if _, err := StatusEdit(rm, "009"); err == nil {
		t.Error("unknown item must fail")
	}
}

func TestStatusPRVectors(t *testing.T) {
	raw, err := os.ReadFile("../../specs/004-github-loop/contracts/vectors/status_pr.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Cases []struct {
			Name  string
			Input struct {
				State     string   `json:"state"`
				BaseText  string   `json:"base_text"`
				HeadText  string   `json:"head_text"`
				Item      string   `json:"item"`
				Publisher string   `json:"publisher"`
				MergedBy  *string  `json:"merged_by"`
				Changed   []string `json:"changed"`
				Approvers []string `json:"approvers"`
			}
			Expect struct {
				OK     bool    `json:"ok"`
				Reason *string `json:"reason"`
			}
		}
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, c := range doc.Cases {
		i := c.Input
		var mergedBy string
		if i.MergedBy != nil {
			mergedBy = *i.MergedBy
		}
		ok, reason := CheckStatusPR(i.State, mergedBy, i.BaseText, i.HeadText, i.Changed, i.Item, i.Approvers, i.Publisher)
		want := ""
		if c.Expect.Reason != nil {
			want = *c.Expect.Reason
		}
		if ok != c.Expect.OK || reason != want {
			t.Errorf("%s: got %v %q, want %v %q", c.Name, ok, reason, c.Expect.OK, want)
		}
	}
}
