package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/prompt"
)

// Port of the former Python pilot's tests/evidence_builder.py + test_audit.py + test_audit_github.py + test_redaction.py.

type ev = map[string]any

func writeJSON(t *testing.T, p string, v any) {
	t.Helper()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	raw, _ := json.Marshal(v)
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeJSONL(t *testing.T, p string, rows []ev) {
	t.Helper()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	var b strings.Builder
	for _, r := range rows {
		raw, _ := json.Marshal(r)
		b.Write(raw)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readJSONL(t *testing.T, p string) []ev {
	t.Helper()
	raw, _ := os.ReadFile(p)
	var out []ev
	for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var r ev
		_ = json.Unmarshal([]byte(l), &r)
		out = append(out, r)
	}
	return out
}

// build creates a synthetic evidence directory for a correct pilot run.
func build(t *testing.T, d string) string {
	t.Helper()
	fix := filepath.Join("..", "..", "pilot", "fixture", "specs", "001-slugify")
	rd := func(n string) string {
		b, err := os.ReadFile(filepath.Join(fix, n))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	spec, tasks, plan := rd("spec.md"), rd("tasks.md"), rd("plan.md")
	full, _, _ := prompt.Build(spec, tasks, plan, "", "")
	induced, rec, err := prompt.Build(spec, tasks, plan, prompt.Induction, "")
	if err != nil {
		t.Fatal(err)
	}
	var events []ev
	e := func(stage string, task, attempt any, event, detail string) {
		events = append(events, ev{"seq": len(events) + 1, "at": "2026-11-13T00:00:00Z", "stage": stage, "task": task,
			"attempt": attempt, "event": event, "detail": detail})
	}
	e("run", nil, nil, "run_started", "")
	e("gate", "A", nil, "gate_allow", "")
	e("gate", "B", nil, "gate_refuse", "")
	e("mode", nil, nil, "mode_confirmed", "{}")
	e("attempt", "A", 1, "attempt_created", "")
	e("attempt", "A", 1, "attempt_collected", "")
	e("grade", "A", 1, "grade_fail", "G5_hidden_tests")
	e("gate", "B", nil, "gate_refuse", "")
	e("retry", "A", 2, "retry_confirmed", "")
	e("attempt", "A", 2, "attempt_created", "")
	e("attempt", "A", 2, "attempt_collected", "")
	e("grade", "A", 2, "grade_pass", "")
	e("publish", "A", 2, "pr_opened", "u1")
	e("gate", "B", nil, "gate_refuse", "")
	e("run", nil, nil, "run_paused(awaiting-merge)", "")
	e("reconcile", "A", 2, "reconcile_applied", "")
	e("status", "A", nil, "task_verified", "")
	e("gate", "B", nil, "gate_allow", "")
	e("attempt", "B", 1, "attempt_created", "")
	e("attempt", "B", 1, "attempt_collected", "")
	e("grade", "B", 1, "grade_pass", "")
	e("publish", "B", 1, "pr_opened", "u2")
	writeJSONL(t, filepath.Join(d, "events.jsonl"), events)
	writeJSON(t, filepath.Join(d, "run.json"), ev{"run_id": filepath.Base(d), "state": "succeeded"})
	for _, a := range []struct {
		task string
		n    int
		p    string
		ind  *prompt.Record
	}{{"A", 1, induced, rec}, {"A", 2, full, nil}, {"B", 1, full, nil}} {
		ad := filepath.Join(d, "attempts", a.task+"-"+string(rune('0'+a.n)))
		_ = os.MkdirAll(ad, 0o755)
		_ = os.WriteFile(filepath.Join(ad, "prompt.md"), []byte(a.p), 0o644)
		if a.ind != nil {
			writeJSON(t, filepath.Join(ad, "induction.json"), a.ind)
		}
		writeJSON(t, filepath.Join(ad, "report.json"), ev{"spend": ev{"cap_usd": 1.0, "token_cap": 2000000, "cap_mode": "token-fallback",
			"cost_usd": 0.0, "tokens": 180000, "stopped_by_cap": false}, "harness": ev{"exit": 0}})
	}
	writeJSON(t, filepath.Join(d, "prs.json"), []ev{
		{"task": "A", "number": 1, "url": "u1", "draft": true, "author": "bot", "publisher_identity": "bot"},
		{"task": "B", "number": 2, "url": "u2", "draft": true, "author": "bot", "publisher_identity": "bot"}})
	var checks []ev
	for i := 1; i <= 4; i++ {
		checks = append(checks, ev{"id": "GM0" + string(rune('0'+i)), "requirement": "FR-002", "result": "pass", "description": "", "detail": ""})
	}
	for _, pid := range []string{"P04", "P09"} {
		for _, at := range []string{"A-1", "A-2", "B-1"} {
			checks = append(checks, ev{"id": pid, "requirement": "FR-007", "result": "pass", "description": "", "detail": at})
		}
	}
	writeJSONL(t, filepath.Join(d, "checks", "results.jsonl"), checks)
	var td []ev
	for i := 1; i <= 3; i++ {
		td = append(td, ev{"id": "T0" + string(rune('0'+i)), "requirement": "N3", "result": "pass", "description": "", "detail": ""})
	}
	writeJSONL(t, filepath.Join(d, "teardown.jsonl"), td)
	return d
}

func results(t *testing.T, d string) map[string]string {
	t.Helper()
	rows, err := Dir(d, nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, r := range rows {
		out[r.ID] = r.Result
	}
	return out
}

func failed(res map[string]string) []string {
	var f []string
	for k, v := range res {
		if v != "pass" {
			f = append(f, k)
		}
	}
	return f
}

func editEvents(t *testing.T, d string, fn func([]ev) []ev) {
	p := filepath.Join(d, "events.jsonl")
	out := fn(readJSONL(t, p))
	for i, e := range out {
		e["seq"] = i + 1
	}
	writeJSONL(t, p, out)
}

func dropEvents(pred func(ev) bool) func([]ev) []ev {
	return func(in []ev) []ev {
		var out []ev
		for _, e := range in {
			if !pred(e) {
				out = append(out, e)
			}
		}
		return out
	}
}

func TestGoodRunAllPass(t *testing.T) {
	d := build(t, filepath.Join(t.TempDir(), "20261113T000000Z"))
	res := results(t, d)
	if len(res) != 11 || len(failed(res)) != 0 {
		t.Fatalf("got %v", res)
	}
}

func TestEachMutationFailsExactlyItsCheck(t *testing.T) {
	mut := map[string]func(t *testing.T, d string){
		"AU01": func(t *testing.T, d string) {
			editEvents(t, d, func(in []ev) []ev {
				for i, e := range in {
					if e["event"] == "task_verified" {
						x := ev{"stage": "attempt", "task": "B", "attempt": 1, "event": "attempt_created", "detail": "", "at": ""}
						return append(append(append([]ev{}, in[:i]...), x), in[i:]...)
					}
				}
				return in
			})
		},
		"AU02": func(t *testing.T, d string) {
			editEvents(t, d, dropEvents(func(e ev) bool { return e["event"] == "grade_fail" }))
		},
		"AU03": func(t *testing.T, d string) {
			writeJSON(t, filepath.Join(d, "prs.json"), []ev{
				{"task": "A", "number": 1, "draft": true, "author": "bot", "publisher_identity": "bot"},
				{"task": "B", "number": 2, "draft": true, "author": "bot", "publisher_identity": "bot"},
				{"task": "A", "number": 3, "draft": true, "author": "bot", "publisher_identity": "bot"}})
		},
		"AU04": func(t *testing.T, d string) { failCheck(t, d, "P04", "A-2") },
		"AU05": func(t *testing.T, d string) { failCheck(t, d, "GM02", "") },
		"AU06": func(t *testing.T, d string) {
			writeJSON(t, filepath.Join(d, "attempts/A-2/report.json"), ev{"spend": ev{"cap_usd": 1.0, "token_cap": 2000000, "cap_mode": "cost",
				"cost_usd": 1.2, "tokens": 180000, "stopped_by_cap": false}})
		},
		"AU07": func(t *testing.T, d string) {
			p := filepath.Join(d, "teardown.jsonl")
			rows := readJSONL(t, p)
			rows[0]["result"] = "fail"
			writeJSONL(t, p, rows)
		},
		"AU08": func(t *testing.T, d string) {
			_ = os.WriteFile(filepath.Join(d, "attempts/A-2/notes.txt"), []byte("token github"+"_pat_"+strings.Repeat("x", 30)), 0o644)
		},
		"AU09": func(t *testing.T, d string) {
			editEvents(t, d, dropEvents(func(e ev) bool { return e["event"] == "retry_confirmed" }))
		},
		"AU10": func(t *testing.T, d string) {
			editEvents(t, d, func(in []ev) []ev {
				var m ev
				var rest []ev
				for _, e := range in {
					if e["event"] == "mode_confirmed" && m == nil {
						m = e
						continue
					}
					rest = append(rest, e)
				}
				for i, e := range rest {
					if e["stage"] == "attempt" {
						return append(append(append([]ev{}, rest[:i+1]...), m), rest[i+1:]...)
					}
				}
				return in
			})
		},
		"AU11": func(t *testing.T, d string) {
			raw, _ := os.ReadFile(filepath.Join(d, "attempts/A-1/prompt.md"))
			_ = os.WriteFile(filepath.Join(d, "attempts/A-2/prompt.md"), raw, 0o644)
		},
	}
	for au, fn := range mut {
		t.Run(au, func(t *testing.T) {
			d := build(t, filepath.Join(t.TempDir(), "20261113T000000Z"))
			fn(t, d)
			f := failed(results(t, d))
			if len(f) != 1 || f[0] != au {
				t.Errorf("failed %v, want [%s]", f, au)
			}
		})
	}
}

func failCheck(t *testing.T, d, id, detail string) {
	p := filepath.Join(d, "checks", "results.jsonl")
	rows := readJSONL(t, p)
	for _, r := range rows {
		if r["id"] == id && (detail == "" || r["detail"] == detail) {
			r["result"] = "fail"
		}
	}
	writeJSONL(t, p, rows)
}

func TestTeardownPendingWhenNotFinal(t *testing.T) {
	d := build(t, filepath.Join(t.TempDir(), "20261113T000000Z"))
	_ = os.Remove(filepath.Join(d, "teardown.jsonl"))
	rows, _ := Dir(d, nil, nil, false)
	for _, r := range rows {
		if r.ID == "AU07" {
			t.Fatal("AU07 present when not final")
		}
	}
	if results(t, d)["AU07"] != "fail" {
		t.Fatal("AU07 should fail when final without teardown")
	}
}

func addApprovalEvents(t *testing.T, d string, details ...string) {
	editEvents(t, d, func(in []ev) []ev {
		for _, det := range details {
			in = append(in, ev{"at": "t", "stage": "approval", "task": nil, "attempt": nil, "event": "approval_channel", "detail": det})
		}
		return in
	})
}

func TestGitHubApproversInDetail(t *testing.T) {
	d := build(t, filepath.Join(t.TempDir(), "20261116T000000Z"))
	addApprovalEvents(t, d, "step=mode approved=true channel=github approver=alice comment=11",
		"step=retry approved=true channel=github approver=alice comment=12", "step=retry approved=false channel=none")
	rows, _ := Dir(d, nil, nil, true)
	for _, r := range rows {
		if (r.ID == "AU09" || r.ID == "AU10") && !strings.Contains(r.Detail, "channels=['github'] approvers=['alice']") {
			t.Errorf("%s detail %q", r.ID, r.Detail)
		}
	}
}

func TestAU03StatusPRs(t *testing.T) {
	base := []ev{
		{"task": "A", "number": 1, "draft": true, "author": "bot", "publisher_identity": "bot"},
		{"task": "B", "number": 2, "draft": true, "author": "bot", "publisher_identity": "bot"}}
	status := func(extra ev) ev {
		m := ev{"task": "A", "kind": "status", "number": 99, "draft": true, "author": "bot", "publisher_identity": "bot"}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	cases := map[string]struct {
		prs  []ev
		want string
	}{
		"one-status":   {append(append([]ev{}, base...), status(nil)), "pass"},
		"two-status":   {append(append([]ev{}, base...), status(nil), status(ev{"number": 98})), "fail"},
		"non-draft":    {append(append([]ev{}, base...), status(ev{"draft": false})), "fail"},
		"unknown-kind": {append(append([]ev{}, base...), status(ev{"kind": "other"})), "fail"},
	}
	for name, c := range cases {
		d := build(t, filepath.Join(t.TempDir(), "x"))
		writeJSON(t, filepath.Join(d, "prs.json"), c.prs)
		if got := results(t, d)["AU03"]; got != c.want {
			t.Errorf("%s: AU03 %s, want %s", name, got, c.want)
		}
	}
}

func TestRedactionScan(t *testing.T) {
	d := t.TempDir()
	keyf := filepath.Join(t.TempDir(), "key")
	envf := filepath.Join(t.TempDir(), "publisher.env")
	_ = os.WriteFile(keyf, []byte("SECRETKEYVALUE\n"), 0o600)
	_ = os.WriteFile(envf, []byte("# comment=IGNORED\nPILOT_SCM_TOKEN=tokval123\nEMPTY=\n"), 0o600)
	files := map[string]string{
		"clean.txt": "nothing here IGNORED",
		"key.txt":   "has SECRETKEYVALUE inside",
		"env.txt":   "x tokval123 y",
		"pem.txt":   "-----BEGIN RSA PRIVATE KEY-----",
		"sub/pat":   "ghp_" + strings.Repeat("a", 20),
	}
	for n, c := range files {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(d, n)), 0o755)
		_ = os.WriteFile(filepath.Join(d, n), []byte(c), 0o644)
	}
	hits, err := RedactionScan(d, []string{keyf}, []string{envf})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"env.txt", "key.txt", "pem.txt", "sub/pat"}
	if strings.Join(hits, ",") != strings.Join(want, ",") {
		t.Fatalf("hits %v, want %v", hits, want)
	}
}

// 004 App mode: app-token.env's kind ("app") and bot login are public; only the token is secret.
func TestRedactionScanAppTokenFile(t *testing.T) {
	d := t.TempDir()
	env := filepath.Join(t.TempDir(), "app-token.env")
	_ = os.WriteFile(env, []byte("PILOT_SCM_TOKEN=Tok3n-value-xyz-987\nPILOT_SCM_KIND=app\nPILOT_SCM_LOGIN=demo-pilot[bot]\n"), 0o600)
	_ = os.WriteFile(filepath.Join(d, "events.jsonl"), []byte("approved by app; PR by demo-pilot[bot]\n"), 0o644)
	if hits, _ := RedactionScan(d, nil, []string{env}); len(hits) != 0 {
		t.Fatalf("public values counted as secrets: %v", hits)
	}
	_ = os.WriteFile(filepath.Join(d, "leak.txt"), []byte("x Tok3n-value-xyz-987 y\n"), 0o644)
	if hits, _ := RedactionScan(d, nil, []string{env}); strings.Join(hits, ",") != "leak.txt" {
		t.Fatalf("token not found: %v", hits)
	}
}

// 025: a test-mode run gets a failing AU12; real records get no AU12 row.
func TestAU12TestRun(t *testing.T) {
	d := build(t, filepath.Join(t.TempDir(), "20261113T000000Z"))
	var cfg map[string]any
	raw, _ := os.ReadFile(filepath.Join(d, "config.json"))
	_ = json.Unmarshal(raw, &cfg)
	if cfg == nil {
		cfg = map[string]any{}
	}
	cfg["test_mode"], cfg["test_github_api"] = true, "http://172.18.0.1:8090"
	b, _ := json.Marshal(cfg)
	_ = os.WriteFile(filepath.Join(d, "config.json"), b, 0o644)
	res := results(t, d)
	if len(res) != 12 || strings.Join(failed(res), ",") != "AU12" {
		t.Fatalf("got %v", res)
	}
}
