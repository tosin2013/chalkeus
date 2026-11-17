package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/fixture"
	"github.com/tosin2013/specfoundry/internal/ghloop"
)

func runResetFixture(cmd *cobra.Command, _ []string) error {
	l, err := layout()
	if err != nil {
		return err
	}
	if name, _ := cmd.Flags().GetString("target"); name != "" && name != config.FixtureTarget {
		return Exit(ExitError, "NOT_A_FIXTURE: reset-fixture force-pushes and only ever acts on the fixture; target %q is a real repository", name)
	}
	cfg, err := config.Load(l.Root, false, true)
	if err != nil {
		return err
	}
	if err := config.RepoGuard(cfg.Repo(), config.OriginRepo(l.Root)); err != nil {
		return err
	}
	gh, err := ghloop.OperatorClient()
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	yes, _ := cmd.Flags().GetBool("yes")
	code, err := fixture.Reset(context.Background(), fixture.ResetOptions{
		Repo: cfg.Repo(), Branch: cfg.Branch(), Template: l.FixtureTemplate, BaseFile: l.FixtureBase,
		Yes: yes, In: cmd.InOrStdin(), Out: cmd.OutOrStdout(),
		ListPRs: func(ctx context.Context) ([]fixture.PRRef, error) {
			prs, err := ghloop.OpenPilotPRs(ctx, gh, cfg.Repo())
			out := make([]fixture.PRRef, 0, len(prs))
			for _, p := range prs {
				out = append(out, fixture.PRRef{Number: p.Number, HeadRef: p.HeadRef})
			}
			return out, err
		},
		ClosePR: func(ctx context.Context, p fixture.PRRef) error {
			return ghloop.ClosePR(ctx, gh, cfg.Repo(), ghloop.PR{Number: p.Number, HeadRef: p.HeadRef}, "Closed by `pilot reset-fixture`.")
		},
	})
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	return Code(code)
}
