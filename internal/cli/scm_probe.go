package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/scm"
	"github.com/tosin2013/chalkeus/internal/scm/hosts"
)

// probeOnHost checks a publisher token file for a repository on a non-GitHub host (037 R4): it
// loads PILOT_SCM_TOKEN (mode 0600) and runs the host's ProbePublisher. GitHub keeps
// ghloop.VerifyFile.
func probeOnHost(h scm.Host, envFile, repo string) scm.TokenProbeResult {
	c, err := ghloop.LoadCredential(envFile)
	if err != nil {
		return scm.TokenProbeResult{Repo: repo, Code: scm.TokenWrongKind, Lines: []scm.ProbeLine{{Name: "token file", Detail: err.Error()}}}
	}
	sc, err := hosts.New(h, c.Token)
	if err != nil {
		return scm.TokenProbeResult{Repo: repo, Code: scm.TokenWrongKind, Lines: []scm.ProbeLine{{Name: "host", Detail: err.Error()}}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return sc.ProbePublisher(ctx, repo)
}

// reportProbe prints a host probe the way ghloop.Report prints GitHub's.
func reportProbe(w io.Writer, h scm.Host, r scm.TokenProbeResult) {
	status := "NOT OK"
	if r.OK() {
		status = "OK"
	}
	fmt.Fprintf(w, "publisher token: %s (%s, %s)\n", status, h.Name, r.Kind)
	fmt.Fprintf(w, "  repo: %s\n", r.Repo)
	for _, l := range r.Lines {
		mark := "ok"
		if !l.OK {
			mark = "problem"
		}
		if l.Detail != "" {
			fmt.Fprintf(w, "  %s: %s: %s\n", mark, l.Name, l.Detail)
		} else {
			fmt.Fprintf(w, "  %s: %s\n", mark, l.Name)
		}
	}
	if r.Code != "" {
		fmt.Fprintf(w, "  code: %s (docs/troubleshooting.md)\n", r.Code)
	}
}
