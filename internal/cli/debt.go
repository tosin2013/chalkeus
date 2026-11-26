package cli

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/google/go-github/v85/github"
	"github.com/spf13/cobra"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/debt"
	"github.com/tosin2013/chalkeus/internal/ghapp"
	"github.com/tosin2013/chalkeus/internal/ghloop"
)

// debtCmd is `./chalkeus forge debt --target <name> [--json]` (029, contracts/cli.md): a
// read-only triage report of the target's open Dependabot PRs and its baseline problems.
func debtCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "debt --target <name> [--json]",
		Short: "read-only: group a target's open Dependabot PRs by root cause, classify them, and flag baseline problems",
		Args:  cobra.NoArgs,
		RunE:  runDebt,
	}
	addTargetFlag(c)
	requireTarget(c)
	c.Flags().Bool("json", false, "print the report as JSON")
	return c
}

func runDebt(cmd *cobra.Command, _ []string) error {
	asJSON, _ := cmd.Flags().GetBool("json")
	l, err := layout()
	if err != nil {
		return err
	}
	cfg, err := config.Load(l.Root, false, false)
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	t, err := loadTarget(cmd, l, cfg)
	if err != nil {
		return err
	}
	if t.Fixture {
		return Exit(ExitError, "debt reads a --target repository, not the fixture")
	}
	if err := gitHubOnly("forge debt", t.Host()); err != nil { // reads Dependabot (037)
		return err
	}
	gh, cred, perms, err := reportClient(cfg, t)
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	rep, err := debt.Build(ctx, debt.GitHub{C: gh}, debt.Options{Target: t.Name, Repo: t.Repo, Credential: cred, Permissions: perms, Now: time.Now()})
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	return writeReport(cmd.OutOrStdout(), rep, asJSON)
}

func writeReport(w io.Writer, rep *debt.Report, asJSON bool) error {
	if asJSON {
		return rep.WriteJSON(w)
	}
	rep.WriteText(w)
	return nil
}

// reportClient is the report's read-only GitHub client (029 FR-008, research R1): in App mode an
// installation token for the target repository restricted to read permissions, held in memory;
// otherwise forge's read client. The publisher's write credential is never loaded.
func reportClient(cfg *config.Config, t config.Target) (*github.Client, string, map[string]string, error) {
	cred := cfg.Get("PILOT_PUBLISHER_CREDENTIAL")
	if cred == "" {
		if _, err := os.Stat(cfg.AppKey()); err == nil {
			cred = "app"
		}
	}
	if cred == "app" {
		key := cfg.AppKey()
		if err := config.CheckMode(key); err != nil {
			return nil, "", nil, err
		}
		tok, perms, err := ghapp.ReadToken(key, cfg.Get("PILOT_APP_ID"), cfg.Get("PILOT_APP_INSTALLATION_ID"), t.Repo, time.Now())
		if err != nil {
			return nil, "", nil, err
		}
		return ghloop.NewClient(tok), "app-read", perms, nil
	}
	c, err := ghloop.OperatorClient()
	if err != nil {
		return nil, "", nil, err
	}
	return c, "operator-read", nil, nil
}
