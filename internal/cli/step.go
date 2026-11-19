package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/author"
	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/readiness"
	"github.com/tosin2013/specfoundry/internal/step"
	"github.com/tosin2013/specfoundry/internal/verbose"
	"github.com/tosin2013/specfoundry/internal/watcher"
)

// stepCmd is `pilot step` (021, contracts/step.md): the laptop manager loop. The owner starts it
// in their own terminal; the session that drives runs never does (it may draft tests, 024).
func stepCmd() *cobra.Command {
	c := &cobra.Command{Use: "step", Args: cobra.NoArgs,
		Short: "021: carry approver-labelled items through every automatic step, waiting at each human gate; never run by the session that drives runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n, _ := cmd.Flags().GetCount("verbose")
			_, envSet := os.LookupEnv("PILOT_VERBOSE")
			if l := watchVerbosity(n, envSet); l != verbose.Level() {
				verbose.Set(l, cmd.ErrOrStderr())
			}
			e, _, err := authorEnv(cmd)
			if err != nil {
				return err
			}
			t := e.Target
			if !stepGitHubOn(t) {
				return Exit(ExitError, "step mode needs GitHub approvals for target %s (PILOT_GITHUB_APPROVALS=on in its profile or pilot.env): items are proposed on their issues and every gate is answered on GitHub", t.Name)
			}
			p, err := pipelineFor(cmd)
			if err != nil {
				return err
			}
			approvers := p.Cfg.Approvers()
			if len(approvers) == 0 {
				return Exit(ExitError, "step mode needs PILOT_APPROVERS in pilot.env: only an approver's label proposes an item")
			}
			loop, err := p.GitHubLoop()
			if err != nil {
				return Exit(ExitError, "%v", err)
			}
			opGH, err := ghloop.OperatorClient()
			if err != nil {
				return Exit(ExitError, "%v", err)
			}
			dir := filepath.Join(e.EvidenceRoot, "step", t.Name)
			log, err := step.OpenLog(dir)
			if err != nil {
				return err
			}
			issues := func(ctx context.Context) (map[string]int, error) {
				all, err := loop.R.Issues(ctx, t.Repo)
				if err != nil {
					return nil, err
				}
				idx, _ := ghloop.ByItem(all)
				m := map[string]int{}
				for _, x := range idx {
					m[x.Item] = x.Issue.Number
				}
				return m, nil
			}
			src := step.Source{Status: targetStatus(t), Issues: issues, Approvers: approvers, AutoDraft: t.AuthorAuto == "on", Log: log,
				Proposal: func(ctx context.Context, n int) (ghloop.Proposal, error) {
					return ghloop.ProposedBy(ctx, opGH, t.Repo, n, t.ProposeLabel)
				},
				Ready: func(item string) (readiness.Result, error) { return e.Ready(t, item) },
				TestsPR: func(ctx context.Context, item string) (bool, error) {
					id, err := e.Identity(ctx, t)
					if err != nil {
						return false, err
					}
					open, err := e.OpenAuthorPRs(ctx, id, t.CompanionRepo, "author/"+item+"-")
					return len(open) > 0, err
				},
				Run: func(item string) (step.RunInfo, bool) {
					return step.NewestRun(e.EvidenceRoot, log.RunDirs(), t.Name, item, func(d string) bool {
						return watcher.Status(d, filepath.Base(d)).Running
					})
				}}
			out := cmd.OutOrStdout()
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			m := &step.Manager{Target: t.Name, Src: src, Log: log, Dir: dir, Out: out, Issues: issues,
				Sync: func(ctx context.Context) error { _, err := loop.Sync(nil); return err },
				Draft: func(ctx context.Context, item string) (string, error) {
					if t.AuthorAuto != "on" {
						return "auto-draft is off", nil
					}
					w, err := authorWatchEnv(e, p)
					if err != nil {
						return "", err
					}
					w.Only = item
					return author.Pass(ctx, w)
				},
				Pilot: func(ctx context.Context, args []string, logPath string) (int, error) {
					// A doctor or run that has started finishes even if the manager is stopped (T010):
					// killing `run` while it prepares would leave a half-made run.
					return runPilot(context.WithoutCancel(ctx), exe, p.L.Root, args, logPath)
				},
				Post: func(ctx context.Context, n int, body string) error {
					w, err := loop.W()
					if err != nil {
						return err
					}
					_, err = w.Comment(ctx, t.Repo, n, body)
					return err
				}}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			once, _ := cmd.Flags().GetBool("once")
			fmt.Fprintf(out, "step: target %s, label %q (approvers: %s), every %d s; Ctrl-C stops it\n", t.Name, t.ProposeLabel, strings.Join(approvers, ", "), t.StepIntervalS)
			for {
				failed, err := m.Pass(ctx)
				if err != nil {
					fmt.Fprintf(out, "step: %v\n", err)
					if once {
						return Exit(ExitError, "%v", err)
					}
				}
				if once {
					if failed {
						return Code(ExitFail)
					}
					return nil
				}
				select {
				case <-ctx.Done():
					_ = log.Add(step.Entry{Action: "step_stopped", Reason: "interrupted"})
					fmt.Fprintln(out, "step: stopped")
					return nil
				case <-time.After(time.Duration(t.StepIntervalS) * time.Second):
				}
			}
		}}
	addTargetFlag(c)
	_ = c.MarkFlagRequired("target")
	c.Flags().Bool("once", false, "one pass, then exit (0: nothing failed; 1: an item's doctor or run failed)")
	return c
}

// runPilot runs the pilot binary itself (research R3) with its output in logPath. A run started
// this way is recorded as started by step mode (PILOT_STARTED_BY, FR-007).
func runPilot(ctx context.Context, exe, root string, args []string, logPath string) (int, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return 0, err
	}
	f, err := os.Create(logPath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	c := exec.CommandContext(ctx, exe, args...)
	c.Dir, c.Stdout, c.Stderr = root, f, f
	c.Env = append(os.Environ(), "PILOT_STARTED_BY=step", "PILOT_VERBOSE=0")
	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}

// stepGitHubOn: step mode needs GitHub approvals. Issues follow them, as for runs (an item run
// keeps its issues on whenever approvals are on, pipeline.issuesOn), so PILOT_GITHUB_ISSUES
// isn't required separately.
func stepGitHubOn(t config.Target) bool { return t.GithubApprovals == "on" }
