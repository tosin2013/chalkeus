// Package verbose prints diagnostics to stderr at the operator's chosen level (016 FR-018):
//
//	-v    step and Job start/finish with durations
//	-vv   pod phases, container states, Kubernetes events, model-proxy refusals as they happen
//	-vvv  live agent activity from worker pods
//
// Diagnostics never touch stdout, exit codes or evidence, so they are outside parity. Every
// line is redacted: secret values (registered by package secret) and the model hosts become ***.
package verbose

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	mu      sync.Mutex
	level   int
	out     io.Writer = io.Discard
	redact  []string
	nowFunc = time.Now
)

// Set configures the level and the writer (stderr in the CLI).
func Set(l int, w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	if w == nil {
		w = io.Discard
	}
	level, out = l, w
}

// Level is the current level (0 = quiet).
func Level() int {
	mu.Lock()
	defer mu.Unlock()
	return level
}

// AddRedaction registers values that must never be printed (values shorter than 4 characters
// are ignored, so they cannot blank out ordinary text).
func AddRedaction(vals ...string) {
	mu.Lock()
	defer mu.Unlock()
	for _, v := range vals {
		if v = strings.TrimSpace(v); len(v) >= 4 {
			redact = append(redact, v)
		}
	}
	sort.Slice(redact, func(i, j int) bool { return len(redact[i]) > len(redact[j]) }) // longest first
}

// Redact replaces every registered value with ***.
func Redact(s string) string {
	mu.Lock()
	defer mu.Unlock()
	return redactLocked(s)
}

func redactLocked(s string) string {
	for _, v := range redact {
		s = strings.ReplaceAll(s, v, "***")
	}
	return s
}

// V prints one line at level n or above.
func V(n int, format string, a ...any) {
	mu.Lock()
	defer mu.Unlock()
	if level < n {
		return
	}
	msg := redactLocked(fmt.Sprintf(format, a...))
	for _, l := range strings.Split(strings.TrimRight(msg, "\n"), "\n") {
		fmt.Fprintf(out, "[%s] %s\n", nowFunc().Format("15:04:05"), l)
	}
}

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// FormatHarnessEvent turns one OpenCode `--format json` event line into a short line, or ""
// for lines that are not events.
func FormatHarnessEvent(line string) string {
	var e struct {
		Type string          `json:"type"`
		Part json.RawMessage `json:"part"`
		Err  json.RawMessage `json:"error"`
	}
	if json.Unmarshal([]byte(line), &e) != nil || e.Type == "" {
		return ""
	}
	var p struct {
		Text   string                     `json:"text"`
		Tool   string                     `json:"tool"`
		Reason string                     `json:"reason"`
		Cost   float64                    `json:"cost"`
		Tokens map[string]json.RawMessage `json:"tokens"`
		State  struct {
			Status string          `json:"status"`
			Input  json.RawMessage `json:"input"`
			Error  string          `json:"error"`
		} `json:"state"`
	}
	_ = json.Unmarshal(e.Part, &p)
	switch e.Type {
	case "text":
		return "agent: " + clip(p.Text, 160)
	case "tool_use":
		s := fmt.Sprintf("tool %s %s", p.Tool, clip(string(p.State.Input), 140))
		if p.State.Status != "" && p.State.Status != "completed" {
			s += " (" + p.State.Status + ")"
		}
		if p.State.Error != "" {
			s += " error: " + clip(p.State.Error, 80)
		}
		return s
	case "step_start":
		return "step started"
	case "step_finish":
		tok := func(k string) string {
			if v, ok := p.Tokens[k]; ok {
				return string(v)
			}
			return "0"
		}
		return fmt.Sprintf("step finished (%s): tokens in=%s out=%s, cost=%g", p.Reason, tok("input"), tok("output"), p.Cost)
	case "error":
		return "error: " + clip(string(e.Err), 160)
	}
	return e.Type
}
