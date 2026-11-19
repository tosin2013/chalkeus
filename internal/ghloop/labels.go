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

// PlanningPRs lists the repository's open pull requests that change its roadmap or specs/ (not
// the pilot's own branches): the PRs whose merge makes a new item proposable (021).
func PlanningPRs(ctx context.Context, gh *github.Client, repo string) ([]string, error) {
	owner, name := SplitRepo(repo)
	prs, _, err := gh.PullRequests.List(ctx, owner, name, &github.PullRequestListOptions{State: "open", ListOptions: github.ListOptions{PerPage: 50}})
	if err != nil {
		return nil, err
	}
	var out []string
	for _, pr := range prs {
		if strings.HasPrefix(pr.GetHead().GetRef(), "pilot/") {
			continue
		}
		files, _, err := gh.PullRequests.ListFiles(ctx, owner, name, pr.GetNumber(), &github.ListOptions{PerPage: 100})
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if p := f.GetFilename(); p == ".specify/memory/roadmap.md" || strings.HasPrefix(p, "specs/") {
				out = append(out, pr.GetHTMLURL()+" ("+pr.GetTitle()+")")
				break
			}
		}
	}
	return out, nil
}
