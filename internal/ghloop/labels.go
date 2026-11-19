package ghloop

import (
	"context"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"
)

// Proposal is whether an item's issue carries the proposal label, and who added it last (021 R2).
type Proposal struct {
	Labelled bool      // the label is on the issue now
	AddedBy  string    // the login of the label's last `labeled` event ("" if none was found)
	AddedAt  time.Time // when that event happened
}

// ProposedBy reads the issue's labels and events: labels carry no author, but the events API
// records who added each one. Only the last `labeled` event for this label counts, so a
// non-approver re-adding the label after an approver is what decides.
func ProposedBy(ctx context.Context, gh *github.Client, repo string, number int, label string) (Proposal, error) {
	owner, name := SplitRepo(repo)
	var p Proposal
	issue, _, err := gh.Issues.Get(ctx, owner, name, number)
	if err != nil {
		return p, err
	}
	for _, l := range issue.Labels {
		if strings.EqualFold(l.GetName(), label) {
			p.Labelled = true
		}
	}
	if !p.Labelled {
		return p, nil
	}
	opts := &github.ListOptions{PerPage: 100}
	for {
		evs, resp, err := gh.Issues.ListIssueEvents(ctx, owner, name, number, opts)
		if err != nil {
			return p, err
		}
		for _, e := range evs { // oldest first: the last match wins
			if e.GetEvent() == "labeled" && strings.EqualFold(e.GetLabel().GetName(), label) {
				p.AddedBy, p.AddedAt = e.GetActor().GetLogin(), e.GetCreatedAt().Time
			}
		}
		if resp == nil || resp.NextPage == 0 {
			return p, nil
		}
		opts.Page = resp.NextPage
	}
}

// IsApprover reports whether login is in the approver list (case-insensitive, without "@").
func IsApprover(login string, approvers []string) bool {
	for _, a := range approvers {
		if login != "" && strings.EqualFold(strings.TrimPrefix(a, "@"), login) {
			return true
		}
	}
	return false
}
