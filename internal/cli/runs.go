package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/tosin2013/chalkeus/internal/audit"
	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/paths"
	"github.com/tosin2013/chalkeus/internal/watcher"
)

func layout() (paths.Layout, error) {
	root, err := paths.Root()
	if err != nil {
		return paths.Layout{}, Exit(ExitError, "%v", err)
	}
	return paths.At(root), nil
}

func runAudit(cmd *cobra.Command, _ []string) error {
	l, err := layout()
	if err != nil {
		return err
	}
	id, _ := cmd.Flags().GetString("run")
	if item, _ := cmd.Flags().GetString("item"); item != "" { // 026: the item view
		name, _ := cmd.Flags().GetString("target")
		if name == "" || id != "" {
			return Exit(ExitError, "--item needs --target <name> (and no --run)")
		}
		cfg, _ := config.Load(l.Root, false, false)
		ok, err := audit.ItemView(l.EvidenceRoot(), name, item, cfg, cmd.OutOrStdout())
		if err != nil {
			return Exit(ExitError, "%v", err)
		}
		if !ok {
			return Code(ExitFail)
		}
		return nil
	}
	r, err := evidence.LoadRun(l.EvidenceRoot(), id)
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	if r == nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "no run found")
		return Code(ExitError)
	}
	cfg, cerr := config.Load(l.Root, false, false)
	if cerr != nil {
		cfg = nil // the reference audits without config rather than failing
	}
	ok, err := audit.Run(r, true, cfg, cmd.OutOrStdout())
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	if !ok {
		return Code(ExitFail)
	}
	return nil
}

func runStatus(cmd *cobra.Command, _ []string) error {
	l, err := layout()
	if err != nil {
		return err
	}
	id, _ := cmd.Flags().GetString("run")
	r, err := evidence.LoadRun(l.EvidenceRoot(), id)
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	if r == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no runs")
		return nil
	}
	state := r.State
	if w := watcher.Status(r.Dir, r.ID); w.Running { // contracts/watcher.md; never saved in run.json
		logPath := w.Log
		if rel, err := filepath.Rel(l.Root, w.Log); err == nil {
			logPath = rel
		}
		state = map[string]any{}
		for k, v := range r.State {
			state[k] = v
		}
		state["watcher"] = map[string]any{"pid": w.Pid, "running": true, "log": logPath}
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(state); err != nil {
		return Exit(ExitError, "%v", err)
	}
	_, _ = fmt.Fprint(cmd.OutOrStdout(), b.String())
	return nil
}
