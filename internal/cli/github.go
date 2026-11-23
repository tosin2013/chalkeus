package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/ghapp"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/pyrepr"
	"github.com/tosin2013/chalkeus/internal/watcher"
)

// `./chalkeus forge github sync|drift|watch` and `./chalkeus forge app-key|app-token` (reference: cmd_github and the
// app-key/app-token branches in cli.py).

func githubErr(cmd *cobra.Command, err error) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "chalkeus: github: %v\n", err)
	return Code(ExitError)
}

// loopFor builds the GitHub loop for `./chalkeus forge github` commands.
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
	detach, _ := cmd.Flags().GetBool("detach")
	stop, _ := cmd.Flags().GetBool("stop")
	if detach || stop {
		return watchControl(cmd, detach)
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

// watchControl is `github watch --run <id> --detach | --stop` (contracts/watcher.md).
func watchControl(cmd *cobra.Command, detach bool) error {
	l, err := layout()
	if err != nil {
		return err
	}
	id, _ := cmd.Flags().GetString("run")
	run, _ := evidence.LoadRun(l.EvidenceRoot(), id)
	if run == nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "no run %s\n", id)
		return Code(ExitError)
	}
	out := cmd.OutOrStdout()
	logPath := watcher.LogFile(run.Dir)
	if r, err := filepath.Rel(l.Root, logPath); err == nil {
		logPath = r
	}
	if !detach {
		pid, err := watcher.Stop(run.Dir, run.ID)
		if errors.Is(err, watcher.ErrNotRunning) {
			fmt.Fprintf(out, "no watcher running for run %s\n", run.ID)
			return nil
		}
		if err != nil {
			return Exit(ExitError, "could not stop the watcher (pid %d): %v", pid, err)
		}
		fmt.Fprintf(out, "watcher stopped (pid %d)\n", pid)
		return nil
	}
	if _, err := pipelineFor(cmd); err != nil { // the watcher needs `./chalkeus forge up` and a valid config
		return err
	}
	if st, _ := run.State["state"].(string); !strings.HasPrefix(st, "paused") {
		fmt.Fprintf(cmd.ErrOrStderr(), "run %s is %s; nothing to watch\n", run.ID, st)
		return Code(ExitError)
	}
	pid, started, err := watcher.Ensure(l.Root, run.Dir, run.ID, nil)
	if err != nil {
		return Exit(ExitError, "could not start the watcher: %v", err)
	}
	if !started {
		fmt.Fprintf(out, "watcher already running (pid %d). Log: %s\n", pid, logPath)
		return nil
	}
	fmt.Fprintf(out, "watcher started (pid %d); you can close this terminal. Log: %s\n", pid, logPath)
	return nil
}
