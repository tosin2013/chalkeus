package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// notImplemented is returned by commands not yet ported; the reference pilot still serves them
// through `PILOT_IMPL=python` (specs/016-go-pilot/research.md R14).
func notImplemented(name string) error {
	return Exit(ExitError, "%s: not implemented in the Go pilot yet (use PILOT_IMPL=python)", name)
}

// NewRoot builds the command tree with the reference pilot's commands, flags and arguments.
func NewRoot(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "pilot",
		Short:         "specfoundry M1 pilot",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(
		versionCmd(version),
		upCmd(), downCmd(), publisherTokenCmd(), importOpencodeCmd(), resetFixtureCmd(),
		runCmd(), resumeCmd(), approveCmd(), gateCmd(), auditCmd(), statusCmd(),
		githubCmd(), appKeyCmd(), appTokenCmd(),
	)
	return root
}

// Execute runs the command line and returns the process exit code.
func Execute(args []string, version string, stdout, stderr io.Writer) int {
	root := NewRoot(version)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	return exitCode(root.Execute(), stderr)
}

// Main is the binary entry point.
func Main(version string) {
	os.Exit(Execute(os.Args[1:], version, os.Stdout, os.Stderr))
}

func versionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the pilot version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "pilot %s (go)\n", version)
			return nil
		},
	}
}
