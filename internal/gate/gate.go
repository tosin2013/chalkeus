// Package gate is the fail-closed gate over spec-status (reference: pilot/specfoundry_pilot/gate.py;
// specs/002-m1-gated-pilot/contracts/gate-decision.schema.json).
//
// allow <=> every rule passes; anything else, including any unexpected input, refuses.
package gate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/tosin2013/specfoundry/internal/evidence"
)

var startable = map[string]bool{"planned": true, "specced": true, "in-progress": true}

// StatusSource runs spec-status for a directory. ExecStatus runs the `spec` binary; a library
// implementation can replace it once spec-status offers a public package (016 research R8).
type StatusSource interface {
	// Status returns the tool's exit code (nil if it could not run) and its stdout.
	Status(root string) (exit *int, stdout string)
	// Version returns the tool's version line.
	Version() string
}

// ExecStatus runs `<Bin> status --json --root <dir>` (Bin defaults to $SPEC_BIN or "spec").
type ExecStatus struct{ Bin string }

func (e ExecStatus) bin() string {
	if e.Bin != "" {
		return e.Bin
	}
	if b := os.Getenv("SPEC_BIN"); b != "" {
		return b
	}
	return "spec"
}

func (e ExecStatus) Status(root string) (*int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, e.bin(), "status", "--json", "--root", root)
	cmd.Stdout = &out
	err := cmd.Run()
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		return nil, "" // spawn failure or timeout: unevaluable
	}
	if ctx.Err() != nil {
		return nil, ""
	}
	code := cmd.ProcessState.ExitCode()
	return &code, out.String()
}

func (e ExecStatus) Version() string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, e.bin(), "--version").Output()
	if err != nil && len(out) == 0 {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return "unavailable"
		}
	}
	lines := strings.Split(string(out), "\n")
	if s := strings.TrimSpace(lines[0]); s != "" {
		return s
	}
	return "unknown"
}

// Decide is the pure decision from the tool's exit code and stdout. It never panics.
func Decide(itemID string, exit *int, stdout string) (decision string, reasons []string) {
	defer func() {
		if recover() != nil {
			decision, reasons = "refuse", []string{"UNEVALUABLE:internal"}
		}
	}()
	return decide(itemID, exit, stdout)
}

func refuse(r ...string) (string, []string) { return "refuse", r }

func decide(itemID string, exit *int, stdout string) (string, []string) {
	// Rule 1: exit code 0 or 1 only (2 = unevaluable).
	if exit == nil {
		return refuse("UNEVALUABLE:spawn_failed")
	}
	if *exit != 0 && *exit != 1 {
		return refuse(fmt.Sprintf("UNEVALUABLE:exit_%d", *exit))
	}
	if strings.TrimSpace(stdout) == "" {
		return refuse("UNEVALUABLE:empty")
	}
	// Rule 2: parseable, schema 1.0, complete, no error.
	var parsed any
	dec := json.NewDecoder(strings.NewReader(stdout))
	dec.UseNumber()
	if err := dec.Decode(&parsed); err != nil {
		return refuse("UNEVALUABLE:bad_json")
	}
	doc, ok := parsed.(map[string]any)
	if !ok || doc["schema_version"] != "1.0" {
		return refuse("SCHEMA_MISMATCH")
	}
	if v, present := doc["error"]; present && v != nil {
		return refuse("UNEVALUABLE:tool_error")
	}
	if doc["complete"] != true {
		return refuse("INCOMPLETE")
	}
	items, ok := doc["items"].([]any)
	if !ok {
		return refuse("SCHEMA_MISMATCH")
	}
	// Rule 3: exactly one item, status known, active.
	var matches []map[string]any
	for _, x := range items {
		if m, ok := x.(map[string]any); ok && m["id"] == itemID {
			matches = append(matches, m)
		}
	}
	if len(matches) == 0 {
		return refuse("ITEM_MISSING")
	}
	if len(matches) > 1 {
		return refuse("ITEM_DUPLICATE")
	}
	item := matches[0]
	var reasons []string
	if item["status_known"] != true {
		reasons = append(reasons, "STATUS_UNKNOWN")
	}
	if item["active"] != true {
		reasons = append(reasons, "INACTIVE")
	}
	// Rule 4: startable status.
	status := item["status"]
	switch {
	case status == "implemented":
		reasons = append(reasons, "ALREADY_IMPLEMENTED")
	case status == "verified":
		reasons = append(reasons, "ALREADY_VERIFIED")
	default:
		s, isStr := status.(string)
		if !isStr || !startable[s] {
			reasons = append(reasons, "STATUS_NOT_STARTABLE:"+slug(status))
		}
	}
	// Rule 5: every dependency satisfied (missing or non-true counts as unsatisfied).
	deps, ok := item["dependencies"].([]any)
	if !ok {
		return refuse("SCHEMA_MISMATCH")
	}
	unsatisfied := 0
	for _, x := range deps {
		m, isMap := x.(map[string]any)
		if isMap && m["satisfied"] == true {
			continue
		}
		unsatisfied++
		id := "unknown"
		if isMap {
			if v, present := m["id"]; present {
				id = pyStr(v)
			}
		}
		reasons = append(reasons, "DEP_UNSATISFIED:"+id)
	}
	// Rule 6: no error diagnostics for this item.
	if diags, ok := doc["diagnostics"].([]any); ok {
		for _, x := range diags {
			if m, ok := x.(map[string]any); ok && m["item_id"] == itemID && m["severity"] == "error" {
				code := "UNKNOWN"
				if v, present := m["code"]; present {
					code = pyStr(v)
				}
				reasons = append(reasons, "ITEM_ERROR_DIAGNOSTIC:"+code)
			}
		}
	}
	// Rule 7: the global blocked_count must agree with the per-item view.
	if unsatisfied > 0 {
		summary, _ := doc["summary"].(map[string]any)
		n, isInt := summary["blocked_count"].(json.Number)
		v, err := n.Int64()
		if !isInt || err != nil || v < 1 {
			reasons = append(reasons, "INCONSISTENT_BLOCKED_COUNT")
		}
	}
	if len(reasons) > 0 {
		return "refuse", reasons
	}
	return "allow", []string{"OK"}
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
	}
	return fmt.Sprint(v)
}

func slug(status any) string {
	var b strings.Builder
	for _, r := range strings.ToLower(pyStr(status)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "none"
	}
	return b.String()
}

// Input records what the decision was made from.
type Input struct {
	FixtureRepo          string  `json:"fixture_repo"`
	FixtureCommit        *string `json:"fixture_commit"`
	SpecVersion          string  `json:"spec_version"`
	SpecStatusExit       *int    `json:"spec_status_exit"`
	SpecStatusJSONSHA256 *string `json:"spec_status_json_sha256"`
}

// Decision is the gate decision record (gate-decision.schema.json).
type Decision struct {
	Version  int      `json:"version"`
	Seq      int      `json:"seq"`
	Task     string   `json:"task"`
	ItemID   string   `json:"item_id"`
	Decision string   `json:"decision"`
	Reasons  []string `json:"reasons"`
	Input    Input    `json:"input"`
	At       string   `json:"at"`
}

// MarshalJSON writes keys sorted, as the reference's write_json does.
func (d Decision) MarshalJSON() ([]byte, error) {
	in := map[string]any{"fixture_repo": d.Input.FixtureRepo, "fixture_commit": d.Input.FixtureCommit,
		"spec_version": d.Input.SpecVersion, "spec_status_exit": d.Input.SpecStatusExit,
		"spec_status_json_sha256": d.Input.SpecStatusJSONSHA256}
	return json.Marshal(map[string]any{"version": d.Version, "seq": d.Seq, "task": d.Task, "item_id": d.ItemID,
		"decision": d.Decision, "reasons": d.Reasons, "input": in, "at": d.At})
}

// Evaluate runs the status source on root and decides for item. When outDir is set it writes
// <seq>-<task>.status.json (the raw tool output) and <seq>-<task>.json (the decision).
func Evaluate(src StatusSource, task, itemID, root string, seq int, outDir, repo string, commit *string) (Decision, error) {
	exit, stdout := src.Status(root)
	decision, reasons := Decide(itemID, exit, stdout)
	var sha *string
	if stdout != "" {
		sum := sha256.Sum256([]byte(stdout))
		s := hex.EncodeToString(sum[:])
		sha = &s
	}
	if seq < 1 {
		seq = 1
	}
	d := Decision{Version: 1, Seq: seq, Task: task, ItemID: itemID, Decision: decision, Reasons: reasons,
		Input: Input{FixtureRepo: repo, FixtureCommit: commit, SpecVersion: src.Version(), SpecStatusExit: exit,
			SpecStatusJSONSHA256: sha}, At: evidence.Now()}
	if outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return d, err
		}
		if err := os.WriteFile(filepath.Join(outDir, fmt.Sprintf("%d-%s.status.json", d.Seq, task)), []byte(stdout), 0o644); err != nil {
			return d, err
		}
		if err := evidence.WriteJSON(filepath.Join(outDir, fmt.Sprintf("%d-%s.json", d.Seq, task)), d); err != nil {
			return d, err
		}
	}
	return d, nil
}

// Refused builds the decision recorded when the fixture could not even be cloned.
func Refused(src StatusSource, task, itemID string, seq int, repo, reason string) Decision {
	if seq < 1 {
		seq = 1
	}
	return Decision{Version: 1, Seq: seq, Task: task, ItemID: itemID, Decision: "refuse", Reasons: []string{reason},
		Input: Input{FixtureRepo: repo, SpecVersion: src.Version()}, At: evidence.Now()}
}

// EvaluateFresh gates against a fresh clone of the fixture's default branch (FR-019).
// clone is injected so tests need no network; nil uses a real git clone.
func EvaluateFresh(src StatusSource, repo, branch, task, itemID string, seq int, outDir string,
	clone func(repo, branch string) (dir, commit string, err error)) (Decision, error) {
	dir, commit, err := clone(repo, branch)
	if err != nil {
		d := Refused(src, task, itemID, seq, repo, "UNEVALUABLE:clone_failed")
		if outDir != "" {
			return d, evidence.WriteJSON(filepath.Join(outDir, fmt.Sprintf("%d-%s.json", d.Seq, task)), d)
		}
		return d, nil
	}
	defer os.RemoveAll(dir)
	return Evaluate(src, task, itemID, dir, seq, outDir, repo, &commit)
}
