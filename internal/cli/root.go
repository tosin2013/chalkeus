package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/verbose"
)

// NewRoot builds the command tree with the reference pilot's commands, flags and arguments.
func NewRoot(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "pilot",
		Short:         "specfoundry M1 pilot",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().CountP("verbose", "v",
		"diagnostics on stderr (Go pilot only): -v steps and timings, -vv pod phases and events, -vvv live agent activity; also PILOT_VERBOSE=N")
	root.AddCommand(
		versionCmd(version),
		upCmd(), downCmd(), publisherTokenCmd(), importOpencodeCmd(), resetFixtureCmd(),
		runCmd(), resumeCmd(), approveCmd(), gateCmd(), auditCmd(), statusCmd(),
		githubCmd(), appKeyCmd(), appTokenCmd(), targetsCmd(), authorCmd(), doctorCmd(),
	)
	return root
}

// Execute runs the command line and returns the process exit code.
func Execute(args []string, version string, stdout, stderr io.Writer) int {
	root := NewRoot(version)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		n, _ := cmd.Flags().GetCount("verbose")
		if n == 0 {
			n, _ = strconv.Atoi(os.Getenv("PILOT_VERBOSE"))
		}
		verbose.Set(n, stderr)
	}
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
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "pilot %s (go)\n", version)
			return nil
		},
	}
}
