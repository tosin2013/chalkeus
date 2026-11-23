package runner

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tosin2013/chalkeus/internal/k8s"
	"github.com/tosin2013/chalkeus/internal/verbose"
)

type fakeCluster struct {
	mu     sync.Mutex
	states []string
	i      int
}

func (f *fakeCluster) JobPod(string, string) string { return "pod-1" }
func (f *fakeCluster) PodState(string, string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.states[min(f.i, len(f.states)-1)]
	f.i++
	return s
}
func (f *fakeCluster) PodEvents(string, string) []k8s.PodEvent {
	return []k8s.PodEvent{{Key: "u1/1", Text: "Normal Pulled: image pulled"}}
}
func (f *fakeCluster) DeploymentLogs(string, string, *time.Time) string {
	return "CONNECT llm.internal.example:443\nRefused: evil.example not allowed\n"
}
func (f *fakeCluster) ExecStream(_ context.Context, _, _ string, w io.Writer, _ ...string) error {
	_, _ = io.WriteString(w, `{"type":"tool_use","part":{"tool":"bash","state":{"status":"completed","input":{"command":"ls"}}}}`+"\n")
	return nil
}

type syncBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func TestWatchPrintsChangesOnce(t *testing.T) {
	var out syncBuf
	verbose.Set(3, &out)
	verbose.AddRedaction("llm.internal.example")
	defer verbose.Set(0, io.Discard)
	pollEvery = 5 * time.Millisecond
	f := &fakeCluster{states: []string{"Pending, worker waiting (ContainerCreating)", "Running, worker running", "Running, worker running"}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { watch(ctx, f, Handle{Kind: "worker", Namespace: "w-a-1", Name: "worker-a-1"}); close(done) }()
	time.Sleep(80 * time.Millisecond)
	cancel()
	<-done
	got := out.String()
	for _, want := range []string{"pod pod-1", "Pending, worker waiting (ContainerCreating)", "Running, worker running",
		"event Normal Pulled: image pulled", "model proxy: Refused: evil.example not allowed", `tool bash {"command":"ls"}`} {
		if n := strings.Count(got, want); n != 1 {
			t.Errorf("%q printed %d times in:\n%s", want, n, got)
		}
	}
	if strings.Contains(got, "llm.internal.example") {
		t.Error("model host leaked")
	}
}

func TestWatchQuietBelowLevel3ForAgent(t *testing.T) {
	var out syncBuf
	verbose.Set(2, &out)
	defer verbose.Set(0, io.Discard)
	pollEvery = 5 * time.Millisecond
	f := &fakeCluster{states: []string{"Running, worker running"}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { watch(ctx, f, Handle{Kind: "worker", Namespace: "n", Name: "j"}); close(done) }()
	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done
	if strings.Contains(out.String(), "tool bash") {
		t.Error("agent activity must need -vvv")
	}
}
