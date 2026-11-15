package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/evidence"
	"github.com/tosin2013/specfoundry/internal/fixture"
	"github.com/tosin2013/specfoundry/internal/gate"
	"github.com/tosin2013/specfoundry/internal/paths"
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
			func(repo, branch string) (string, string, error) { return fixture.FreshClone(repo, branch, 1) })
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
