// Package cli is the `pilot` command line (specs/016-go-pilot/contracts/cli.md).
package cli

import (
	"errors"
	"fmt"
	"io"
)

// Exit codes, identical to the reference pilot (specs/002-m1-gated-pilot/contracts/cli.md).
const (
	ExitOK          = 0
	ExitFail        = 1
	ExitError       = 2
	ExitGuard       = 3
	ExitAwaitMerge  = 4
	ExitDeclined    = 5
	ExitAwaitRetry  = 6
	ExitGateRefusal = 10
)

// ExitErr carries a command's exit code. Msg, when set, is printed as `pilot: <Msg>` on stderr.
type ExitErr struct {
	Code int
	Msg  string
}

func (e *ExitErr) Error() string { return e.Msg }

// Exit returns an ExitErr with the given code and message; a zero code means success.
func Exit(code int, format string, a ...any) error {
	if code == ExitOK {
		return nil
	}
	return &ExitErr{Code: code, Msg: fmt.Sprintf(format, a...)}
}

// Code is a bare exit code with nothing to print (the command already printed its output).
func Code(code int) error {
	if code == ExitOK {
		return nil
	}
	return &ExitErr{Code: code}
}

// exitCode maps an error from command execution to the process exit code, printing the
// message the way the reference does (`pilot: <msg>` on stderr). Unknown errors, including
// usage errors (unknown command or flag), exit 2.
func exitCode(err error, stderr io.Writer) int {
	if err == nil {
		return ExitOK
	}
	var e *ExitErr
	if errors.As(err, &e) {
		if e.Msg != "" {
			fmt.Fprintf(stderr, "pilot: %s\n", e.Msg)
		}
		return e.Code
	}
	fmt.Fprintf(stderr, "pilot: %s\n", err)
	return ExitError
}
