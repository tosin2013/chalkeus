//go:build windows

package watcher

import (
	"os"
	"syscall"
)

// Compiled but not supported end to end (016 R12): the pilot runs on Linux.

const (
	detachedProcess       = 0x00000008
	createNewProcessGroup = 0x00000200
)

func detached() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: detachedProcess | createNewProcessGroup}
}

func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	_ = p.Release()
	return true
}

func terminate(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

// DetachedAttr starts a process detached from the caller's console.
func DetachedAttr() *syscall.SysProcAttr { return detached() }
