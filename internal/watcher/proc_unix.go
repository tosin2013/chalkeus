//go:build !windows

package watcher

import (
	"errors"
	"syscall"
)

func detached() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func terminate(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }

// DetachedAttr starts a process in a new session, without the caller's controlling terminal.
func DetachedAttr() *syscall.SysProcAttr { return detached() }
