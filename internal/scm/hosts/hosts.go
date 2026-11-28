// Package hosts builds the scm.Client for a host (037). It is the only package that knows every
// implementation, so callers depend on scm.Client alone.
package hosts

import (
	"context"
	"os"
	"strings"

	"github.com/tosin2013/chalkeus/internal/scm"
	"github.com/tosin2013/chalkeus/internal/scm/bitbucket"
	"github.com/tosin2013/chalkeus/internal/scm/github"
	"github.com/tosin2013/chalkeus/internal/secret"
)

// New returns a client for h acting with token.
func New(h scm.Host, token secret.Value) (scm.Client, error) {
	switch h.KindOrGitHub() {
	case scm.GitHub:
		if h.Kind == "" {
			h = scm.NewHost(scm.GitHub, "", "")
		}
		return github.New(h, token), nil
	case scm.Bitbucket:
		return bitbucket.New(h, token)
	}
	return nil, &scm.Error{Code: scm.ErrInvalid, Msg: string(h.Kind) + " support isn't built yet (roadmap 037)"}
}

// Operator returns a client acting with the operator's own credential for h (research R8). On
// Bitbucket, BITBUCKET_EMAIL makes it send Basic auth (email:token), as an Atlassian API token
// without scopes needs.
func Operator(h scm.Host) (scm.Client, error) {
	tok, err := scm.OperatorCredential(h)
	if err != nil {
		return nil, err
	}
	if h.KindOrGitHub() == scm.Bitbucket {
		return bitbucket.NewBasic(h, strings.TrimSpace(os.Getenv("BITBUCKET_EMAIL")), tok)
	}
	return New(h, tok)
}

// OpenPilotHeads lists the head branches of open pilot/* change requests on repo (readiness and
// the run's open-PR checks).
func OpenPilotHeads(ctx context.Context, c scm.Client, repo string) ([]string, error) {
	crs, err := c.OpenChangeRequests(ctx, repo, "pilot/")
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(crs))
	for _, cr := range crs {
		out = append(out, cr.HeadRef)
	}
	return out, nil
}
