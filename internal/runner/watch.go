package runner

import (
	"bufio"
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/verbose"
)

// Verbose diagnostics while a unit runs (016 FR-018). Nothing here feeds Collect, Status or
// evidence: the watcher only prints to stderr, and stops when the Job ends.

// cluster is what the watcher reads; *k8s.Client implements it (a fake in tests).
type cluster interface {
	JobPod(ns, job string) string
	PodState(ns, pod string) string
	PodEvents(ns, pod string) []k8s.PodEvent
	DeploymentLogs(ns, name string, since *time.Time) string
	ExecStream(ctx context.Context, ns, pod string, w io.Writer, cmd ...string) error
}

var pollEvery = 2 * time.Second

// HarnessLog is the worker's agent event log (pilot/images/worker/watchdog.py).
const HarnessLog = "/tmp/harness.jsonl"

// watch prints pod state changes, new events and (for workers) model-proxy refusals at -vv,
// and streams the agent's events at -vvv, until ctx is cancelled.
func watch(ctx context.Context, c cluster, h Handle) {
	var wg sync.WaitGroup
	defer wg.Wait()
	tag := h.Kind + " " + h.Namespace + "/" + h.Name
	start := time.Now()
	pod, state := "", ""
	seenEv, seenProxy := map[string]bool{}, map[string]bool{}
	streaming := false
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	for {
		if pod == "" {
			if pod = c.JobPod(h.Namespace, h.Name); pod != "" {
				verbose.V(2, "%s: pod %s", tag, pod)
			}
		}
		if pod != "" {
			if s := c.PodState(h.Namespace, pod); s != "" && s != state {
				state = s
				verbose.V(2, "%s: %s", tag, s)
			}
			for _, e := range c.PodEvents(h.Namespace, pod) {
				if !seenEv[e.Key] {
					seenEv[e.Key] = true
					verbose.V(2, "%s: event %s", tag, e.Text)
				}
			}
			if h.Kind == "worker" && verbose.Level() >= 3 && !streaming && strings.HasPrefix(state, "Running") {
				streaming = true
				wg.Add(1)
				go func(pod string) {
					defer wg.Done()
					streamHarness(ctx, c, h.Namespace, pod, tag)
				}(pod)
			}
		}
		if h.Kind == "worker" {
			for _, l := range strings.Split(c.DeploymentLogs("pilot-egress-model", "proxy", &start), "\n") {
				if strings.Contains(strings.ToLower(l), "refused") && !seenProxy[l] {
					seenProxy[l] = true
					verbose.V(2, "%s: model proxy: %s", tag, strings.TrimSpace(l))
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// streamHarness tails the agent's event log in the worker pod and prints one line per event.
func streamHarness(ctx context.Context, c cluster, ns, pod, tag string) {
	pr, pw := io.Pipe()
	go func() {
		err := c.ExecStream(ctx, ns, pod, pw, "tail", "-n", "+1", "-F", HarnessLog)
		_ = pw.CloseWithError(err)
	}()
	sc := bufio.NewScanner(pr)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		if s := verbose.FormatHarnessEvent(sc.Text()); s != "" {
			verbose.V(3, "%s: %s", tag, s)
		}
	}
	_ = pr.Close()
}
