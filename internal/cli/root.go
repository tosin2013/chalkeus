package cli

import (
	"fmt"
	"github.com/tosin2013/chalkeus/internal/scm"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tosin2013/chalkeus/internal/author"
	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/names"
	"github.com/tosin2013/chalkeus/internal/verbose"
)

// NewRoot builds the command tree: the root `chalkeus` (027 R1), with `version` and `forge`, which
// holds every pilot command (the reference pilot's commands, flags and arguments).
func NewRoot(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "chalkeus",
		Short:         "Chalkeus: humans plan, agents build",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().CountP("verbose", "v",
		"diagnostics on stderr: -v steps and timings, -vv pod phases and events, -vvv live agent activity; also FORGE_VERBOSE=N")
	root.AddCommand(versionCmd(version), NewForge(version))
	return root
}

// NewForge is `./chalkeus forge`: every pilot command.
func NewForge(version string) *cobra.Command {
	author.ForgeVersion = version // 041: recorded in proofs
	forge := &cobra.Command{
		Use:   "forge",
		Short: "run roadmap items with agents, under your approval",
		Args:  cobra.NoArgs, // an unknown command is a usage error, not help
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	forge.AddCommand(
		versionCmd(version),
		upCmd(), downCmd(), publisherTokenCmd(), importOpencodeCmd(), resetFixtureCmd(),
		runCmd(), resumeCmd(), approveCmd(), gateCmd(), auditCmd(), statusCmd(),
		githubCmd(), appKeyCmd(), appTokenCmd(), targetsCmd(), authorCmd(), doctorCmd(), stepCmd(), slicesCmd(), debtCmd(),
	)
	return forge
}

// Forge returns the `forge` command of a tree built by NewRoot.
func Forge(root *cobra.Command) *cobra.Command {
	for _, c := range root.Commands() {
		if c.Name() == "forge" {
			return c
		}
	}
	return nil
}

// Execute runs the command line and returns the process exit code.
func Execute(args []string, version string, stdout, stderr io.Writer) int {
	root := NewRoot(version)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		n, _ := cmd.Flags().GetCount("verbose")
		if n == 0 {
			n, _ = strconv.Atoi(names.Env("VERBOSE"))
		}
		verbose.Set(n, stderr)
		return applyTestMode()
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
		Short: "print the Chalkeus version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "chalkeus %s\n", version)
			return nil
		},
	}
}

// applyTestMode points the pilot at the integration tests' fake GitHub (025), only with
// PILOT_TEST_MODE=1, and marks every run as a test run.
func applyTestMode() error {
	tm, err := config.TestMode()
	if err != nil {
		return err
	}
	if !tm.On {
		return nil
	}
	if tm.API != "" {
		ghloop.API = strings.TrimSuffix(tm.API, "/")
		scm.GitHubAPI = ghloop.API // 037: scm/github builds its clients from scm.NewHost
	}
	if tm.GitBase != "" {
		fixture.BaseURL = tm.GitBase
		scm.GitHubGitBase = tm.GitBase
	}
	evidence.TestMode = tm.Record()
	return nil
}
