package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/ghapp"
	"github.com/tosin2013/chalkeus/internal/paths"
	"github.com/tosin2013/chalkeus/internal/pipeline"
)

// pipelineFor builds the pipeline for the command's target: --target on `run`, or the target an
// existing run (--run, else the latest) recorded in run.json (014 FR-004a: a paused run always
// continues against its own target). Without either it is the fixture, exactly as before.
func pipelineFor(cmd *cobra.Command) (*pipeline.P, error) {
	l, err := layout()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(l.UpOK); err != nil {
		return nil, &config.Error{Msg: "`./chalkeus forge up` has not completed (missing pilot/.build/up.ok)", Code: ExitError}
	}
	cfg, err := config.Load(l.Root, true, true)
	if err != nil {
		return nil, err
	}
	t, err := targetFor(cmd, l, cfg)
	if err != nil {
		return nil, err
	}
	creds, err := credsForTarget(cfg, t)
	if err != nil {
		return nil, err
	}
	p, err := pipeline.New(l, cfg, cmd.OutOrStdout(), creds)
	if err != nil {
		return nil, Exit(ExitError, "%v", err)
	}
	p.Target = t
	return p, nil
}

// targetFor is the --target profile, or the target recorded by the run the command acts on.
func targetFor(cmd *cobra.Command, l paths.Layout, cfg *config.Config) (config.Target, error) {
	if f := cmd.Flags().Lookup("target"); f != nil && f.Value.String() != "" {
		return loadTarget(cmd, l, cfg)
	}
	if f := cmd.Flags().Lookup("run"); f != nil {
		run, _ := evidence.LoadRun(l.EvidenceRoot(), f.Value.String())
		if run != nil && run.Kind() == evidence.KindItem {
			tg, _ := run.State["target"].(map[string]any)
			name, _ := tg["name"].(string)
			t, err := cfg.LoadTarget(name, config.OriginRepo(l.Root))
			if err != nil {
				return t, err
			}
			if repo, _ := tg["repo"].(string); repo != t.Repo {
				return t, Exit(ExitError, "run %s was started against %s, but target %q is now %s; restore the profile to continue it",
					run.ID, repo, name, t.Repo)
			}
			return t, nil
		}
	}
	return cfg.Fixture(), nil
}

// credsForTarget is the publisher credential for a target: the fixture's as before; for a real
// target, the GitHub App minting tokens for that repository only, or the target's own token in
// targets/<name>.publisher.env (014).
func credsForTarget(cfg *config.Config, t config.Target) (pipeline.Creds, error) {
	if t.Fixture {
		return credsFor(cfg)
	}
	cred := cfg.Get("PILOT_PUBLISHER_CREDENTIAL")
	if !t.Host().IsGitHub() { // 037: the GitHub App is GitHub-only; other hosts use the target's own token file
		cred = "token"
	}
	if cred == "" {
		if _, err := os.Stat(cfg.AppKey()); err == nil {
			cred = "app"
		} else {
			cred = "token"
		}
	}
	if cred == "app" {
		a := ghapp.App{Cfg: cfg, Repo: t.Repo}
		return pipeline.Creds{Env: a.CurrentEnv, App: true, BotLogin: a.BotLogin}, nil
	}
	if err := config.CheckMode(t.PublisherEnv); err != nil {
		return pipeline.Creds{}, Exit(ExitError, "target %q needs its own publisher token in %s (mode 0600), "+
			"a fine-grained token for %s only (contents and pull requests: write)", t.Name, t.PublisherEnv, t.Repo)
	}
	return pipeline.Creds{Env: func() (string, error) { return t.PublisherEnv, nil }}, nil
}

func runRun(cmd *cobra.Command, _ []string) error {
	item, _ := cmd.Flags().GetString("item")
	name, _ := cmd.Flags().GetString("target")
	sliceFlag, _ := cmd.Flags().GetString("slice")
	isTarget := name != "" && name != config.FixtureTarget
	switch {
	case sliceFlag != "" && item == "":
		return Exit(ExitError, "--slice needs --item NNN: a slice is part of one item")
	case item != "" && !isTarget:
		return Exit(ExitError, "--item needs --target <name>: the fixture always runs A then B")
	case isTarget && item == "":
		return Exit(ExitError, "--target %s needs --item NNN: an item run builds one prepared item", name)
	}
	p, err := pipelineFor(cmd)
	if err != nil {
		return err
	}
	if item != "" {
		return Code(p.RunItem(item, sliceFlag))
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
