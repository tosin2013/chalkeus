// Package evidence is the run evidence record and resumable run state
// (reference: the former Python pilot's evidence.py; specs/002-m1-gated-pilot/contracts/evidence-record.md).
package evidence

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TimeFormat is the evidence timestamp format (%Y-%m-%dT%H:%M:%SZ, UTC).
const TimeFormat = "2006-01-02T15:04:05Z"

// Now returns the current UTC time in the evidence format.
func Now() string { return time.Now().UTC().Format(TimeFormat) }

// SHA256File returns the hex SHA-256 of a file.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// marshal encodes like the reference's json.dumps(..., sort_keys=True): map keys sorted, no
// HTML escaping, non-ASCII escaped as \uXXXX (ensure_ascii), ", " and ": " separators on one
// line, or indent=2. The result ends with "\n".
func marshal(v any, indent bool) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	out := ensureASCII(b.String())
	if !indent {
		out = pyJSON(strings.TrimSuffix(out, "\n")) + "\n"
	}
	return []byte(out), nil
}

func ensureASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r < 0x80:
			b.WriteRune(r)
		case r < 0x10000:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			r -= 0x10000
			fmt.Fprintf(&b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
		}
	}
	return b.String()
}

func decode(raw []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber() // keep numbers as written (1.0 stays 1.0), as Python round-trips them
	return d.Decode(v)
}

// Decode parses JSON keeping numbers as written (json.Number).
func Decode(raw []byte, v any) error { return decode(raw, v) }

// WriteJSON writes v as indented JSON with sorted object keys (maps sort; structs keep field order).
func WriteJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := marshal(v, true)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// AppendJSONL appends one JSON line.
func AppendJSONL(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := marshal(v, false)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(raw)
	return err
}

// Row is one decoded JSON object from a .jsonl file.
type Row = map[string]any

// ReadJSONL reads a .jsonl file; a missing file is an empty list.
func ReadJSONL(path string) ([]Row, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []Row
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r Row
		if err := decode([]byte(line), &r); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, r)
	}
	return out, sc.Err()
}

// CheckRow is a check or audit result row.
type CheckRow struct {
	ID          string `json:"id"`
	Requirement string `json:"requirement"`
	Result      string `json:"result"`
	Description string `json:"description"`
	Detail      string `json:"detail"`
}

// NewCheckRow builds a row; detail is cut to 2000 characters, as in the reference.
func NewCheckRow(id, requirement, result, description, detail string) CheckRow {
	if r := []rune(detail); len(r) > 2000 {
		detail = string(r[:2000])
	}
	return CheckRow{ID: id, Requirement: requirement, Result: result, Description: description, Detail: detail}
}

// MarshalJSON writes the keys sorted, matching the reference's sort_keys output.
func (c CheckRow) MarshalJSON() ([]byte, error) {
	return marshal(map[string]string{"id": c.ID, "requirement": c.Requirement, "result": c.Result,
		"description": c.Description, "detail": c.Detail}, false)
}

// Run is one pilot run's directory and state.
type Run struct {
	Dir   string
	ID    string
	State map[string]any
	// LastApproval is the approval record used by the most recent decision (012/004 channels).
	LastApproval map[string]any
}

// OpenRun loads a run directory (state empty if run.json is missing).
func OpenRun(dir string) (*Run, error) {
	r := &Run{Dir: dir, ID: filepath.Base(dir), State: map[string]any{}}
	raw, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	if err := decode(raw, &r.State); err != nil {
		return nil, fmt.Errorf("%s/run.json: %w", dir, err)
	}
	return r, nil
}

// Save writes run.json.
func (r *Run) Save() error { return WriteJSON(filepath.Join(r.Dir, "run.json"), r.State) }

func (r *Run) next(key string) (int, error) {
	n := toInt(r.State[key]) + 1
	r.State[key] = n
	return n, r.Save()
}

// Event appends one event to events.jsonl. task and attempt may be nil.
func (r *Run) Event(stage string, task, attempt any, event, detail string) error {
	seq, err := r.next("event_seq")
	if err != nil {
		return err
	}
	return AppendJSONL(filepath.Join(r.Dir, "events.jsonl"), map[string]any{
		"seq": seq, "at": Now(), "stage": stage, "task": task, "attempt": attempt, "event": event, "detail": detail})
}

// Events reads events.jsonl.
func (r *Run) Events() ([]Row, error) { return ReadJSONL(filepath.Join(r.Dir, "events.jsonl")) }

// NextGateSeq returns the next gate sequence number.
func (r *Run) NextGateSeq() (int, error) { return r.next("gate_seq") }

// AttemptDir returns (and creates) attempts/<task>-<n>.
func (r *Run) AttemptDir(task string, n int) (string, error) {
	d := filepath.Join(r.Dir, "attempts", fmt.Sprintf("%s-%d", task, n))
	return d, os.MkdirAll(d, 0o755)
}

// SetRunState sets state and reason, saves, and records a run_<state> event.
func (r *Run) SetRunState(state string, reason any) error {
	r.State["state"] = state
	r.State["reason"] = reason
	if err := r.Save(); err != nil {
		return err
	}
	rs := ""
	if s, ok := reason.(string); ok {
		rs = s
	}
	return r.Event("run", nil, nil, "run_"+state, rs)
}

// AllRuns lists run directories (those with run.json) under root, sorted by name.
func AllRuns(root string) ([]string, error) {
	ents, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if _, err := os.Stat(filepath.Join(root, e.Name(), "run.json")); err == nil {
			out = append(out, filepath.Join(root, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

// LoadRun opens run id under root, or the latest run when id is empty. It returns nil, nil
// when there is no such run.
func LoadRun(root, id string) (*Run, error) {
	if id != "" {
		d := filepath.Join(root, id)
		if _, err := os.Stat(filepath.Join(d, "run.json")); err != nil {
			return nil, nil
		}
		return OpenRun(d)
	}
	runs, err := AllRuns(root)
	if err != nil || len(runs) == 0 {
		return nil, err
	}
	return OpenRun(runs[len(runs)-1])
}

func toInt(v any) int {
	switch x := v.(type) {
	case json.Number:
		n, _ := x.Int64()
		return int(n)
	case float64:
		return int(x)
	case int:
		return x
	}
	return 0
}

// KnownLimits are listed at the end of every run summary (verbatim from the reference).
var KnownLimits = []string{
	"Gate runs on the host (no published spec-status CLI image yet, PRD §8 ask 3).",
	"Grader uses `spec status --json`, not `verify-readiness` (report-only skill mode not shipped, PRD §8 ask 1).",
	"Reconcile is a deterministic pilot equivalent of the spec-status reconcile skill.",
	"Spend cap is enforced by the pilot watchdog and deadline unless a provider-side cap is declared (R2).",
	"Single-node kind only; multi-node policy enforcement not covered.",
}

func pyStr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case string:
		return x
	case bool:
		if x {
			return "True"
		}
		return "False"
	case json.Number:
		return x.String()
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprint(x)
	}
	return fmt.Sprint(v)
}

func orDash(m map[string]any, k string) string {
	v, ok := m[k]
	if !ok || v == nil {
		return "-"
	}
	if s, ok := v.(string); ok && s == "" {
		return "-"
	}
	return pyStr(v)
}

// WriteSummary writes summary.md (the reference write_summary).
func (r *Run) WriteSummary() (string, error) {
	ev, err := r.Events()
	if err != nil {
		return "", err
	}
	st := r.State
	state := pyStr(st["state"])
	head := "**State:** " + state
	if reason, ok := st["reason"].(string); ok && reason != "" {
		head += " (" + reason + ")"
	}
	title := "# M1 pilot run "
	if r.Kind() == KindItem {
		title = "# pilot item run "
	}
	out := []string{title + r.ID, "", head, ""}
	if len(ev) > 0 {
		t0, e0 := time.Parse(TimeFormat, pyStr(ev[0]["at"]))
		t1, e1 := time.Parse(TimeFormat, pyStr(ev[len(ev)-1]["at"]))
		if e0 == nil && e1 == nil {
			wait := 0
			for _, e := range ev {
				if pyStr(e["event"]) == "merge_wait_seconds" {
					n := 0
					_, _ = fmt.Sscanf(pyStr(e["detail"]), "%d", &n)
					wait += n
				}
			}
			total := int(t1.Sub(t0).Seconds())
			out = append(out, fmt.Sprintf("**Elapsed:** %d s total, %d s excluding the human merge wait", total, total-wait), "")
		}
	}
	if mode, ok := st["mode"]; ok && mode != nil {
		raw, _ := marshal(mode, false) // sorted keys, Python spacing
		out = append(out, "**Mode:** `"+strings.TrimSuffix(string(raw), "\n")+"`", "")
	}
	out = append(out, "## Attempts", "", "| Task | Attempt | Induction | Grade | Cost (USD) | Tokens |", "|---|---|---|---|---|---|")
	tasks, _ := st["tasks"].(map[string]any)
	for _, label := range r.TaskLabels() {
		t, _ := tasks[label].(map[string]any)
		atts, _ := t["attempts"].([]any)
		for _, x := range atts {
			a, _ := x.(map[string]any)
			sp, _ := a["spend"].(map[string]any)
			if sp == nil {
				sp = map[string]any{}
			}
			res := orDash(a, "result")
			if reason, ok := a["reason"].(string); ok && reason != "" {
				res += " (" + reason + ")"
			}
			out = append(out, fmt.Sprintf("| %s | %s | %s | %s | %s | %s |", label, pyStr(a["n"]), orDash(a, "induction"), res, orDash(sp, "cost_usd"), orDash(sp, "tokens")))
		}
	}
	var prs []map[string]any
	if raw, err := os.ReadFile(filepath.Join(r.Dir, "prs.json")); err == nil {
		_ = decode(raw, &prs)
	}
	out = append(out, "", "## Draft PRs", "")
	for _, p := range prs {
		out = append(out, fmt.Sprintf("- Task %s: [%s](%s) (draft=%s, author=%s)", pyStr(p["task"]), pyStr(p["url"]), pyStr(p["url"]), pyStr(p["draft"]), pyStr(p["author"])))
	}
	for _, name := range []string{"checks/results.jsonl", "audit.jsonl", "teardown.jsonl"} {
		rows, _ := ReadJSONL(filepath.Join(r.Dir, name))
		if len(rows) > 0 {
			pass := 0
			for _, x := range rows {
				if pyStr(x["result"]) == "pass" {
					pass++
				}
			}
			out = append(out, fmt.Sprintf("- `%s`: %d of %d pass", name, pass, len(rows)))
		}
	}
	out = append(out, "", "## Events", "", "| Seq | Stage | Task | Attempt | Event | Detail |", "|---|---|---|---|---|---|")
	for _, e := range ev {
		d := strings.ReplaceAll(strings.ReplaceAll(pyStr(e["detail"]), "|", "\\|"), "\n", " ")
		if rs := []rune(d); len(rs) > 120 {
			d = string(rs[:120])
		}
		task, att := "", ""
		if e["task"] != nil && pyStr(e["task"]) != "" {
			task = pyStr(e["task"])
		}
		if e["attempt"] != nil && pyStr(e["attempt"]) != "0" {
			att = pyStr(e["attempt"])
		}
		out = append(out, fmt.Sprintf("| %s | %s | %s | %s | %s | %s |", pyStr(e["seq"]), pyStr(e["stage"]), task, att, pyStr(e["event"]), d))
	}
	out = append(out, "", "## Known limits", "")
	for _, l := range KnownLimits {
		out = append(out, "- "+l)
	}
	p := filepath.Join(r.Dir, "summary.md")
	return p, os.WriteFile(p, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

// pyJSON converts Go's compact JSON to Python's json.dumps spacing (", " and ": ").
func pyJSON(raw string) string {
	var b strings.Builder
	inStr, esc := false, false
	for _, c := range raw {
		b.WriteRune(c)
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case ',', ':':
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// RecordCheck appends a check row to file (relative to the run dir) and prints it the way the
// reference does: "  PASS P01   FR-005   description".
func (r *Run) RecordCheck(out io.Writer, file, id, requirement, result, description, detail string) (CheckRow, error) {
	row := NewCheckRow(id, requirement, result, description, detail)
	if err := AppendJSONL(filepath.Join(r.Dir, file), row); err != nil {
		return row, err
	}
	mark := "FAIL"
	if result == "pass" {
		mark = "PASS"
	}
	fmt.Fprintf(out, "  %s %-5s %-8s %s\n", mark, id, requirement, description)
	return row, nil
}

// Task returns the run's state for a task label, creating it ({"state":"pending","attempts":[]}).
func (r *Run) Task(label string) map[string]any {
	tasks, _ := r.State["tasks"].(map[string]any)
	if tasks == nil {
		tasks = map[string]any{}
		r.State["tasks"] = tasks
	}
	t, _ := tasks[label].(map[string]any)
	if t == nil {
		t = map[string]any{"state": "pending", "attempts": []any{}}
		tasks[label] = t
	}
	return t
}

// PyFloat formats a float the way Python's json.dumps does (1.0 stays "1.0").
func PyFloat(f float64) json.Number {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return json.Number(s)
}

// Run kinds (014 R1): the fixture's A→B run, or one prepared item of a target repository.
const (
	KindFixture = "fixture"
	KindItem    = "item"
)

// Kind is the run's kind; records written before 014 have none and are fixture runs.
func (r *Run) Kind() string {
	if k, _ := r.State["kind"].(string); k == KindItem {
		return KindItem
	}
	return KindFixture
}

// TaskLabels are the run's task labels in order: A then B for the fixture (as the reference
// summary listed them), the item id for an item run.
func (r *Run) TaskLabels() []string {
	if r.Kind() == KindFixture {
		return []string{"A", "B"}
	}
	tasks, _ := r.State["tasks"].(map[string]any)
	out := make([]string, 0, len(tasks))
	for k := range tasks {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// SetItemRun marks the run as an item run of target (name, repo, branch, toolchain) and item.
func (r *Run) SetItemRun(target map[string]any, item string) {
	r.State["kind"] = KindItem
	r.State["target"] = target
	r.State["item"] = item
}

// NewRun creates a run directory under root with a fresh state (the reference new_run); impl
// "go" marks runs the Go pilot started (016 R16).
func NewRun(root string) (*Run, error) {
	id := time.Now().UTC().Format("20060102T150405Z")
	d := filepath.Join(root, id)
	for _, sub := range []string{"gate", "attempts", "reconcile", "checks"} {
		if err := os.MkdirAll(filepath.Join(d, sub), 0o755); err != nil {
			return nil, err
		}
	}
	r := &Run{Dir: d, ID: id, State: map[string]any{"run_id": id, "state": "in-progress", "reason": nil, "created": Now(),
		"tasks": map[string]any{}, "event_seq": 0, "gate_seq": 0, "impl": "go"}}
	return r, r.Save()
}

// WriteConfigJSON records the public config values and the secret files' hashes (never values).
func WriteConfigJSON(r *Run, public map[string]string, configDir string, secretFiles []string) error {
	secrets := map[string]any{}
	for _, p := range secretFiles {
		rel, err := filepath.Rel(configDir, p)
		if err != nil {
			rel = filepath.Base(p)
		}
		var sha any
		if s, err := SHA256File(p); err == nil {
			sha = s
		}
		secrets[rel] = map[string]any{"sha256": sha}
	}
	return WriteJSON(filepath.Join(r.Dir, "config.json"), map[string]any{"pilot_env": public, "secret_files": secrets})
}

func firstLine(args ...string) string {
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil && len(out) == 0 {
		return fmt.Sprintf("unavailable (%T)", err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return ""
	}
	return strings.Split(s, "\n")[0]
}

// WriteVersions records tool and image versions (versions.txt).
func WriteVersions(r *Run, pins map[string]string, build, kubeContext, cluster string, imgs map[string]string) error {
	server := ""
	if out, err := exec.Command("kubectl", "--context", kubeContext, "version", "-o", "json").Output(); err == nil {
		var v struct {
			ServerVersion struct {
				GitVersion string `json:"gitVersion"`
			} `json:"serverVersion"`
		}
		if json.Unmarshal(out, &v) == nil {
			server = v.ServerVersion.GitVersion
		}
	}
	calico := "unknown"
	if b, err := os.ReadFile(filepath.Join(build, "calico.sha256")); err == nil {
		calico = strings.TrimSpace(string(b))
	}
	src := "unknown"
	if b, err := os.ReadFile(filepath.Join(build, "image-source.txt")); err == nil {
		src = strings.TrimSpace(string(b))
	}
	lines := []string{
		"run_id: " + r.ID,
		"kind: " + firstLine("kind", "version"),
		"kubectl: " + firstLine("kubectl", "version", "--client"),
		"server: " + server,
		"node_image: " + firstLine("docker", "inspect", "--format", "{{.Config.Image}}", cluster+"-control-plane"),
		fmt.Sprintf("calico: %s sha256=%s", pins["CALICO_VERSION"], calico),
		fmt.Sprintf("spec: %s sha256=%s", firstLine("spec", "--version"), pins["SPEC_SHA256"]),
		fmt.Sprintf("opencode: %s sha256=%s", pins["OPENCODE_VERSION"], pins["OPENCODE_SHA256"]),
		"registry_image: " + pins["REGISTRY_IMAGE"],
		"image_source: " + src,
	}
	keys := make([]string, 0, len(imgs))
	for k := range imgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("image_%s: %s", k, imgs[k]))
	}
	lines = append(lines, "host: "+firstLine("uname", "-srm"))
	return os.WriteFile(filepath.Join(r.Dir, "versions.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
