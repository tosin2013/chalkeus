package worker

import (
	"errors"
	"testing"
	"time"

	"github.com/tosin2013/chalkeus/internal/evidence"
)

// 031 T034: the network check before an attempt's namespace is created.
func TestNetworkReady(t *testing.T) {
	events := func(run *evidence.Run) [][2]string {
		rows, _ := run.Events()
		var out [][2]string
		for _, r := range rows {
			out = append(out, [2]string{r["event"].(string), r["detail"].(string)})
		}
		return out
	}
	newRun := func() *evidence.Run {
		d := t.TempDir()
		return &evidence.Run{Dir: d, ID: "20990101T000000Z", State: map[string]any{}}
	}

	run := newRun()
	if !networkReady(Deps{}, run, "A", 1, Limits{DeadlineS: 1200}) || len(events(run)) != 0 {
		t.Fatal("nil Network: must pass without events")
	}

	var asked time.Duration
	run = newRun()
	ok := networkReady(Deps{Network: func(need time.Duration) (string, error) { asked = need; return "", nil }}, run, "A", 1, Limits{DeadlineS: 1200})
	if !ok || asked != 20*time.Minute+NetworkMargin || len(events(run)) != 0 {
		t.Fatalf("fresh token: ok=%v need=%v events=%v", ok, asked, events(run))
	}

	run = newRun()
	ok = networkReady(Deps{Network: func(time.Duration) (string, error) { return "calico-node restarted", nil }}, run, "B", 1, Limits{DeadlineS: 1200})
	if got := events(run); !ok || len(got) != 1 || got[0] != [2]string{"network_repaired", "calico-node restarted"} {
		t.Fatalf("repaired: ok=%v events=%v", ok, got)
	}

	run = newRun()
	ok = networkReady(Deps{Network: func(time.Duration) (string, error) { return "", errors.New("token expired") }}, run, "B", 1, Limits{DeadlineS: 1200})
	got := events(run)
	if ok || len(got) != 2 || got[0][0] != "attempt_error" || got[1] != [2]string{"attempt_failed", "harness_error"} {
		t.Fatalf("failed: ok=%v events=%v", ok, got)
	}
}
