package cli

import (
	"context"
	"github.com/tosin2013/chalkeus/internal/scm/hosts"

	"github.com/spf13/cobra"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/fixture"
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
	h := cfg.FixtureHost()
	if err := config.RepoGuardFor(h.Kind, cfg.Repo(), config.OriginRepo(l.Root)); err != nil {
		return err
	}
	sc, err := hosts.Operator(h)
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	yes, _ := cmd.Flags().GetBool("yes")
	code, err := fixture.Reset(context.Background(), fixture.ResetOptions{
		Repo: cfg.Repo(), Branch: cfg.Branch(), Host: h, Template: l.FixtureTemplate, BaseFile: l.FixtureBase,
		Yes: yes, In: cmd.InOrStdin(), Out: cmd.OutOrStdout(),
		ListPRs: func(ctx context.Context) ([]fixture.PRRef, error) {
			prs, err := sc.OpenChangeRequests(ctx, cfg.Repo(), "pilot/")
			out := make([]fixture.PRRef, 0, len(prs))
			for _, p := range prs {
				out = append(out, fixture.PRRef{Number: p.ID, HeadRef: p.HeadRef})
			}
			return out, err
		},
		ClosePR: func(ctx context.Context, p fixture.PRRef) error {
			return sc.CloseChangeRequest(ctx, cfg.Repo(), p.Number, "Closed by `./chalkeus forge reset-fixture`.")
		},
	})
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	return Code(code)
}
