package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tosin2013/chalkeus/internal/decisions"
	"github.com/tosin2013/chalkeus/internal/evidence"
)

func runApprove(cmd *cobra.Command, args []string) error {
	l, err := layout()
	if err != nil {
		return err
	}
	id, _ := cmd.Flags().GetString("run")
	run, err := evidence.LoadRun(l.EvidenceRoot(), id)
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	if run == nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "no run %s\n", id)
		return Code(ExitError)
	}
	code, msg := decisions.Record(run, args[0], "harness", "", nil)
	if code == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), msg)
	} else {
		fmt.Fprintln(cmd.ErrOrStderr(), msg)
	}
	return Code(code)
}
