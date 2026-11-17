package ghloop

import (
	"context"
	"fmt"
)

// fakeGH is an in-memory GitHub for loop tests (reference: the former Python pilot's tests/ghfake.py): issues,
// labels, comments and permissions; no network. It implements Reader and Writer.
type fakeGH struct {
	issues      []Issue
	labels      map[string]bool
	comments    map[int][]Comment
	permissions map[string]string
	publisher   string
	ops         []string
	fail        string // an op name, "read" or "write" to simulate a GitHub error
	next        int
	cid         int64
	prs         map[int]PRView
}

func newFake(publisher string, perms map[string]string) *fakeGH {
	if perms == nil {
		perms = map[string]string{}
	}
	return &fakeGH{labels: map[string]bool{}, comments: map[int][]Comment{}, permissions: perms, publisher: publisher,
		next: 1, cid: 1000, prs: map[int]PRView{}}
}

func (f *fakeGH) loop(approvers ...string) *Loop {
	return &Loop{Repo: "o/x-pilot-fixture", Branch: "main", Approvers: approvers, R: f,
		W: func() (Writer, error) { return f, nil }}
}

func (f *fakeGH) readErr() error {
	if f.fail == "read" {
		return &Error{"GitHub unreachable"}
	}
	return nil
}

func (f *fakeGH) Issues(context.Context, string) ([]Issue, error) {
	if err := f.readErr(); err != nil {
		return nil, err
	}
	out := make([]Issue, len(f.issues))
	for i, x := range f.issues {
		x.Labels = append([]string(nil), x.Labels...)
		out[i] = x
	}
	return out, nil
}

func (f *fakeGH) Labels(context.Context, string) (map[string]bool, error) {
	if err := f.readErr(); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for k := range f.labels {
		out[k] = true
	}
	return out, nil
}

func (f *fakeGH) Comments(_ context.Context, _ string, n int) ([]Comment, error) {
	if err := f.readErr(); err != nil {
		return nil, err
	}
	return append([]Comment(nil), f.comments[n]...), nil
}

func (f *fakeGH) Permission(_ context.Context, _, login string) string {
	if p, ok := f.permissions[login]; ok {
		return p
	}
	return "none"
}

func (f *fakeGH) PR(_ context.Context, _ string, n int) (PRView, error) {
	if err := f.readErr(); err != nil {
		return PRView{}, err
	}
	return f.prs[n], nil
}

func (f *fakeGH) write(op string) error {
	if f.fail == op || f.fail == "write" {
		return &Error{fmt.Sprintf("ghops %s failed", op)}
	}
	f.ops = append(f.ops, op)
	return nil
}

func (f *fakeGH) Whoami(context.Context) (string, error) {
	if err := f.write("whoami"); err != nil {
		return "", err
	}
	return f.publisher, nil
}

func (f *fakeGH) EnsureLabels(_ context.Context, _ string, labels []Label) ([]string, error) {
	if err := f.write("ensure-labels"); err != nil {
		return nil, err
	}
	var out []string
	for _, l := range labels {
		f.labels[l.Name] = true
		out = append(out, l.Name)
	}
	return out, nil
}

func (f *fakeGH) CreateIssue(_ context.Context, _, title, body string, labels []string) (int, error) {
	if err := f.write("create-issue"); err != nil {
		return 0, err
	}
	n := f.next
	f.next++
	f.issues = append(f.issues, Issue{Number: n, Title: title, Body: body, Labels: append([]string(nil), labels...), State: "open"})
	return n, nil
}

func (f *fakeGH) issue(n int) *Issue {
	for i := range f.issues {
		if f.issues[i].Number == n {
			return &f.issues[i]
		}
	}
	panic(n)
}

func (f *fakeGH) EditIssue(_ context.Context, _ string, n int, e IssueEdit) error {
	if err := f.write("edit-issue"); err != nil {
		return err
	}
	i := f.issue(n)
	if e.Body != nil {
		i.Body = *e.Body
	}
	if e.Labels != nil {
		i.Labels = append([]string(nil), (*e.Labels)...)
	}
	if e.Title != nil {
		i.Title = *e.Title
	}
	return nil
}

func (f *fakeGH) Comment(_ context.Context, _ string, n int, body string) (int64, error) {
	if err := f.write("comment"); err != nil {
		return 0, err
	}
	return f.addComment(n, body, f.publisher, false, false), nil
}

func (f *fakeGH) CloseIssue(_ context.Context, _ string, n int) error {
	if err := f.write("close-issue"); err != nil {
		return err
	}
	f.issue(n).State = "closed"
	return nil
}

func (f *fakeGH) addComment(n int, body, author string, edited, app bool) int64 {
	f.cid++
	ts := fmt.Sprintf("2026-11-16T10:%02d:00Z", f.cid%60)
	up := ts
	if edited {
		up = "2026-11-16T23:59:00Z"
	}
	f.comments[n] = append(f.comments[n], Comment{ID: f.cid, Body: body, Author: author, CreatedAt: ts, UpdatedAt: up, App: app})
	return f.cid
}

func (f *fakeGH) count(op string) int {
	n := 0
	for _, o := range f.ops {
		if o == op {
			n++
		}
	}
	return n
}
