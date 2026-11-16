//go:build !windows

package watcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func runDir(t *testing.T) string {
	t.Helper()
	d := filepath.Join(t.TempDir(), "20261116T101500Z")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	return d
}

// fakeAlive treats the given pids as live watchers.
func fakeAlive(t *testing.T, live map[int]bool) {
	t.Helper()
	old := Alive
	Alive = func(pid int, _ string) bool { return live[pid] }
	t.Cleanup(func() { Alive = old })
}

func TestEnsureIsIdempotent(t *testing.T) {
	d := runDir(t)
	live := map[int]bool{}
	fakeAlive(t, live)
	spawns := 0
	spawn := func(cmd *exec.Cmd) (int, error) { spawns++; live[4242] = true; return 4242, nil }
	pid, started, err := Ensure("/repo", d, "R", spawn)
	if err != nil || pid != 4242 || !started {
		t.Fatal(pid, started, err)
	}
	pid, started, err = Ensure("/repo", d, "R", spawn)
	if err != nil || pid != 4242 || started || spawns != 1 {
		t.Fatal(pid, started, err, spawns)
	}
	if st := Status(d, "R"); !st.Running || st.Pid != 4242 || st.Log != LogFile(d) {
		t.Fatalf("%+v", st)
	}
}

func TestStalePidfileIsReplaced(t *testing.T) {
	d := runDir(t)
	_ = os.MkdirAll(filepath.Dir(PidFile(d)), 0o755)
	_ = os.WriteFile(PidFile(d), []byte("999999\n"), 0o644)
	live := map[int]bool{}
	fakeAlive(t, live)
	pid, started, err := Ensure("/repo", d, "R", func(*exec.Cmd) (int, error) { live[777] = true; return 777, nil })
	if err != nil || !started || pid != 777 {
		t.Fatal(pid, started, err)
	}
	raw, _ := os.ReadFile(PidFile(d))
	if strings.TrimSpace(string(raw)) != "777" {
		t.Fatal(string(raw))
	}
}

func TestSpawnArgumentsUseANewSession(t *testing.T) {
	d := runDir(t)
	fakeAlive(t, map[int]bool{})
	var got *exec.Cmd
	_, _, err := Ensure("/repo", d, "R1", func(cmd *exec.Cmd) (int, error) { got = cmd; return 1, nil })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.Args[1:], " ") != "github watch --run R1" || got.Dir != "/repo" {
		t.Fatalf("%v %s", got.Args, got.Dir)
	}
	exe, _ := os.Executable()
	if got.Path != exe {
		t.Fatalf("not the same binary: %s", got.Path)
	}
	if got.SysProcAttr == nil || !newSession(got.SysProcAttr) {
		t.Fatalf("not a new session: %+v", got.SysProcAttr)
	}
	if f, ok := got.Stdin.(*os.File); !ok || f.Name() != os.DevNull {
		t.Fatalf("stdin %v", got.Stdin)
	}
	if f, ok := got.Stdout.(*os.File); !ok || f.Name() != LogFile(d) || got.Stderr != got.Stdout {
		t.Fatalf("stdout %v", got.Stdout)
	}
}

func TestStopAndRelease(t *testing.T) {
	d := runDir(t)
	fakeAlive(t, map[int]bool{})
	if _, err := Stop(d, "R"); err != ErrNotRunning {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Dir(PidFile(d)), 0o755)
	_ = os.WriteFile(PidFile(d), []byte("55\n"), 0o644)
	Release(d, 56) // another pid: kept
	if _, err := os.Stat(PidFile(d)); err != nil {
		t.Fatal("pidfile of another process removed")
	}
	Release(d, 55)
	if _, err := os.Stat(PidFile(d)); err == nil {
		t.Fatal("pidfile kept")
	}
}

// A real detached process: it gets its own session, and a live process that is not this run's
// watcher (a reused pid) does not count as one.
func TestRealDetachedProcess(t *testing.T) {
	if _, err := os.Stat("/proc/self/cmdline"); err != nil {
		t.Skip("needs /proc")
	}
	cmd := exec.Command("sleep", "30")
	cmd.SysProcAttr = detached()
	pid, err := Spawn(cmd)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = terminate(pid) })
	if sid := sessionOf(t, pid); sid != pid {
		t.Fatalf("session %d, want %d", sid, pid)
	}
	if !processAlive(pid) {
		t.Fatal("not alive")
	}
	if Alive(pid, "R") {
		t.Fatal("an unrelated process counted as the watcher")
	}
	d := runDir(t)
	_ = os.MkdirAll(filepath.Dir(PidFile(d)), 0o755)
	_ = os.WriteFile(PidFile(d), []byte(strconv.Itoa(pid)), 0o644)
	if Status(d, "R").Running {
		t.Fatal("status trusts a reused pid")
	}
}
