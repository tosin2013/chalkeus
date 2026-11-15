// Package evidence is the run evidence record and resumable run state
// (reference: pilot/specfoundry_pilot/evidence.py; specs/002-m1-gated-pilot/contracts/evidence-record.md).
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
	"path/filepath"
	"sort"
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
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func marshal(v any, indent bool) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false) // Python's json.dumps does not escape <, > or &
	if indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil // Encode ends with "\n", like the reference's write_json
}

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
	defer f.Close()
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
	defer f.Close()
	var out []Row
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r Row
		if err := json.Unmarshal([]byte(line), &r); err != nil {
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
	if err := json.Unmarshal(raw, &r.State); err != nil {
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
	case float64:
		return int(x)
	case int:
		return x
	case json.Number:
		n, _ := x.Int64()
		return int(n)
	}
	return 0
}
