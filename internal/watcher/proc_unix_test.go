//go:build !windows

package watcher

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func newSession(a *syscall.SysProcAttr) bool { return a.Setsid }

// sessionOf reads field 6 of /proc/<pid>/stat (the session id).
func sessionOf(t *testing.T, pid int) int {
	t.Helper()
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	f := strings.Fields(s[strings.LastIndex(s, ")")+2:])
	n, _ := strconv.Atoi(f[3])
	return n
}
