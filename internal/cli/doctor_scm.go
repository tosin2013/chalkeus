package cli

import (
	"context"
	"fmt"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/doctor"
	"github.com/tosin2013/chalkeus/internal/scm"
)

// doctorHostRows adds 037's rows for a target on GitLab or Bitbucket (contracts/checks.md): the
// publisher credential reaches only the target, and every approver resolves to an id with write
// access. A check that can't be evaluated fails (constitution II).
func doctorHostRows(ctx context.Context, r *doctor.Report, sc scm.Client, t config.Target, cfg *config.Config) {
	h := sc.Host()
	pr := probeOnHost(h, t.PublisherEnv, t.Repo)
	if pr.OK() {
		r.Add("scm", "publisher credential", doctor.Pass, fmt.Sprintf("%s for %s only", pr.Kind, t.Repo))
	} else {
		why := pr.Code
		for _, l := range pr.Lines {
			if !l.OK {
				why += ": " + l.Name + ": " + l.Detail
				break
			}
		}
		r.Add("scm", "publisher credential", doctor.Fail, why+" ("+t.PublisherEnv+")")
	}
	approvers := t.ApproversOr(cfg)
	if len(approvers) == 0 {
		r.Add("scm", "approvers", doctor.Fail, "no FORGE_APPROVERS: set the account ids allowed to approve a status change in the target's profile")
		return
	}
	for _, given := range approvers {
		a, err := sc.ResolveAccount(ctx, given)
		if err != nil {
			r.Add("scm", "approver "+given, doctor.Fail, err.Error())
			continue
		}
		p, err := sc.Permission(ctx, t.Repo, a)
		switch {
		case err != nil:
			r.Add("scm", "approver "+given, doctor.Fail, "can't verify approvers' write access: "+err.Error())
		case !p.CanWrite():
			r.Add("scm", "approver "+given, doctor.Fail, fmt.Sprintf("%s has %s access to %s; approvers need write", a.Name, p, t.Repo))
		default:
			r.Add("scm", "approver "+given, doctor.Pass, fmt.Sprintf("%s (%s)", a.Name, p))
		}
	}
}
