package author

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v85/github"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/secret"
)

// Identity is who opens the companion pull request, by security level (roadmap C-14):
// minimal, the operator's own access; recommended, a companion-only token; strict, refused
// until cluster mode (011).
type Identity struct {
	Level          string
	Login          string
	CanWriteTarget bool
	Client         *github.Client
}

// ResolveIdentity finds the level's identity and checks what it can write. At recommended it
// refuses an identity that can write the target; at minimal it records that it can.
func ResolveIdentity(ctx context.Context, t config.Target) (*Identity, error) {
	id := &Identity{Level: t.SecurityLevel}
	switch t.SecurityLevel {
	case config.LevelStrict:
		return nil, refuse(CodeNeedsClusterMode, "the strict level runs the test author as its own Job in cluster mode (roadmap 011), which isn't built yet; use minimal or recommended")
	case config.LevelRecommended:
		tok, err := t.AuthorToken()
		if err != nil {
			return nil, err
		}
		id.Client = ghloop.NewClient(secret.New(tok))
	default:
		c, err := ghloop.OperatorClient()
		if err != nil {
			return nil, err
		}
		id.Client = c
	}
	u, _, err := id.Client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("could not identify the test author's GitHub identity: %v", err)
	}
	id.Login = u.GetLogin()
	canCompanion, err := canPush(ctx, id.Client, t.CompanionRepo)
	if err != nil {
		return nil, fmt.Errorf("companion %s: %v", t.CompanionRepo, err)
	}
	if !canCompanion {
		return nil, fmt.Errorf("%s cannot write the companion %s; the test author needs contents and pull-request write there", id.Login, t.CompanionRepo)
	}
	id.CanWriteTarget, err = canPush(ctx, id.Client, t.Repo)
	if err != nil {
		return nil, fmt.Errorf("target %s: %v", t.Repo, err)
	}
	if id.CanWriteTarget && t.SecurityLevel == config.LevelRecommended {
		return nil, refuse(CodeCanWriteTarget, "%s can write the target %s; at the recommended level the test author's identity must write only the companion (see docs/targets.md)", id.Login, t.Repo)
	}
	return id, nil
}

// canPush reports whether the client can push to repo. A repository the identity can't see
// (404) counts as "can't push".
func canPush(ctx context.Context, c *github.Client, repo string) (bool, error) {
	owner, name := ghloop.SplitRepo(repo)
	r, resp, err := c.Repositories.Get(ctx, owner, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return r.GetPermissions().GetPush(), nil
}
