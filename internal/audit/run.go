package audit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/evidence"
)

// Run audits a run directory, writes audit.jsonl, records an audit_done event, rewrites
// summary.md and prints one line per check (the reference audit.run). cfg may be nil.
func Run(r *evidence.Run, final bool, cfg *config.Config, out io.Writer) (bool, error) {
	keyFiles, envFiles, cleanup := secretFiles(cfg)
	defer cleanup()
	rows, err := Dir(r.Dir, keyFiles, envFiles, final)
	if err != nil {
		return false, err
	}
	auditFile := filepath.Join(r.Dir, "audit.jsonl")
	_ = os.Remove(auditFile)
	pass := 0
	for _, row := range rows {
		if err := evidence.AppendJSONL(auditFile, row); err != nil {
			return false, err
		}
		mark := "FAIL"
		if row.Result == "pass" {
			mark = "PASS"
			pass++
		}
		_, _ = fmt.Fprintf(out, "  %s %s %s\n", mark, row.ID, row.Description)
	}
	if err := r.Event("audit", nil, nil, "audit_done", fmt.Sprintf("%d/%d", pass, len(rows))); err != nil {
		return false, err
	}
	if _, err := r.WriteSummary(); err != nil {
		return false, err
	}
	return pass == len(rows), nil
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

// secretFiles are the files whose contents must never appear in evidence (AU08): the model key
// files, a temp file of the model hosts, and the publisher/App token env files. cleanup removes
// the temp file. cfg may be nil.
func secretFiles(cfg *config.Config) (keyFiles, envFiles []string, cleanup func()) {
	cleanup = func() {}
	if cfg != nil {
		for _, p := range cfg.ModelFiles() {
			if filepath.Base(p) != "opencode.json" && exists(p) {
				keyFiles = append(keyFiles, p)
			}
		}
		if hosts := cfg.ModelHosts(); len(hosts) > 0 { // the provider host must not appear in evidence either
			f, err := os.CreateTemp("", "model-hosts")
			if err == nil {
				_, _ = f.WriteString(strings.Join(hosts, "\n") + "\n")
				_ = f.Close()
				name := f.Name()
				cleanup = func() { _ = os.Remove(name) }
				keyFiles = append(keyFiles, f.Name())
			}
		}
		if exists(cfg.PublisherEnv) {
			envFiles = append(envFiles, cfg.PublisherEnv)
		}
		if p := filepath.Join(cfg.ConfigDir, "app-token.env"); exists(p) { // 004: the App token too
			envFiles = append(envFiles, p)
		}
	}
	return keyFiles, envFiles, cleanup
}
