package debt

import (
	"context"
	"strings"
	"testing"
	"time"
)

// memReader is an in-memory Reader.
type memReader struct {
	prs       []PR
	checks    map[string][]Check // sha -> checks
	noChecks  map[string]bool    // sha -> ErrNoAccess
	files     map[string]string
	alerts    []Alert
	alertsErr error
}

func (m *memReader) DefaultBranch(context.Context, string) (string, string, error) {
	return "main", "MAIN", nil
}
func (m *memReader) OpenPRs(context.Context, string) ([]PR, error) { return m.prs, nil }
func (m *memReader) Checks(_ context.Context, _, sha string) ([]Check, error) {
	if m.noChecks[sha] {
		return nil, ErrNoAccess
	}
	return m.checks[sha], nil
}
func (m *memReader) File(_ context.Context, _, p, _ string) ([]byte, error) {
	if s, ok := m.files[p]; ok {
		return []byte(s), nil
	}
	return nil, nil
}
func (m *memReader) Alerts(context.Context, string) ([]Alert, error) { return m.alerts, m.alertsErr }

func dep(n int, title, branch string) PR {
	return PR{Number: n, Title: title, Author: DependabotLogin, AuthorType: "Bot", HeadRef: branch, HeadSHA: "S" + string(rune('A'+n))}
}

func build(t *testing.T, m *memReader) (*Report, map[int]Entry) {
	t.Helper()
	rep, err := Build(context.Background(), m, Options{Target: "t", Repo: "o/r", Credential: "operator-read", Now: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	by := map[int]Entry{}
	for _, g := range rep.Groups {
		for _, e := range g.PRs {
			by[e.Number] = e
		}
	}
	for _, e := range rep.NotTriaged {
		by[e.Number] = e
	}
	return rep, by
}

func pass(names ...string) []Check {
	var out []Check
	for _, n := range names {
		out = append(out, Check{Name: n, State: CheckPassing})
	}
	return out
}

func TestClassifyRules(t *testing.T) {
	m := &memReader{
		files: map[string]string{"package-lock.json": `{"lockfileVersion":3,"packages":{"node_modules/old":{"version":"2.0.0"}}}`},
		checks: map[string][]Check{
			"MAIN": pass("test", "lint"),
		},
		alerts: []Alert{{GHSA: "GHSA-aaaa", Package: "patchy", Patched: "1.0.1"}},
	}
	add := func(p PR, checks []Check) {
		m.prs = append(m.prs, p)
		m.checks[p.HeadSHA] = checks
	}
	add(dep(1, "Bump old from 1.0.0 to 1.5.0", "dependabot/npm_and_yarn/old-1.5.0"), pass("test"))                                       // main has 2.0.0
	add(dep(2, "Bump patchy from 1.0.0 to 1.0.1", "dependabot/npm_and_yarn/patchy-1.0.1"), pass("test", "lint"))                         // safe, fixes an alert
	add(dep(3, "Bump big from 1.0.0 to 2.0.0", "dependabot/npm_and_yarn/big-2.0.0"), pass("test"))                                       // major
	add(dep(4, "Bump broken from 1.0.0 to 1.1.0", "dependabot/npm_and_yarn/broken-1.1.0"), []Check{{Name: "test", State: CheckFailing}}) // own failure
	add(dep(5, "Bump slow from 1.0.0 to 1.0.1", "dependabot/npm_and_yarn/slow-1.0.1"), []Check{{Name: "test", State: CheckPending}})
	add(dep(6, "Bump dup from 1.0.0 to 1.1.0", "dependabot/npm_and_yarn/dup-1.1.0"), pass("test")) // superseded by #7
	add(dep(7, "Bump dup from 1.0.0 to 1.2.0", "dependabot/npm_and_yarn/dup-1.2.0"), pass("test"))
	add(dep(8, "Bump actions/checkout from 4 to 5", "dependabot/github_actions/actions/checkout-5"), nil)
	add(PR{Number: 9, Title: "Update all", Author: "renovate[bot]", AuthorType: "Bot", HeadRef: "renovate/all", HeadSHA: "R"}, nil)
	add(PR{Number: 10, Title: "A person's change", Author: "someone", AuthorType: "User", HeadRef: "feature", HeadSHA: "P"}, nil)
	p11 := dep(11, "Bump hidden from 1.0.0 to 1.0.1", "dependabot/npm_and_yarn/hidden-1.0.1")
	m.prs = append(m.prs, p11)
	m.noChecks = map[string]bool{p11.HeadSHA: true}

	rep, by := build(t, m)
	for n, want := range map[int]string{1: ClassSuperseded, 2: ClassSafe, 3: ClassNeedsCode, 4: ClassNeedsCode, 5: ClassPending,
		6: ClassSuperseded, 7: ClassSafe, 8: ClassNotTriaged, 9: ClassNotTriaged, 11: ClassNotTriaged} {
		if by[n].Class != want {
			t.Errorf("#%d: %s (%s), want %s", n, by[n].Class, by[n].Reason, want)
		}
	}
	if _, ok := by[10]; ok {
		t.Error("a person's PR was triaged")
	}
	if a, _ := by[2].Advisories.([]string); len(a) != 1 || a[0] != "GHSA-aaaa" {
		t.Errorf("advisories %v", by[2].Advisories)
	}
	if !by[3].DependencyItem || !by[4].DependencyItem || by[2].DependencyItem {
		t.Error("dependency_item")
	}
	if !strings.Contains(by[1].Reason, "old 2.0.0") || !strings.Contains(by[6].Reason, "#7") {
		t.Errorf("reasons %q %q", by[1].Reason, by[6].Reason)
	}
	if !strings.Contains(strings.Join(rep.Unreadable, ";"), "PR checks") {
		t.Errorf("unreadable %v", rep.Unreadable)
	}
	if rep.Counts[ClassNotTriaged] != 3 || rep.Counts[ClassSafe] != 2 {
		t.Errorf("counts %v", rep.Counts)
	}
}

// A failing check never makes a PR safe unless it's shared, and unreadable checks never do.
func TestNeverFailsOpenToSafe(t *testing.T) {
	m := &memReader{checks: map[string][]Check{"MAIN": pass("test")}, alertsErr: ErrNoAccess}
	m.prs = []PR{dep(1, "Bump a from 1.0.0 to 1.0.1", "dependabot/npm_and_yarn/a-1.0.1")}
	m.noChecks = map[string]bool{m.prs[0].HeadSHA: true}
	_, by := build(t, m)
	if by[1].Class == ClassSafe || by[1].Advisories != Unknown {
		t.Fatalf("%+v", by[1])
	}
}

// A PR-only check failing on most PRs is shared; failing on a few, it's the PR's own.
func TestSharedPROnlyChecks(t *testing.T) {
	m := &memReader{checks: map[string][]Check{"MAIN": pass("test")}}
	for i := 1; i <= 4; i++ {
		p := dep(i, "Bump p"+string(rune('a'+i))+" from 1.0.0 to 1.0.1", "dependabot/npm_and_yarn/x")
		m.prs = append(m.prs, p)
		cs := []Check{{Name: "auto-merge", State: CheckFailing}}
		if i == 4 {
			cs = append(cs, Check{Name: "node20", State: CheckFailing})
		}
		m.checks[p.HeadSHA] = cs
	}
	rep, by := build(t, m)
	if rep.SharedChecks["auto-merge"] == "" || rep.SharedChecks["node20"] != "" {
		t.Fatal(rep.SharedChecks)
	}
	if by[1].Class != ClassSafe || by[4].Class != ClassNeedsCode {
		t.Fatalf("%s %s", by[1].Class, by[4].Class)
	}
}

func TestUnreadableNamesTheAppSetting(t *testing.T) {
	m := &memReader{checks: map[string][]Check{}, noChecks: map[string]bool{"MAIN": true}, alertsErr: ErrNoAccess}
	rep, err := Build(context.Background(), m, Options{Repo: "o/r", Credential: "app-read", Now: time.Now(),
		Permissions: map[string]string{"checks": "granted", "statuses": "missing", "vulnerability_alerts": "missing"}})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(rep.Unreadable, "; ")
	for _, want := range []string{"default branch's checks: grant the GitHub App Commit statuses: read", "advisories: grant the GitHub App Dependabot alerts: read"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}
