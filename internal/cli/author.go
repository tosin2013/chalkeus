package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/author"
	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/fixture"
	"github.com/tosin2013/specfoundry/internal/gate"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/readiness"
)

// authorCmd is `pilot author` (018): the test author's helper commands. Only a separate
// test-author session runs these (the test-author skill); the session that drives runs never
// does (the pilot skill).
func authorCmd() *cobra.Command {
	g := &cobra.Command{
		Use:   "author",
		Short: "test author (018): draft an item's hidden tests and open a companion pull request; never run by the session that drives runs",
	}
	mk := func(use, short string, run func(cmd *cobra.Command, e author.Env, item string) error) *cobra.Command {
		c := &cobra.Command{Use: use, Short: short, Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				e, item, err := authorEnv(cmd)
				if err != nil {
					return err
				}
				return run(cmd, e, item)
			}}
		addTargetFlag(c)
		c.Flags().String("item", "", "roadmap item NNN of the target")
		_ = c.MarkFlagRequired("target")
		_ = c.MarkFlagRequired("item")
		return c
	}
	start := mk("start", "check the item and create the draft workspace", func(cmd *cobra.Command, e author.Env, item string) error {
		replace, _ := cmd.Flags().GetBool("replace")
		_, err := author.Start(context.Background(), e, item, replace)
		return err
	})
	start.Flags().Bool("replace", false, "propose replacing the item's existing hidden tests (still only a pull request)")
	check := mk("check", "prove on the cluster that the project's tests pass and the drafted tests fail on the unchanged base", func(cmd *cobra.Command, e author.Env, item string) error {
		p, err := pipelineFor(cmd)
		if err != nil {
			return err
		}
		_, err = author.Check(context.Background(), e, item, p.AuthorBaseline)
		return err
	})
	submit := mk("submit", "open the companion pull request for a checked draft", func(cmd *cobra.Command, e author.Env, item string) error {
		_, err := author.Submit(context.Background(), e, item)
		return err
	})
	status := mk("status", "print the latest draft's state and evidence folder", func(cmd *cobra.Command, e author.Env, item string) error {
		d, err := author.LoadDraft(author.Root(e.EvidenceRoot), "", e.Target.Name, item)
		if err != nil {
			return Exit(ExitError, "%v", err)
		}
		raw, _ := json.MarshalIndent(map[string]any{"draft_id": d.ID, "state": d.State, "level": d.Level, "identity": d.Identity,
			"files": len(d.Files), "checks": d.Checks, "mapping": d.Mapping, "pr": d.PR, "evidence": d.Dir(), "workspace": d.Workspace}, "", "  ")
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	})
	g.AddCommand(start, check, submit, status)
	return g
}

func authorEnv(cmd *cobra.Command) (author.Env, string, error) {
	item, _ := cmd.Flags().GetString("item")
	l, err := layout()
	if err != nil {
		return author.Env{}, "", err
	}
	cfg, err := config.Load(l.Root, false, false)
	if err != nil {
		return author.Env{}, "", err
	}
	t, err := loadTarget(cmd, l, cfg)
	if err != nil {
		return author.Env{}, "", err
	}
	if t.Fixture {
		return author.Env{}, "", Exit(ExitError, "the test author works on a --target repository's items, not the fixture")
	}
	gh, err := ghloop.OperatorClient()
	if err != nil {
		return author.Env{}, "", Exit(ExitError, "%v", err)
	}
	deps := readiness.Deps{Src: gate.ExecStatus{},
		Clone: func(repo, branch string) (string, string, error) { return fixture.FreshClone(repo, branch, 1) },
		OpenPRHeads: func(repo string) ([]string, error) {
			prs, err := ghloop.OpenPilotPRs(context.Background(), gh, repo)
			heads := make([]string, 0, len(prs))
			for _, p := range prs {
				heads = append(heads, p.HeadRef)
			}
			return heads, err
		}}
	deps.SkipHidden = true
	ready := func(t config.Target, item string) (readiness.Result, error) {
		return readiness.Check(deps, t, item, 0, "")
	}
	return author.Env{Target: t, EvidenceRoot: l.EvidenceRoot(), BuildDir: l.Build, Ready: ready,
		Identity: author.ResolveIdentity, OpenAuthorPRs: author.OpenAuthorPRs, Out: cmd.OutOrStdout()}, item, nil
}
