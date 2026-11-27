package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/ci"
	"github.com/tosin2013/chalkeus/internal/config"
)

type fakeCI struct {
	files   map[string][]byte
	err     error
	pkg     []byte
	runs    []ci.JobRun
	runsErr error
}

func (f fakeCI) Workflows(context.Context, string, string) (map[string][]byte, error) {
	return f.files, f.err
}

func (f fakeCI) File(_ context.Context, _, _, p string) ([]byte, error) {
	if p == "package.json" && f.pkg != nil {
		return f.pkg, nil
	}
	return nil, errors.New("404")
}

func (f fakeCI) LatestRuns(context.Context, string, string) ([]ci.JobRun, error) {
	return f.runs, f.runsErr
}

func TestReviewCI(t *testing.T) {
	tg := config.Target{Repo: "o/r", Branch: "main", TestCommand: "npm test", Toolchain: "node24"}
	got := reviewCI(context.Background(), fakeCI{files: map[string][]byte{}}, tg, "")
	if len(got) != 1 || got[0].Status != ci.Info || !strings.Contains(got[0].Detail, "no CI found") {
		t.Fatalf("no CI: %+v", got)
	}
	got = reviewCI(context.Background(), fakeCI{err: errors.New("boom")}, tg, "")
	if len(got) != 1 || !strings.Contains(got[0].Detail, "couldn't read the CI configuration: boom") {
		t.Fatalf("read error: %+v", got)
	}
	wf := []byte("name: CI\non: [pull_request]\njobs:\n  t:\n    name: Test\n    steps:\n      - run: npm run lint\n      - run: npm test\n")
	got = reviewCI(context.Background(), fakeCI{
		files:   map[string][]byte{".github/workflows/ci.yml": wf, ".github/workflows/bad.yml": []byte("on: [push\n")},
		pkg:     []byte(`{"scripts":{"test":"jest","lint":"eslint ."}}`),
		runsErr: errors.New("403"),
	}, tg, "")
	var text []string
	for _, f := range got {
		text = append(text, f.Status+" "+f.Check+": "+f.Detail)
	}
	all := strings.Join(text, "\n")
	for _, want := range []string{"INFO workflows: couldn't parse .github/workflows/bad.yml",
		"INFO default branch: couldn't read CI results: 403",
		"PASS test command: CI: `npm test` → `jest` (CI › Test)",
		"WARN checks grading doesn't run: lint: `npm run lint` → `eslint .` (CI › Test)"} {
		if !strings.Contains(all, want) {
			t.Errorf("lacks %q:\n%s", want, all)
		}
	}
}
