package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/author"
	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/doctor"
	"github.com/tosin2013/specfoundry/internal/fixture"
	"github.com/tosin2013/specfoundry/internal/gate"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/grader"
	"github.com/tosin2013/specfoundry/internal/images"
	"github.com/tosin2013/specfoundry/internal/readiness"
)

// doctorCmd is `pilot doctor --target <name> [--item NNN] [--baseline]` (014): a read-only
// preflight that lists every problem in one pass. It never prints hidden-test contents.
func doctorCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "doctor --target <name> [--item NNN] [--baseline]",
		Short: "read-only preflight: check a target's profile, cluster and GitHub access, and one item's readiness, reporting every problem at once",
		Args:  cobra.NoArgs,
		RunE:  runDoctor,
	}
	addTargetFlag(c)
	_ = c.MarkFlagRequired("target")
	c.Flags().String("item", "", "also check roadmap item NNN: every readiness rule, without stopping at the first problem")
	c.Flags().Bool("baseline", false, "with --item: on the cluster, check that the install and test commands pass on today's code, offline")
	return c
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	item, _ := cmd.Flags().GetString("item")
	withBaseline, _ := cmd.Flags().GetBool("baseline")
	if withBaseline && item == "" {
		return Exit(ExitError, "--baseline needs --item NNN")
	}
	var r doctor.Report
	defer func() { r.Print(out) }()
	l, err := layout()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Profile.
	cfg, err := config.Load(l.Root, false, false)
	if !r.OK("profile", "pilot.env", err, "") {
		return Code(ExitFail)
	}
	t, err := loadTarget(cmd, l, cfg)
	if !r.OK("profile", "target profile", err, fmt.Sprintf("%s (%s@%s), toolchain %s, level %s", t.Name, t.Repo, t.Branch, t.Toolchain, t.SecurityLevel)) {
		return Code(ExitFail)
	}
	if t.Fixture {
		return Exit(ExitError, "doctor checks a --target repository, not the fixture")
	}
	if t.SecurityLevel == config.LevelRecommended {
		_, err := t.AuthorToken()
		r.OK("profile", "test author token (recommended level)", err, "present, mode 0600")
	}

	// Machine and cluster.
	if hosts := cfg.ModelHosts(); len(hosts) == 0 {
		r.Add("cluster", "model host", doctor.Fail, "PILOT_MODEL_EGRESS_HOSTS is empty in pilot.env")
	} else {
		r.Add("cluster", "model host", doctor.Pass, fmt.Sprintf("%d host(s) configured", len(hosts)))
	}
	// The model key: one tiny request from this machine with the worker's model (a blocked key
	// used to show up only after every attempt had failed).
	if probe, err := cfg.ProbeModel(ctx); err != nil {
		r.Add("cluster", "model key", doctor.Fail, err.Error())
	} else {
		switch {
		case probe.Status >= 200 && probe.Status < 300:
			r.Add("cluster", "model key", doctor.Pass, "the provider accepts the worker's key (provider "+probe.Provider+")")
		case probe.Status == 401 || probe.Status == 403:
			r.Add("cluster", "model key", doctor.Fail, fmt.Sprintf("the provider rejected the worker's key (%d): %s. Fix or replace the key, then run `pilot/pilot import-opencode` again",
				probe.Status, probe.Message))
		default:
			r.Add("cluster", "model key", doctor.Warn, fmt.Sprintf("couldn't confirm the key: the provider answered %d to a one-token test request", probe.Status))
		}
	}
	upOK := false
	if _, err := os.Stat(l.UpOK); err != nil {
		r.Add("cluster", "pilot up", doctor.Fail, "`pilot/pilot up` has not completed")
	} else {
		upOK = true
		r.Add("cluster", "pilot up", doctor.Pass, "")
		running, err1 := images.Load(l)
		lock, err2 := images.ReadLock(l)
		if err := errors.Join(err1, err2); err != nil {
			r.Add("cluster", "images", doctor.Fail, err.Error())
		} else {
			stale := []string{}
			for k, key := range map[string]string{"worker": "WORKER_IMAGE", "tools": "TOOLS_IMAGE", "egress": "EGRESS_IMAGE"} {
				if !doctor.DigestMatches(running[k], lock[key]) {
					stale = append(stale, k)
				}
			}
			if len(stale) > 0 {
				r.Add("cluster", "images", doctor.Fail, fmt.Sprintf("the cluster runs older images than pilot/images.lock (%v); run `pilot/pilot up`", stale))
			} else {
				r.Add("cluster", "images", doctor.Pass, "match pilot/images.lock")
			}
		}
	}

	// GitHub access.
	gh, err := ghloop.OperatorClient()
	if !r.OK("github", "your GitHub token", err, "") {
		return Code(ExitFail)
	}
	for _, repo := range []string{t.Repo, t.CompanionRepo} {
		owner, name := ghloop.SplitRepo(repo)
		_, _, err := gh.Repositories.Get(ctx, owner, name)
		r.OK("github", "can read "+repo, err, "")
	}
	id, err := author.ResolveIdentity(ctx, t)
	switch {
	case err != nil:
		r.Add("github", "test author identity", doctor.Warn, err.Error()+" (needed only to draft hidden tests with the test author)")
	case id.CanWriteTarget:
		r.Add("github", "test author identity", doctor.Pass, fmt.Sprintf("%s (level %s; it can also write the target)", id.Login, id.Level))
	default:
		r.Add("github", "test author identity", doctor.Pass, fmt.Sprintf("%s (level %s; companion only)", id.Login, id.Level))
	}
	owner, name := ghloop.SplitRepo(t.Repo)
	if _, resp, err := gh.Repositories.GetBranchProtection(ctx, owner, name, t.Branch); err != nil {
		switch {
		case resp != nil && resp.StatusCode == http.StatusNotFound:
			r.Add("github", "branch protection", doctor.Warn, t.Repo+"@"+t.Branch+" is not protected; protect it so changes arrive only by reviewed PR (important at the minimal level)")
		default:
			r.Add("github", "branch protection", doctor.Warn, "could not check: "+err.Error())
		}
	} else {
		r.Add("github", "branch protection", doctor.Pass, t.Branch+" is protected")
	}
	if item == "" {
		return doctorExit(&r)
	}

	// The item: every readiness rule.
	deps := readiness.Deps{Src: gate.ExecStatus{}, AllRules: true,
		Clone: func(repo, branch string) (string, string, error) { return fixture.FreshClone(repo, branch, 1) },
		OpenPRHeads: func(repo string) ([]string, error) {
			prs, err := ghloop.OpenPilotPRs(ctx, gh, repo)
			heads := make([]string, 0, len(prs))
			for _, p := range prs {
				heads = append(heads, p.HeadRef)
			}
			return heads, err
		}}
	res, err := readiness.Check(deps, t, item, 0, "")
	if !r.OK("item", "readiness", err, "") {
		return Code(ExitFail)
	}
	r.Readiness(res)
	if !withBaseline {
		return doctorExit(&r)
	}

	// The project on the cluster, offline: toolchain, install, the project's tests on today's code.
	if !upOK {
		r.Add("baseline", "project tests offline", doctor.Skip, "needs `pilot/pilot up`")
		return doctorExit(&r)
	}
	if res.Commit == "" || res.SpecDir == "" {
		r.Add("baseline", "project tests offline", doctor.Skip, "needs the item's spec and a readable target")
		return doctorExit(&r)
	}
	p, err := pipelineFor(cmd)
	if !r.OK("baseline", "pipeline", err, "") {
		return doctorExit(&r)
	}
	overlay, hasHidden, err := doctorOverlay(t, item, res)
	if !r.OK("baseline", "hidden tests overlay", err, "") {
		return doctorExit(&r)
	}
	_, _ = fmt.Fprintln(out, "doctor: running the item's baseline on the cluster (a few minutes)...")
	dir := filepath.Join(l.Build, "doctor", time.Now().UTC().Format("20060102T150405Z"))
	defer func() { _ = os.RemoveAll(dir) }()
	report, _, err := p.ItemBaseline(dir, "doctor-"+filepath.Base(dir), item, res, overlay, false)
	if !r.OK("baseline", "toolchain, install and prefetch", err, "") {
		return doctorExit(&r)
	}
	by := map[string]string{}
	checks, _ := report["checks"].([]any)
	for _, x := range checks {
		c, _ := x.(map[string]any)
		by[fmt.Sprint(c["id"])] = fmt.Sprint(c["result"])
	}
	if by["G4_visible_tests"] == "pass" {
		r.Add("baseline", "project tests offline", doctor.Pass, "the test command passes on today's code with no network")
	} else {
		r.Add("baseline", "project tests offline", doctor.Fail, "the test command fails on today's code with no network (see BASELINE_SUITE_FAILS in docs/troubleshooting.md)")
	}
	switch {
	case !hasHidden:
		r.Add("baseline", "hidden tests fail on today's code", doctor.Skip, "no hidden tests yet")
	case by["G5_hidden_tests"] == "pass":
		r.Add("baseline", "hidden tests fail on today's code", doctor.Pass, "")
	default:
		r.Add("baseline", "hidden tests fail on today's code", doctor.Fail, "the hidden tests already pass (BASELINE_HIDDEN_PASSES)")
	}
	return doctorExit(&r)
}

// doctorOverlay is the item's hidden tests, or, when there are none yet, a placeholder file that
// isn't a test, so the baseline still proves the project's own tests run offline.
func doctorOverlay(t config.Target, item string, res readiness.Result) ([]byte, bool, error) {
	if res.HiddenFiles > 0 {
		cdir, commit, err := fixture.FreshClone(t.CompanionRepo, "", 1)
		if err != nil {
			return nil, false, err
		}
		defer func() { _ = os.RemoveAll(cdir) }()
		tarb, _, err := grader.Overlay(cdir, t.CompanionRepo, commit, t.CompanionItemDir(item), item)
		return tarb, true, err
	}
	d, err := os.MkdirTemp("", "pilot-doctor-")
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = os.RemoveAll(d) }()
	if err := os.WriteFile(filepath.Join(d, ".specfoundry-doctor"), []byte("placeholder, not a test\n"), 0o644); err != nil {
		return nil, false, err
	}
	tarb, _, err := grader.Overlay(d, t.CompanionRepo, "none", ".", item)
	return tarb, false, err
}

func doctorExit(r *doctor.Report) error {
	if r.Failed() {
		return Code(ExitFail)
	}
	return nil
}
