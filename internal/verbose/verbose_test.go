package verbose

import (
	"strings"
	"testing"
	"time"
)

func capture(t *testing.T, l int) *strings.Builder {
	t.Helper()
	var b strings.Builder
	Set(l, &b)
	nowFunc = func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { Set(0, nil); mu.Lock(); redact = nil; mu.Unlock(); nowFunc = time.Now })
	return &b
}

func TestLevels(t *testing.T) {
	b := capture(t, 2)
	V(1, "one")
	V(2, "two")
	V(3, "three")
	if got := b.String(); got != "[12:00:00] one\n[12:00:00] two\n" {
		t.Fatalf("%q", got)
	}
}

func TestQuietByDefault(t *testing.T) {
	b := capture(t, 0)
	V(1, "x")
	if b.Len() != 0 {
		t.Fatal("level 0 must print nothing")
	}
}

func TestRedaction(t *testing.T) {
	b := capture(t, 3)
	AddRedaction("llm.internal.example", "ghp_secretvalue", "ab") // "ab" is too short to register
	V(1, "calling https://llm.internal.example/v1 with ghp_secretvalue about ab")
	got := b.String()
	if strings.Contains(got, "llm.internal.example") || strings.Contains(got, "ghp_secretvalue") || !strings.Contains(got, "about ab") {
		t.Fatalf("%q", got)
	}
}

func TestFormatHarnessEvent(t *testing.T) {
	cases := map[string]string{
		`{"type":"text","part":{"type":"text","text":"All three\ntasks done"}}`:                                     "agent: All three tasks done",
		`{"type":"tool_use","part":{"tool":"bash","state":{"status":"completed","input":{"command":"pytest -q"}}}}`: `tool bash {"command":"pytest -q"}`,
		`{"type":"step_start","part":{}}`: "step started",
		`{"type":"step_finish","part":{"reason":"stop","cost":0.01,"tokens":{"input":12,"output":3}}}`: "step finished (stop): tokens in=12 out=3, cost=0.01",
		`not json`: "",
		`{"type":"something_new"}`: "something_new",
	}
	for in, want := range cases {
		if got := FormatHarnessEvent(in); got != want {
			t.Errorf("%s:\n got %q\nwant %q", in, got, want)
		}
	}
}
