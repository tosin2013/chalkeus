package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/tosin2013/chalkeus/internal/scm"
	"github.com/tosin2013/chalkeus/internal/scm/hosts"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/gate"
	"github.com/tosin2013/chalkeus/internal/paths"
	"github.com/tosin2013/chalkeus/internal/readiness"
)

// activeRun is the latest run when it is in progress or paused (the reference active_run).
func activeRun(root string) (*evidence.Run, error) {
	r, err := evidence.LoadRun(root, "")
	if err != nil || r == nil {
		return nil, err
	}
	s, _ := r.State["state"].(string)
	if s == "in-progress" || strings.HasPrefix(s, "paused") {
		return r, nil
	}
	return nil, nil
}

func runGate(cmd *cobra.Command, args []string) error {
	l, err := layout()
	if err != nil {
		return err
	}
	if item, _ := cmd.Flags().GetString("item"); item != "" {
		return runItemReadiness(cmd, l, item)
	}
	if name, _ := cmd.Flags().GetString("target"); name != "" && name != config.FixtureTarget {
		return Exit(ExitError, "gate --target %s needs --item NNN (gate {A|B} is the fixture's)", name)
	}
	task := args[0]
	item := paths.Tasks[task]
	run, err := activeRun(l.EvidenceRoot())
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	seq, outDir := 0, ""
	if run != nil {
		if seq, err = run.NextGateSeq(); err != nil {
			return Exit(ExitError, "%v", err)
		}
		outDir = filepath.Join(run.Dir, "gate")
	}
	src := gate.ExecStatus{}
	var d gate.Decision
	if root, _ := cmd.Flags().GetString("root"); root != "" {
		d, err = gate.Evaluate(src, task, item, root, seq, outDir, "local:"+root, nil)
	} else {
		cfg, cerr := config.Load(l.Root, false, true)
		if cerr != nil {
			return cerr
		}
		d, err = gate.EvaluateFresh(src, cfg.Repo(), cfg.Branch(), task, item, seq, outDir,
			func(repo, branch string) (string, string, error) {
				return fixture.FreshClone(cfg.FixtureHost(), repo, branch, 1)
			})
	}
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	if run != nil {
		if err := run.Event("gate", task, nil, "gate_"+d.Decision, strings.Join(d.Reasons, ",")); err != nil {
			return Exit(ExitError, "%v", err)
		}
	}
	raw, _ := json.Marshal(d)
	var out bytes.Buffer
	_ = json.Indent(&out, raw, "", "  ")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), out.String())
	if d.Decision != "allow" {
		return Code(ExitGateRefusal)
	}
	return nil
}

// runItemReadiness is `./chalkeus forge gate --target <name> --item NNN` (014 US2): the readiness rules
// only, against a fresh clone of the target; no cluster, no run.
func runItemReadiness(cmd *cobra.Command, l paths.Layout, item string) error {
	name, _ := cmd.Flags().GetString("target")
	if name == "" || name == config.FixtureTarget {
		return Exit(ExitError, "--item needs --target <name>: the fixture always runs A then B")
	}
	cfg, err := config.Load(l.Root, false, false)
	if err != nil {
		return err
	}
	t, err := loadTarget(cmd, l, cfg)
	if err != nil {
		return err
	}
	sc, err := hosts.Operator(t.Host())
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	clone := func(repo, branch string) (string, string, error) {
		return fixture.FreshClone(t.Host(), repo, branch, 1)
	}
	if root, _ := cmd.Flags().GetString("root"); root != "" {
		// Check a local checkout of the target instead of its branch on GitHub: its committed
		// HEAD, copied, because readiness deletes its clone. The companion and open PRs are
		// still read from GitHub.
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "gate: checking the committed HEAD of %s instead of %s@%s\n", root, t.Repo, t.Branch)
		clone = func(repo, branch string) (string, string, error) {
			if repo != t.Repo {
				return fixture.FreshClone(t.Host(), repo, branch, 1)
			}
			return LocalClone(root)
		}
	}
	res, err := readiness.Check(readiness.Deps{Src: gate.ExecStatus{},
		Clone: clone,
		OpenPRHeads: func(repo string) ([]string, error) {
			return hosts.OpenPilotHeads(context.Background(), sc, repo)
		}}, t, item, 0, "")
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	raw, _ := json.Marshal(res)
	var out bytes.Buffer
	_ = json.Indent(&out, raw, "", "  ")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), out.String())
	if res.Decision != "allow" {
		return Code(ExitGateRefusal)
	}
	return nil
}

// LocalClone copies the committed HEAD of a local git checkout into a new temporary directory
// (the caller removes it) and returns that directory and the commit.
func LocalClone(root string) (string, string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	d, err := os.MkdirTemp("", "pilot-gate-root-")
	if err != nil {
		return "", "", err
	}
	if _, err := fixture.Git(scm.Host{}, "", "clone", "--quiet", "--no-hardlinks", abs, d); err != nil {
		_ = os.RemoveAll(d)
		return "", "", fmt.Errorf("--root %s is not a git checkout: %v", root, err)
	}
	commit, err := fixture.Git(scm.Host{}, d, "rev-parse", "HEAD")
	if err != nil {
		_ = os.RemoveAll(d)
		return "", "", err
	}
	return d, strings.TrimSpace(commit), nil
}
