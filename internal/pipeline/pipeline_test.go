package pipeline

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/decisions"
	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/paths"
)

func newP(t *testing.T) (*P, *evidence.Run, *bytes.Buffer) {
	t.Helper()
	d := filepath.Join(t.TempDir(), "20990101T000000Z")
	_ = os.MkdirAll(d, 0o755)
	r := &evidence.Run{Dir: d, ID: filepath.Base(d), State: map[string]any{"run_id": filepath.Base(d), "state": "in-progress", "impl": "go"}}
	_ = r.Save()
	v, _ := config.ValidateValues(map[string]string{"PILOT_FIXTURE_REPO": "o/x-pilot-fixture", "PILOT_MODEL": "p/m", "PILOT_MODEL_EGRESS_HOSTS": "h"})
	var out bytes.Buffer
	decisions.HasTTY = func() bool { return false }
	return &P{L: paths.At(t.TempDir()), Cfg: &config.Config{Values: v}, Out: &out}, r, &out
}

func TestNoTerminalPausesInsteadOfFailing(t *testing.T) {
	p, r, out := newP(t)
	if code := p.modeNotConfirmed(r, "none"); code != ExitDeclined || r.State["state"] != "paused(awaiting-mode)" {
		t.Fatalf("%d %v", code, r.State["state"])
	}
	if !strings.Contains(out.String(), "paused: mode not confirmed. no terminal: approve with `pilot/pilot approve --run") {
		t.Fatalf("%q", out.String())
	}
}

func TestExplicitNoOnTerminalFails(t *testing.T) {
	p, r, _ := newP(t)
	p.modeNotConfirmed(r, "tty")
	if r.State["state"] != "failed" || r.State["reason"] != "MODE_NOT_CONFIRMED" {
		t.Fatalf("%v", r.State)
	}
}

func TestModeJSONLikePython(t *testing.T) {
	p, _, _ := newP(t)
	if got := modeJSON(p.ModeOf()); got != `{"name": "step-by-step", "retry_limit": 2, "spend_cap_usd": 1.0, "token_cap": 2000000, "worker_deadline_s": 1200}` {
		t.Fatalf("%s", got)
	}
}

func TestConfirmModeHeadlessRecordsNone(t *testing.T) {
	p, r, out := newP(t)
	ok, ch := p.confirmMode(r)
	if ok || ch != "none" || !strings.Contains(out.String(), "Run mode for this pilot: {\"name\": \"step-by-step\"") {
		t.Fatalf("%v %s %q", ok, ch, out.String())
	}
	ev, _ := r.Events()
	last := ev[len(ev)-1]
	if last["event"] != "mode_declined" {
		t.Fatalf("%v", last)
	}
}
