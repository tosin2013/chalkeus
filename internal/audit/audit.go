// Package audit is the evidence-only audit AU01–AU11 and the redaction scan
// (reference: the former Python pilot's audit.py). Detail strings match the reference byte for
// byte; specs/016-go-pilot/golden/audit-expected.json is the proof.
package audit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tosin2013/chalkeus/internal/scm"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/prompt"
	"github.com/tosin2013/chalkeus/internal/pyrepr"
)

// Patterns are the token-like strings that must never appear in evidence.
// The token prefixes must start a word: "risk-assessment-YYYY-MM-DD" contains "sk-" followed by
// 16 characters and was flagged in a target's docs (documcp run 20261125T223435Z). Most evidence
// is JSON, where a newline or tab inside a string is written \n or \t: a token after one still
// starts a word, though the character before it is a letter.
var Patterns = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----|(?:^|[^A-Za-z0-9]|\\[nrt])(?:github_pat_[A-Za-z0-9_]{10,}|gh[pousr]_[A-Za-z0-9]{16,}|glpat-[A-Za-z0-9_-]{20,}|sk-[A-Za-z0-9_-]{16,})|Bearer [A-Za-z0-9._-]{16,}`)

// nonSecretEnvKeys are the App token file's metadata (app-token.env): its kind ("app") and the
// bot login are public, and "app" would match nearly every file. Every other value is a secret.
var nonSecretEnvKeys = map[string]bool{"PILOT_SCM_KIND": true, "PILOT_SCM_LOGIN": true}

// RedactionScan returns the files under root (relative, sorted) that contain a token-like
// pattern, any line of a key file, or any value from an env file (KEY=VALUE, non-comment).
func RedactionScan(root string, keyFiles, envFiles []string) ([]string, error) {
	var fixed [][]byte
	for _, kf := range keyFiles {
		raw, err := os.ReadFile(kf)
		if err != nil || len(raw) == 0 {
			continue
		}
		for _, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
			fixed = append(fixed, []byte(line)) // grep -F -f: every line is a pattern
		}
	}
	for _, ef := range envFiles {
		raw, err := os.ReadFile(ef)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") || !strings.Contains(line, "=") {
				continue
			}
			k, v, _ := strings.Cut(line, "=")
			if nonSecretEnvKeys[strings.TrimSpace(k)] {
				continue
			}
			if v != "" {
				fixed = append(fixed, []byte(v))
			}
		}
	}
	var hits []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".graded-") && p != root {
			// reconcile's temporary clone of the target (base + the passing change): the target's
			// own files, not evidence, and removed after the run.
			return filepath.SkipDir
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if matches(raw, fixed) {
			rel, _ := filepath.Rel(root, p)
			hits = append(hits, rel)
		}
		return nil
	})
	sort.Strings(hits)
	return hits, err
}

func matches(raw []byte, fixed [][]byte) bool {
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<28)
	for sc.Scan() {
		line := sc.Bytes()
		if Patterns.Match(line) {
			return true
		}
		for _, f := range fixed {
			if bytes.Contains(line, f) {
				return true
			}
		}
	}
	return false
}

// kv parses "a=b c=d" detail strings; later keys win, as in the reference's dict().
func kv(detail string) map[string]string {
	out := map[string]string{}
	for _, f := range strings.Fields(detail) {
		if k, v, ok := strings.Cut(f, "="); ok {
			out[k] = v
		}
	}
	return out
}

func str(e evidence.Row, k string) string { s, _ := e[k].(string); return s }

// approvals returns step -> values of `field` for approved approval_channel events; when
// githubOnly, only channel=github events count (the reference _approvers).
func approvals(ev []evidence.Row, field string, githubOnly bool) (map[string][]any, []string) {
	out := map[string][]any{}
	var order []string
	for _, e := range ev {
		d := str(e, "detail")
		if str(e, "event") != "approval_channel" || !strings.Contains(d, "approved=true") {
			continue
		}
		m := kv(d)
		if githubOnly && m["channel"] != "github" {
			continue
		}
		step := m["step"]
		var val any
		if v, ok := m[field]; ok {
			val = v
		}
		if _, seen := out[step]; !seen {
			order = append(order, step)
		}
		out[step] = append(out[step], val)
	}
	return out, order
}

func list(m map[string][]any, step string) string {
	v, ok := m[step]
	if !ok {
		return "[]"
	}
	return pyrepr.Value(v)
}

func ghSuffix(m map[string][]any, step string) string {
	if len(m[step]) == 0 {
		return ""
	}
	return " approvers=" + pyrepr.Value(m[step])
}

type attempt struct {
	task string
	n    int
	dir  string
}

func attempts(d string) []attempt {
	matches, _ := filepath.Glob(filepath.Join(d, "attempts", "*-*"))
	sort.Strings(matches)
	var out []attempt
	for _, a := range matches {
		task, n, _ := strings.Cut(filepath.Base(a), "-")
		i, err := strconv.Atoi(n)
		if err != nil {
			continue
		}
		out = append(out, attempt{task, i, a})
	}
	return out
}

// NA is the result of a check that doesn't apply to the run (033: AU02 and AU03 on a run the owner
// ended for planning). It counts neither as a pass nor as a failure.
const NA = "n/a"

func row(id string, ok bool, desc, detail string) evidence.CheckRow {
	r := "fail"
	if ok {
		r = "pass"
	}
	return evidence.NewCheckRow(id, "audit", r, desc, detail)
}

func readJSON(path string, v any) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return evidence.Decode(raw, v) == nil
}

func num(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	}
	return 0, false
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case json.Number:
		f, _ := x.Float64()
		return f != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

func intRepr(i int, ok bool) string {
	if !ok {
		return "None"
	}
	return strconv.Itoa(i)
}

// Dir audits one evidence directory. keyFiles/envFiles feed the redaction scan; final adds
// AU07 even when no teardown has been recorded.
func Dir(d string, keyFiles, envFiles []string, final bool) ([]evidence.CheckRow, error) {
	ev, err := evidence.ReadJSONL(filepath.Join(d, "events.jsonl"))
	if err != nil {
		return nil, err
	}
	checks, err := evidence.ReadJSONL(filepath.Join(d, "checks", "results.jsonl"))
	if err != nil {
		return nil, err
	}
	idx := func(pred func(evidence.Row) bool) (int, bool) {
		for i, e := range ev {
			if pred(e) {
				return i, true
			}
		}
		return 0, false
	}
	var rows []evidence.CheckRow

	// AU01: B never started (or allowed) before A verified; at least one refusal of B.
	cut := len(ev)
	if i, ok := idx(func(e evidence.Row) bool { return str(e, "event") == "task_verified" && str(e, "task") == "A" }); ok {
		cut = i
	}
	var bAtt, bAllow, bRef int
	for _, e := range ev[:cut] {
		if str(e, "task") != "B" {
			continue
		}
		if str(e, "stage") == "attempt" {
			bAtt++
		}
		switch str(e, "event") {
		case "gate_allow":
			bAllow++
		case "gate_refuse":
			bRef++
		}
	}
	rows = append(rows, row("AU01", bAtt == 0 && bAllow == 0 && bRef > 0,
		"B refused (and never attempted) until A verified (SC-001)",
		fmt.Sprintf("refusals=%d attempts_before=%d allows_before=%d", bRef, bAtt, bAllow)))

	// AU02: A/1 fail -> A/n>=2 pass -> B pass, in order.
	var grades []any
	type g struct {
		task string
		n    float64
		ev   string
	}
	var gs []g
	for _, e := range ev {
		evn := str(e, "event")
		if str(e, "stage") == "grade" && (evn == "grade_pass" || evn == "grade_fail") {
			grades = append(grades, pyrepr.Tuple{e["task"], e["attempt"], evn})
			n, _ := num(e["attempt"])
			gs = append(gs, g{str(e, "task"), n, evn})
		}
	}
	ok2 := false
	if len(gs) > 0 && gs[0].task == "A" && gs[0].n == 1 && gs[0].ev == "grade_fail" {
		aPass, bPass := -1, -1
		for i, x := range gs {
			if aPass < 0 && x.task == "A" && x.ev == "grade_pass" && x.n >= 2 {
				aPass = i
			}
			if bPass < 0 && x.task == "B" && x.ev == "grade_pass" {
				bPass = i
			}
		}
		ok2 = aPass >= 0 && bPass >= 0 && aPass < bPass
	}
	if grades == nil {
		grades = []any{}
	}
	rows = append(rows, row("AU02", ok2, "order A/1 fail -> A/2+ pass -> B pass (SC-002)", pyrepr.Value(grades)))

	// AU03: exactly 2 task PRs (A, B), plus 0 or 1 status-change PR per item (004 R7); every PR a
	// draft authored by the publisher identity. A missing `kind` means `task` (older records).
	var prs []map[string]any
	readJSON(filepath.Join(d, "prs.json"), &prs)
	var taskTasks, perItem []string
	nTask, nStatus := 0, 0
	allPerItemTruthy := true
	allOK := true
	detail := make([]pyrepr.OrderedMap, 0, len(prs))
	keys := []string{"task", "kind", "number", "draft", "author", "publisher_identity"}
	for _, p := range prs {
		kind := "task"
		if k, ok := p["kind"]; ok {
			kind, _ = k.(string)
		}
		switch kind {
		case "task":
			nTask++
			t, _ := p["task"].(string)
			taskTasks = append(taskTasks, t)
		case "status":
			nStatus++
			t, _ := p["task"].(string)
			perItem = append(perItem, t)
			if !truthy(p["task"]) {
				allPerItemTruthy = false
			}
		}
		draft, isBool := p["draft"].(bool)
		if !isBool || !draft || !truthy(p["author"]) || fmt.Sprint(p["author"]) != fmt.Sprint(p["publisher_identity"]) {
			allOK = false
		}
		m := pyrepr.OrderedMap{Keys: keys, Values: map[string]any{}}
		for _, k := range keys {
			m.Values[k] = p[k]
		}
		detail = append(detail, m)
	}
	sort.Strings(taskTasks)
	uniq := map[string]bool{}
	for _, t := range perItem {
		uniq[t] = true
	}
	ok3 := nTask == 2 && len(taskTasks) == 2 && taskTasks[0] == "A" && taskTasks[1] == "B" &&
		len(uniq) == len(perItem) && allPerItemTruthy && nTask+nStatus == len(prs) && allOK
	rows = append(rows, row("AU03", ok3,
		"exactly 2 draft task PRs (plus at most 1 status PR per item), all by the publisher identity (SC-003)",
		pyrepr.JSONDumps(detail)))

	// AU04: credential checks present and passing. 037: P13 (the publisher reaches only its Git
	// host) must pass too when the run recorded it; runs from before 037 have none.
	var p04, p09, cred int
	credPass := true
	for _, c := range checks {
		switch str(c, "id") {
		case "P04":
			p04++
		case "P09":
			p09++
		case "P13":
		default:
			continue
		}
		cred++
		if str(c, "result") != "pass" {
			credPass = false
		}
	}
	rows = append(rows, row("AU04", p04 > 0 && p09 > 0 && credPass,
		"no worker/grader held an SCM credential (P04, P09) (SC-004)", fmt.Sprintf("rows=%d", cred)))

	// AU05: gate fail-closed matrix.
	gm, gmPass := 0, true
	for _, c := range checks {
		if strings.HasPrefix(str(c, "id"), "GM") {
			gm++
			if str(c, "result") != "pass" {
				gmPass = false
			}
		}
	}
	rows = append(rows, row("AU05", gm > 0 && gmPass, "gate fail-closed matrix passed (SC-005)", fmt.Sprintf("rows=%d", gm)))

	// AU06: spend within cap per attempt.
	bad6 := []string{}
	for _, a := range attempts(d) {
		var rep map[string]any
		if !readJSON(filepath.Join(a.dir, "report.json"), &rep) {
			continue
		}
		sp, _ := rep["spend"].(map[string]any)
		if sp == nil {
			continue // harness never ran
		}
		var within bool
		if str(sp, "cap_mode") == "cost" {
			c, _ := num(sp["cost_usd"])
			cp, _ := num(sp["cap_usd"])
			within = c <= cp
		} else {
			t, _ := num(sp["tokens"])
			tc, _ := num(sp["token_cap"])
			within = t <= tc
		}
		if !within && !truthy(sp["stopped_by_cap"]) {
			bad6 = append(bad6, fmt.Sprintf("%s-%d", a.task, a.n))
		}
	}
	rows = append(rows, row("AU06", len(bad6) == 0, "every attempt within its spend cap (SC-008)", "over="+pyrepr.Strings(bad6)))

	// AU07: teardown verified.
	td, err := evidence.ReadJSONL(filepath.Join(d, "teardown.jsonl"))
	if err != nil {
		return nil, err
	}
	if final || len(td) > 0 {
		tdPass := len(td) > 0
		for _, t := range td {
			if str(t, "result") != "pass" {
				tdPass = false
			}
		}
		rows = append(rows, row("AU07", tdPass, "teardown verified (SC-006)", fmt.Sprintf("rows=%d", len(td))))
	}

	// AU08: redaction scan.
	hits, err := RedactionScan(d, keyFiles, envFiles)
	if err != nil {
		return nil, err
	}
	if hits == nil {
		hits = []string{}
	}
	rows = append(rows, row("AU08", len(hits) == 0, "no secrets in the evidence record", "hits="+pyrepr.Strings(hits)))

	// AU09: every retry was confirmed by the operator.
	bad9 := []string{}
	for i, e := range ev {
		if str(e, "stage") != "attempt" || str(e, "event") != "attempt_created" {
			continue
		}
		n := 1.0
		if x, ok := num(e["attempt"]); ok && x != 0 {
			n = x
		}
		if n <= 1 {
			continue
		}
		found := false
		for _, x := range ev[:i] {
			if str(x, "event") == "retry_confirmed" && fmt.Sprint(x["task"]) == fmt.Sprint(e["task"]) && fmt.Sprint(x["attempt"]) == fmt.Sprint(e["attempt"]) {
				found = true
				break
			}
		}
		if !found {
			bad9 = append(bad9, fmt.Sprintf("%v-%s", e["task"], pyrepr.Value(e["attempt"])))
		}
	}
	channels, _ := approvals(ev, "channel", false)
	approvers, _ := approvals(ev, "approver", true)
	rows = append(rows, row("AU09", len(bad9) == 0, "every retry confirmed by the operator (FR-017)",
		fmt.Sprintf("unconfirmed=%s channels=%s%s", pyrepr.Strings(bad9), list(channels, "retry"), ghSuffix(approvers, "retry"))))

	// AU10: mode confirmed before the first attempt.
	m, mok := idx(func(e evidence.Row) bool { return str(e, "event") == "mode_confirmed" })
	a, aok := idx(func(e evidence.Row) bool { return str(e, "stage") == "attempt" })
	rows = append(rows, row("AU10", mok && (!aok || m < a), "mode confirmed before any worker staged (FR-023)",
		fmt.Sprintf("mode_seq=%s first_attempt=%s channels=%s%s", intRepr(m, mok), intRepr(a, aok), list(channels, "mode"), ghSuffix(approvers, "mode"))))

	// AU11: induction only on A/1, and it withheld exactly the acceptance section.
	bad11 := []string{}
	a2sha := ""
	for _, at := range attempts(d) {
		raw, err := os.ReadFile(filepath.Join(at.dir, "prompt.md"))
		_, indErr := os.Stat(filepath.Join(at.dir, "induction.json"))
		induced := indErr == nil
		if err != nil {
			bad11 = append(bad11, fmt.Sprintf("%s-%d:no prompt", at.task, at.n))
			continue
		}
		p := string(raw)
		if at.task == "A" && at.n == 1 {
			if !induced || strings.Contains(p, prompt.Heading) {
				bad11 = append(bad11, "A-1:not induced")
			}
		} else if induced || !strings.Contains(p, prompt.Heading) {
			bad11 = append(bad11, fmt.Sprintf("%s-%d:induced or missing section", at.task, at.n))
		} else if at.task == "A" && at.n == 2 {
			a2sha, _ = prompt.SectionSHA256(p)
		}
	}
	var ind map[string]any
	if readJSON(filepath.Join(d, "attempts", "A-1", "induction.json"), &ind) && a2sha != "" {
		if s, _ := ind["removed_sha256"].(string); s != a2sha {
			bad11 = append(bad11, "A-1 removed span != A-2 section")
		}
	}
	rows = append(rows, row("AU11", len(bad11) == 0, "induction applied to A/1 only, withholding exactly the acceptance section", pyrepr.Strings(bad11)))
	if runKind(d) == evidence.KindItem {
		rows = itemVariants(d, rows)
	}
	// AU12 (025): a run made in test mode (fake GitHub, stub harness) is never evidence. Records
	// without test_mode (every real one) get no AU12 row, so their audit is unchanged.
	var cfg map[string]any
	if readJSON(filepath.Join(d, "config.json"), &cfg) && cfg["test_mode"] == true {
		rows = append(rows, row("AU12", false, "not a test run (integration test mode: fake GitHub, stub harness)",
			fmt.Sprintf("test_github_api=%v test_git_base=%v", cfg["test_github_api"], cfg["test_git_base"])))
	}
	return rows, nil
}

// runKind reads run.json's kind; records without one (every pre-014 record) are fixture runs.
func runKind(d string) string {
	var st map[string]any
	if !readJSON(filepath.Join(d, "run.json"), &st) {
		return evidence.KindFixture
	}
	return (&evidence.Run{State: st}).Kind()
}

// itemVariants replaces the fixture-shaped checks (AU01, AU02, AU03, AU11) with their item-run
// variants (014 specs/014-any-project/contracts/audit-item.md). The other checks are shared.
func itemVariants(d string, rows []evidence.CheckRow) []evidence.CheckRow {
	var st map[string]any
	readJSON(filepath.Join(d, "run.json"), &st)
	item, _ := st["item"].(string)
	ev, _ := evidence.ReadJSONL(filepath.Join(d, "events.jsonl"))
	repl := map[string]evidence.CheckRow{
		"AU01": itemAU01(d, item, ev),
		"AU02": itemAU02(item, ev),
		"AU03": itemAU03(d, st),
		"AU11": itemAU11(d, item),
	}
	sliceID, _ := st["slice"].(string)
	if sliceID != "" { // 026: a slice run's PR counts, and the slice order (AU13)
		repl["AU03"] = sliceAU03(d, st, item, sliceID)
	}
	if st["state"] == "stopped(back-to-planning)" { // 033 FR-011: no final pass or PR is expected
		why := "run ended by the owner for planning"
		repl["AU02"] = evidence.NewCheckRow("AU02", "audit", NA, repl["AU02"].Description, why)
		repl["AU03"] = evidence.NewCheckRow("AU03", "audit", NA, repl["AU03"].Description, why)
	}
	out := make([]evidence.CheckRow, 0, len(rows)+1)
	for _, r := range rows {
		if x, ok := repl[r.ID]; ok {
			r = x
		}
		if r.ID == "AU04" {
			r = itemAU04(d, r)
		}
		out = append(out, r)
	}
	if sliceID != "" {
		out = append(out, sliceAU13(d, st, item, sliceID))
	}
	return out
}

// itemAU04 adds P11 to AU04 for an item run: no hidden-test file reached any worker (014 FR-008).
func itemAU04(d string, base evidence.CheckRow) evidence.CheckRow {
	checks, _ := evidence.ReadJSONL(filepath.Join(d, "checks", "results.jsonl"))
	n, pass := 0, true
	for _, c := range checks {
		if str(c, "id") == "P11" {
			n++
			pass = pass && str(c, "result") == "pass"
		}
	}
	ok := base.Result == "pass" && n > 0 && pass
	// With a dependency cache, P12 must pass too: graders had no egress (014 R7).
	var st map[string]any
	readJSON(filepath.Join(d, "run.json"), &st)
	p12 := 0
	if _, cache := st["cache"]; cache {
		for _, c := range checks {
			if str(c, "id") == "P12" {
				p12++
				ok = ok && str(c, "result") == "pass"
			}
		}
		ok = ok && p12 > 0
	}
	// The security level (roadmap C-14, 018) is reported, never a failure.
	level, _ := st["security_level"].(string)
	if level == "" {
		level = "unrecorded"
	}
	return row("AU04", ok, base.Description+"; no hidden test reached a worker (P11, 014)",
		fmt.Sprintf("%s p11_rows=%d p12_rows=%d security_level=%s", base.Detail, n, p12, level))
}

// itemAU01: only the item was gated allow, its readiness passed, and no other task has events
// in the attempt stage.
func itemAU01(d, item string, ev []evidence.Row) evidence.CheckRow {
	allows, other := 0, 0
	for _, e := range ev {
		if str(e, "event") == "gate_allow" {
			if str(e, "task") != item {
				other++
			}
			allows++
		}
		if str(e, "stage") == "attempt" && str(e, "task") != item {
			other++
		}
	}
	pass, total := 0, 0
	recs, _ := filepath.Glob(filepath.Join(d, "gate", "*-"+item+".readiness.json"))
	sort.Strings(recs)
	allowed := false
	if len(recs) > 0 {
		var res struct {
			Decision string `json:"decision"`
			Rules    []struct {
				Result string `json:"result"`
			} `json:"rules"`
		}
		if readJSON(recs[len(recs)-1], &res) {
			allowed = res.Decision == "allow"
			for _, r := range res.Rules {
				total++
				if r.Result == "pass" {
					pass++
				}
			}
		}
	}
	ok := item != "" && allows > 0 && other == 0 && allowed && total > 0 && pass == total
	return row("AU01", ok, "only the prepared item was allowed and attempted (014 FR-002)",
		fmt.Sprintf("item=%s readiness=%d/%d other_attempts=%d", item, pass, total, other))
}

// itemAU02: the baseline passed before the first attempt; then zero or more failed grades and
// exactly one pass, which is the last.
func itemAU02(item string, ev []evidence.Row) evidence.CheckRow {
	baseline, firstAttempt := -1, -1
	var grades []string
	for i, e := range ev {
		if str(e, "event") == "baseline_pass" && baseline < 0 {
			baseline = i
		}
		if str(e, "stage") == "attempt" && firstAttempt < 0 {
			firstAttempt = i
		}
		evn := str(e, "event")
		if str(e, "stage") == "grade" && (evn == "grade_pass" || evn == "grade_fail") && str(e, "task") == item {
			grades = append(grades, strings.TrimPrefix(evn, "grade_"))
		}
	}
	passes := 0
	for _, g := range grades {
		if g == "pass" {
			passes++
		}
	}
	ok := baseline >= 0 && firstAttempt >= 0 && baseline < firstAttempt && passes == 1 && grades[len(grades)-1] == "pass"
	b := "missing"
	if baseline >= 0 {
		b = "pass"
	}
	return row("AU02", ok, "baseline passed, then failed attempts and exactly one final pass (014 FR-007)",
		fmt.Sprintf("baseline=%s order=%s", b, pyrepr.Strings(grades)))
}

// itemAU03: exactly 1 task PR and at most 1 status PR, all drafts by the publisher identity, on
// the target repository.
func itemAU03(d string, st map[string]any) evidence.CheckRow {
	var prs []map[string]any
	readJSON(filepath.Join(d, "prs.json"), &prs)
	target, _ := st["target"].(map[string]any)
	repo, _ := target["repo"].(string)
	nTask, nStatus, ok := 0, 0, repo != ""
	for _, p := range prs {
		switch p["kind"] {
		case "task", nil:
			nTask++
		case "status":
			nStatus++
		default:
			ok = false
		}
		if !prOnHost(p, st, repo) {
			ok = false
		}
	}
	ok = ok && nTask == 1 && nStatus <= 1
	return row("AU03", ok, "exactly 1 draft task PR (plus at most 1 status PR) on the target, by the publisher identity (014)",
		fmt.Sprintf("repo=%s task_prs=%d status_prs=%d", repo, nTask, nStatus))
}

// itemAU11: no attempt was induced, and every prompt carries the spec unaltered: the spec text
// in the prompt hashes to meta.spec_sha256, the same for every attempt.
func itemAU11(d, item string) evidence.CheckRow {
	var bad []string
	dirs, _ := filepath.Glob(filepath.Join(d, "attempts", item+"-*"))
	sort.Strings(dirs)
	want := ""
	for _, a := range dirs {
		name := filepath.Base(a)
		var meta map[string]any
		if !readJSON(filepath.Join(a, "meta.json"), &meta) {
			bad = append(bad, name+": no meta.json")
			continue
		}
		if meta["induction"] != nil {
			bad = append(bad, name+": induced")
		}
		sha, _ := meta["spec_sha256"].(string)
		if want == "" {
			want = sha
		}
		raw, err := os.ReadFile(filepath.Join(a, "prompt.md"))
		if sha == "" || sha != want || err != nil || prompt.SpecSHA256(specSection(string(raw))) != sha {
			bad = append(bad, name+": spec altered or unrecorded")
		}
	}
	if len(dirs) == 0 {
		bad = append(bad, "no attempts")
	}
	return row("AU11", len(bad) == 0, "no induction; every attempt got the item's spec unaltered (014)",
		fmt.Sprintf("attempts=%d %s", len(dirs), pyrepr.Strings(bad)))
}

// specSection is the "# Specification" part of an item prompt.
func specSection(p string) string {
	_, rest, ok := strings.Cut(p, "\n# Specification\n\n")
	if !ok {
		return ""
	}
	for _, next := range []string{"\n# Plan\n", "\n# Task list ("} {
		if i := strings.Index(rest, next); i >= 0 {
			rest = rest[:i]
		}
	}
	return rest
}

// runHost is the run's Git host from run.json "scm" (037); a run from before 037 is GitHub.
func runHost(st map[string]any) scm.Host {
	m, _ := st["scm"].(map[string]any)
	kind, _ := m["kind"].(string)
	name, _ := m["host"].(string)
	k, err := scm.ParseKind(kind)
	if err != nil {
		return scm.Host{Kind: scm.Kind(kind), Name: name}
	}
	if k != scm.GitLab {
		name = ""
	}
	return scm.NewHost(k, name, "")
}

// prOnHost is AU03's rule for one PR on an item or slice run: a draft by the publisher identity,
// its URL on the run's Git host and repository (037: the host's form), and, when the publisher
// read-back recorded ids, the author's id equal to the run's publisher id.
func prOnHost(p map[string]any, st map[string]any, repo string) bool {
	draft, _ := p["draft"].(bool)
	url, _ := p["url"].(string)
	if !draft || !truthy(p["author"]) || fmt.Sprint(p["author"]) != fmt.Sprint(p["publisher_identity"]) ||
		!strings.Contains(url, scm.ChangeRequestURLPrefix(runHost(st), repo)) {
		return false
	}
	if id, _ := p["author_id"].(string); id != "" {
		if pub, _ := st["publisher_id"].(string); pub != "" && scm.NormalID(id) != scm.NormalID(pub) {
			return false
		}
	}
	return true
}
