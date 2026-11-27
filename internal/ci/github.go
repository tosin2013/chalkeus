package ci

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	gh "github.com/google/go-github/v85/github"
)

// Reader fetches a target's CI inputs from its Git host. GitHub Actions is the first; other
// hosts (037) add theirs behind the same interface.
type Reader interface {
	// Workflows returns the CI configuration files at ref, path → content; empty when the
	// repository has none.
	Workflows(ctx context.Context, repo, ref string) (map[string][]byte, error)
	// File returns one file at ref, or an error when it doesn't exist.
	File(ctx context.Context, repo, ref, path string) ([]byte, error)
	// LatestRuns returns each gating workflow's latest completed run's jobs on branch.
	LatestRuns(ctx context.Context, repo, branch string) ([]JobRun, error)
}

// GitHub reads GitHub Actions with the operator's read credential.
type GitHub struct{ C *gh.Client }

const workflowDir = ".github/workflows"

func split(repo string) (string, string) {
	o, n, _ := strings.Cut(repo, "/")
	return o, n
}

func (g GitHub) Workflows(ctx context.Context, repo, ref string) (map[string][]byte, error) {
	o, n := split(repo)
	_, dir, resp, err := g.C.Repositories.GetContents(ctx, o, n, workflowDir, &gh.RepositoryContentGetOptions{Ref: ref})
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return map[string][]byte{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, f := range dir {
		name := f.GetName()
		if f.GetType() != "file" || (!strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml")) {
			continue
		}
		raw, err := g.File(ctx, repo, ref, f.GetPath())
		if err != nil {
			return nil, err
		}
		out[f.GetPath()] = raw
	}
	return out, nil
}

func (g GitHub) File(ctx context.Context, repo, ref, path string) ([]byte, error) {
	o, n := split(repo)
	f, _, _, err := g.C.Repositories.GetContents(ctx, o, n, path, &gh.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("%s is not a file", path)
	}
	s, err := f.GetContent()
	return []byte(s), err
}

func (g GitHub) LatestRuns(ctx context.Context, repo, branch string) ([]JobRun, error) {
	o, n := split(repo)
	runs, _, err := g.C.Actions.ListRepositoryWorkflowRuns(ctx, o, n, &gh.ListWorkflowRunsOptions{
		Branch: branch, Status: "completed", ListOptions: gh.ListOptions{PerPage: 50}})
	if err != nil {
		return nil, err
	}
	latest := map[int64]*gh.WorkflowRun{} // per workflow: the newest completed push run
	for _, r := range runs.WorkflowRuns {
		if ev := r.GetEvent(); ev != "push" && ev != "merge_group" {
			continue
		}
		if cur, ok := latest[r.GetWorkflowID()]; !ok || r.GetCreatedAt().After(cur.GetCreatedAt().Time) {
			latest[r.GetWorkflowID()] = r
		}
	}
	var out []JobRun
	for _, r := range latest {
		jobs, _, err := g.C.Actions.ListWorkflowJobs(ctx, o, n, r.GetID(), &gh.ListWorkflowJobsOptions{ListOptions: gh.ListOptions{PerPage: 100}})
		if err != nil {
			return nil, err
		}
		for _, j := range jobs.Jobs {
			jr := JobRun{Workflow: r.GetName(), Job: j.GetName(), Conclusion: j.GetConclusion(), URL: j.GetHTMLURL(),
				When: r.GetCreatedAt().Format(time.RFC3339)}
			if jr.Conclusion == "failure" {
				for _, s := range j.Steps {
					if s.GetConclusion() == "failure" {
						jr.FailedStep = s.GetName()
						break
					}
				}
				jr.ErrorLine = g.errorLine(ctx, o, n, j.GetID(), jr.FailedStep)
			}
			out = append(out, jr)
		}
	}
	sort.Slice(out, func(i, k int) bool {
		if out[i].Workflow != out[k].Workflow {
			return out[i].Workflow < out[k].Workflow
		}
		return out[i].Job < out[k].Job
	})
	return out, nil
}

// errorLine reads the failed job's log (best effort; "" on any error) and picks the line that
// explains the failure, within the failed step's section when it can find it.
func (g GitHub) errorLine(ctx context.Context, owner, name string, jobID int64, step string) string {
	u, _, err := g.C.Actions.GetWorkflowJobLogs(ctx, owner, name, jobID, 2)
	if err != nil || u == nil {
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ""
	}
	resp, err := http.DefaultClient.Do(req) // a pre-signed URL: no credential is sent
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	return ErrorLine(StepSection(string(raw), step))
}

// StepSection cuts a job log to the failed step's part when the step's `##[group]Run` header
// can be found; otherwise it returns the whole log. GitHub names a `run` step's group after its
// command, so the section is located by the last group before the job's first error marker.
func StepSection(log, _ string) string {
	end := strings.Index(log, "##[error]")
	if end < 0 {
		return log
	}
	start := strings.LastIndex(log[:end], "##[group]Run ")
	if start < 0 {
		return log
	}
	stop := end + strings.Index(log[end:], "\n")
	if stop < end {
		stop = len(log)
	}
	return log[start:stop]
}
