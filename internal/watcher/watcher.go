// Package watcher runs a run's `github watch` loop as a detached process, so the operator can
// close the terminal during a run (issue #25; specs/016-go-pilot/contracts/watcher.md).
//
// One watcher per run: github/watch.pid names it, with a liveness check. It is the same binary
// running `./chalkeus forge github watch --run <id>` in a new session (no controlling terminal, so it can
// never answer a prompt), stdin from /dev/null, output appended to github/watch.log. It exits,
// and removes the pidfile, when the run is no longer paused.
package watcher

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// PidFile is the run's watcher pidfile.
func PidFile(runDir string) string { return filepath.Join(runDir, "github", "watch.pid") }

// LogFile is the run's watcher log.
func LogFile(runDir string) string { return filepath.Join(runDir, "github", "watch.log") }

// readPid returns the pid in the pidfile, or 0.
func readPid(runDir string) int {
	raw, err := os.ReadFile(PidFile(runDir))
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// Alive reports whether pid is a live watcher for runID (replaced in tests). Where /proc is
// readable, a reused pid that is not this run's watcher counts as dead.
var Alive = func(pid int, runID string) bool {
	if !processAlive(pid) {
		return false
	}
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return true // no /proc (macOS): trust the signal check
	}
	cmd := strings.ReplaceAll(string(raw), "\x00", " ")
	return strings.Contains(cmd, "github watch") && strings.Contains(cmd, runID)
}

// State is a watcher's status.
type State struct {
	Pid     int
	Running bool
	Log     string
}

// Status reads the run's watcher state.
func Status(runDir, runID string) State {
	pid := readPid(runDir)
	return State{Pid: pid, Running: pid > 0 && Alive(pid, runID), Log: LogFile(runDir)}
}

// Spawner starts the detached process (replaced in tests).
type Spawner func(cmd *exec.Cmd) (pid int, err error)

// Spawn starts cmd and releases it.
func Spawn(cmd *exec.Cmd) (int, error) {
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}

// Command builds the watcher command: exe forge github watch --run <id>, in a new session, stdin
// from /dev/null, stdout and stderr to log, working directory root.
func Command(exe, root, runID string, log *os.File, devnull *os.File) *exec.Cmd {
	cmd := exec.Command(exe, "forge", "github", "watch", "--run", runID)
	cmd.Dir = root
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, log, log
	cmd.SysProcAttr = detached()
	return cmd
}

// Ensure starts the run's watcher unless a live one exists. It returns the watcher's pid and
// whether it was started now.
func Ensure(root, runDir, runID string, spawn Spawner) (int, bool, error) {
	if st := Status(runDir, runID); st.Running {
		return st.Pid, false, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return 0, false, err
	}
	if err := os.MkdirAll(filepath.Dir(PidFile(runDir)), 0o755); err != nil {
		return 0, false, err
	}
	log, err := os.OpenFile(LogFile(runDir), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = log.Close() }()
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = devnull.Close() }()
	if spawn == nil {
		spawn = Spawn
	}
	pid, err := spawn(Command(exe, root, runID, log, devnull))
	if err != nil {
		return 0, false, err
	}
	if err := os.WriteFile(PidFile(runDir), []byte(strconv.Itoa(pid)+"\n"), 0o644); err != nil {
		return pid, true, err
	}
	return pid, true, nil
}

// ErrNotRunning means no live watcher exists for the run.
var ErrNotRunning = errors.New("no watcher running")

// Stop sends SIGTERM to the run's watcher and removes the pidfile.
func Stop(runDir, runID string) (int, error) {
	st := Status(runDir, runID)
	_ = os.Remove(PidFile(runDir))
	if !st.Running {
		return st.Pid, ErrNotRunning
	}
	return st.Pid, terminate(st.Pid)
}

// Release removes the pidfile if it names pid (the watcher itself, on exit).
func Release(runDir string, pid int) {
	if readPid(runDir) == pid {
		_ = os.Remove(PidFile(runDir))
	}
}
