package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/ghapp"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/pyrepr"
)

// `pilot github sync|drift|watch` and `pilot app-key|app-token` (reference: cmd_github and the
// app-key/app-token branches in cli.py).

func githubErr(cmd *cobra.Command, err error) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "pilot: github: %v\n", err)
	return Code(ExitError)
}

// loopFor builds the GitHub loop for `pilot github` commands.
func loopFor(cfg *config.Config) (*ghloop.Loop, error) {
	r, err := ghloop.NewReader()
	if err != nil {
		return nil, err
	}
	creds, err := credsFor(cfg)
	if err != nil {
		return nil, err
	}
	return &ghloop.Loop{Repo: cfg.Repo(), Branch: cfg.Branch(), Approvers: cfg.Approvers(), R: r,
		W: func() (ghloop.Writer, error) {
			env, err := creds.Env()
			if err != nil {
				return nil, err
			}
			return ghloop.NewWriter(env)
		},
		Snapshot: ghloop.DefaultSnapshot(cfg.Repo(), cfg.Branch())}, nil
}

func githubSetup(cmd *cobra.Command) (*config.Config, *ghloop.Loop, error) {
	l, err := layout()
	if err != nil {
		return nil, nil, err
	}
	cfg, err := config.Load(l.Root, false, true)
	if err != nil {
		return nil, nil, err
	}
	lp, err := loopFor(cfg)
	if err != nil {
		if _, ok := err.(*config.Error); ok {
			return nil, nil, err
		}
		return nil, nil, githubErr(cmd, err)
	}
	return cfg, lp, nil
}

func runGithubSync(cmd *cobra.Command) error {
	_, lp, err := githubSetup(cmd)
	if err != nil {
		return err
	}
	s, err := lp.Sync(nil)
	if err != nil {
		return githubErr(cmd, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "github sync: %d created, %d changed; issues %s\n", s.Created, s.Changed, pyrepr.Value(s.Issues))
	return nil
}

func runGithubDrift(cmd *cobra.Command) error {
	_, lp, err := githubSetup(cmd)
	if err != nil {
		return err
	}
	var run *evidence.Run
	if id, _ := cmd.Flags().GetString("run"); id != "" {
		l, _ := layout()
		run, _ = evidence.LoadRun(l.EvidenceRoot(), id)
		if run == nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "no run %s\n", id)
			return Code(ExitError)
		}
	}
	rep, err := lp.Drift(run)
	if err != nil {
		return githubErr(cmd, err)
	}
	out := cmd.OutOrStdout()
	for _, d := range rep {
		fmt.Fprintln(out, pyrepr.SortedJSON(d.Values))
	}
	fmt.Fprintf(out, "github drift: %d difference(s)\n", len(rep))
	if len(rep) > 0 {
		return Code(ExitFail)
	}
	return nil
}

func runGithubWatch(cmd *cobra.Command) error {
	if d, _ := cmd.Flags().GetBool("detach"); d {
		return notImplemented("github watch --detach")
	}
	if s, _ := cmd.Flags().GetBool("stop"); s {
		return notImplemented("github watch --stop")
	}
	p, err := pipelineFor(cmd)
	if err != nil {
		return err
	}
	id, _ := cmd.Flags().GetString("run")
	return Code(p.Watch(id, 15*time.Second, nil))
}

func appSetup() (ghapp.App, string, error) {
	l, err := layout()
	if err != nil {
		return ghapp.App{}, "", err
	}
	cfg, err := config.Load(l.Root, false, true)
	if err != nil {
		return ghapp.App{}, "", err
	}
	return ghapp.App{Cfg: cfg}, config.OriginRepo(l.Root), nil
}

func runAppKey(cmd *cobra.Command) error {
	a, canary, err := appSetup()
	if err != nil {
		return err
	}
	return Code(a.StoreKey(cmd.InOrStdin(), cmd.OutOrStdout(), canary))
}

func runAppToken(cmd *cobra.Command) error {
	a, canary, err := appSetup()
	if err != nil {
		return err
	}
	return Code(a.Check(cmd.OutOrStdout(), "", canary))
}
