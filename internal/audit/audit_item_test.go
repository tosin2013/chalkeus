package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/prompt"
)

const itemSpec = "# Shout\n\n## Allowed paths\n- `app.py`\n"

// itemRecord writes a synthetic, passing item-run record (014 contracts/audit-item.md).
func itemRecord(t *testing.T, mutate func(d string, ev *[]map[string]any, prs *[]map[string]any)) string {
	t.Helper()
	d := t.TempDir()
	for _, sub := range []string{"gate", "attempts/002-1", "attempts/002-2", "checks"} {
		_ = os.MkdirAll(filepath.Join(d, sub), 0o755)
	}
	wj := func(p string, v any) {
		if err := evidence.WriteJSON(filepath.Join(d, p), v); err != nil {
			t.Fatal(err)
		}
	}
	wj("run.json", map[string]any{"run_id": "r", "kind": "item", "item": "002", "state": "succeeded",
		"target": map[string]any{"name": "g", "repo": "me/greeting-service"}})
	wj("gate/1-002.readiness.json", map[string]any{"decision": "allow", "rules": []any{
		map[string]any{"id": "gate", "result": "pass"}, map[string]any{"id": "hidden_tests", "result": "pass"}}})
	text, _ := prompt.BuildItem(prompt.Item{Spec: itemSpec, Tasks: "- [ ] T1\n", TasksPath: "specs/002/tasks.md",
		AllowedPaths: []string{"app.py"}, Toolchain: "python", TestCommand: "make test"})
	for _, n := range []string{"1", "2"} {
		wj("attempts/002-"+n+"/meta.json", map[string]any{"induction": nil, "spec_sha256": prompt.SpecSHA256(itemSpec)})
		_ = os.WriteFile(filepath.Join(d, "attempts/002-"+n, "prompt.md"), []byte(text), 0o644)
	}
	ev := []map[string]any{
		{"stage": "gate", "task": "002", "event": "gate_allow"},
		{"stage": "baseline", "task": "002", "event": "baseline_pass"},
		{"stage": "attempt", "task": "002", "attempt": 1, "event": "attempt_started"},
		{"stage": "grade", "task": "002", "attempt": 1, "event": "grade_fail"},
		{"stage": "attempt", "task": "002", "attempt": 2, "event": "attempt_started"},
		{"stage": "grade", "task": "002", "attempt": 2, "event": "grade_pass"},
	}
	prs := []map[string]any{
		{"kind": "task", "task": "002", "draft": true, "author": "bot[bot]", "publisher_identity": "bot[bot]", "url": "https://github.com/me/greeting-service/pull/5"},
		{"kind": "status", "task": "002", "draft": true, "author": "bot[bot]", "publisher_identity": "bot[bot]", "url": "https://github.com/me/greeting-service/pull/6"},
	}
	if mutate != nil {
		mutate(d, &ev, &prs)
	}
	var lines []byte
	for _, e := range ev {
		raw, _ := json.Marshal(e)
		lines = append(append(lines, raw...), '\n')
	}
	_ = os.WriteFile(filepath.Join(d, "events.jsonl"), lines, 0o644)
	_ = os.WriteFile(filepath.Join(d, "checks/results.jsonl"), nil, 0o644)
	wj("prs.json", prs)
	return d
}

func itemResults(t *testing.T, d string) map[string]string {
	t.Helper()
	rows, err := Dir(d, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, r := range rows {
		out[r.ID] = r.Result + " " + r.Detail
	}
	return out
}

func TestItemVariantsPass(t *testing.T) {
	r := itemResults(t, itemRecord(t, nil))
	for _, id := range []string{"AU01", "AU02", "AU03", "AU11"} {
		if r[id][:4] != "pass" {
			t.Errorf("%s: %s", id, r[id])
		}
	}
}

func TestItemVariantsFail(t *testing.T) {
	cases := map[string]struct {
		id     string
		mutate func(d string, ev *[]map[string]any, prs *[]map[string]any)
	}{
		"another item allowed": {"AU01", func(_ string, ev *[]map[string]any, _ *[]map[string]any) {
			*ev = append(*ev, map[string]any{"stage": "gate", "task": "003", "event": "gate_allow"})
		}},
		"another item attempted": {"AU01", func(_ string, ev *[]map[string]any, _ *[]map[string]any) {
			*ev = append(*ev, map[string]any{"stage": "attempt", "task": "003", "event": "attempt_started"})
		}},
		"readiness refused": {"AU01", func(d string, _ *[]map[string]any, _ *[]map[string]any) {
			_ = evidence.WriteJSON(filepath.Join(d, "gate/1-002.readiness.json"), map[string]any{"decision": "refuse"})
		}},
		"no baseline": {"AU02", func(_ string, ev *[]map[string]any, _ *[]map[string]any) { *ev = append((*ev)[:1], (*ev)[2:]...) }},
		"two passes": {"AU02", func(_ string, ev *[]map[string]any, _ *[]map[string]any) {
			*ev = append(*ev, map[string]any{"stage": "grade", "task": "002", "attempt": 3, "event": "grade_pass"})
		}},
		"last attempt failed": {"AU02", func(_ string, ev *[]map[string]any, _ *[]map[string]any) {
			*ev = append(*ev, map[string]any{"stage": "grade", "task": "002", "attempt": 3, "event": "grade_fail"})
		}},
		"two task PRs": {"AU03", func(_ string, _ *[]map[string]any, prs *[]map[string]any) { *prs = append(*prs, (*prs)[0]) }},
		"PR on another repo": {"AU03", func(_ string, _ *[]map[string]any, prs *[]map[string]any) {
			(*prs)[0]["url"] = "https://github.com/me/other/pull/5"
		}},
		"not a draft": {"AU03", func(_ string, _ *[]map[string]any, prs *[]map[string]any) { (*prs)[1]["draft"] = false }},
		"induced": {"AU11", func(d string, _ *[]map[string]any, _ *[]map[string]any) {
			_ = evidence.WriteJSON(filepath.Join(d, "attempts/002-2/meta.json"), map[string]any{"induction": map[string]any{"name": "x"}, "spec_sha256": prompt.SpecSHA256(itemSpec)})
		}},
		"spec altered in prompt": {"AU11", func(d string, _ *[]map[string]any, _ *[]map[string]any) {
			p := filepath.Join(d, "attempts/002-1/prompt.md")
			_ = os.WriteFile(p, []byte("intro\n# Specification\n\nsomething else\n"), 0o644)
		}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if r := itemResults(t, itemRecord(t, c.mutate)); r[c.id][:4] != "fail" {
				t.Fatalf("%s should fail: %s", c.id, r[c.id])
			}
		})
	}
}
