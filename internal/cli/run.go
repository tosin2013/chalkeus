package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/pipeline"
)

func pipelineFor(cmd *cobra.Command) (*pipeline.P, error) {
	l, err := layout()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(l.UpOK); err != nil {
		return nil, &config.Error{Msg: "`pilot up` has not completed (missing pilot/.build/up.ok)", Code: ExitError}
	}
	cfg, err := config.Load(l.Root, true, true)
	if err != nil {
		return nil, err
	}
	credEnv, err := credentialEnv(cfg)
	if err != nil {
		return nil, err
	}
	p, err := pipeline.New(l, cfg, cmd.OutOrStdout(), credEnv)
	if err != nil {
		return nil, Exit(ExitError, "%v", err)
	}
	return p, nil
}

func runRun(cmd *cobra.Command, _ []string) error {
	p, err := pipelineFor(cmd)
	if err != nil {
		return err
	}
	return Code(p.Run())
}

func runResume(cmd *cobra.Command, _ []string) error {
	p, err := pipelineFor(cmd)
	if err != nil {
		return err
	}
	id, _ := cmd.Flags().GetString("run")
	return Code(p.Resume(id))
}
