package debt

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

// Limit is how many bot PRs one report triages (research R2).
const Limit = 100

// lockFile is the file each toolchain's resolved versions are read from (research R5).
var lockFile = map[string]string{
	"node24":      "package-lock.json",
	"python3.13":  "requirements.txt",
	"go":          "go.mod",
	"jdk21-maven": "pom.xml",
}

// Options are the report's fixed inputs.
type Options struct {
	Target      string
	Repo        string
	Credential  string            // "app-read" or "operator-read"
	Permissions map[string]string // optional permission -> granted|missing (App mode)
	Now         time.Time
}

// pr is one Dependabot PR on its way to an Entry.
type pr struct {
	PR
	bump      Bump
	parsed    bool
	checks    []Check
	checksErr error
}

// Build reads everything through r and classifies it. Only the default branch's own reads are
// fatal; anything else unreadable is recorded and shown as unknown.
func Build(ctx context.Context, r Reader, o Options) (*Report, error) {
	rep := &Report{Version: 1, Target: o.Target, Repo: o.Repo, Credential: o.Credential, Permissions: o.Permissions,
		GeneratedAt: o.Now.UTC().Format(time.RFC3339), Groups: []Group{}, NotTriaged: []Entry{}, BaselineFlags: []Flag{},
		Unreadable: []string{}, DefaultChecksFailing: []string{}, SharedChecks: map[string]string{}, Counts: map[string]int{}}
	branch, sha, err := r.DefaultBranch(ctx, o.Repo)
	if err != nil {
		return nil, err
	}
	rep.DefaultBranch, rep.DefaultCommit = branch, sha
	defaultFailing, defaultRuns := map[string]bool{}, map[string]bool{}
	dchecks, err := r.Checks(ctx, o.Repo, sha)
	if err != nil {
		rep.Unreadable = append(rep.Unreadable, "the default branch's checks: "+whyPerm(err, o.Permissions, "checks", "statuses"))
	}
	for _, c := range dchecks {
		defaultRuns[c.Name] = true
		if c.State == CheckFailing && !defaultFailing[c.Name] {
			defaultFailing[c.Name] = true
			rep.DefaultChecksFailing = append(rep.DefaultChecksFailing, c.Name)
		}
	}
	sort.Strings(rep.DefaultChecksFailing)
	open, err := r.OpenPRs(ctx, o.Repo)
	if err != nil {
		return nil, err
	}
	var prs []*pr
	for _, p := range open {
		switch {
		case IsDependabot(p):
		case p.AuthorType == "Bot":
			rep.NotTriaged = append(rep.NotTriaged, Entry{Number: p.Number, URL: p.URL, Author: p.Author, Packages: []Package{},
				Level: LevelUnknown, Advisories: Unknown, Class: ClassNotTriaged, Reason: p.Author + ": only Dependabot PRs are triaged"})
			continue
		default:
			continue // a person's PR: not the report's business
		}
		if len(prs) >= Limit {
			rep.NotTriaged = append(rep.NotTriaged, Entry{Number: p.Number, URL: p.URL, Author: p.Author, Packages: []Package{},
				Level: LevelUnknown, Advisories: Unknown, Class: ClassNotTriaged, Reason: fmt.Sprintf("over the limit of %d PRs per report", Limit)})
			continue
		}
		x := &pr{PR: p}
		x.bump, x.parsed = ParseBump(p)
		if x.parsed && x.bump.Toolchain != "" {
			x.checks, x.checksErr = r.Checks(ctx, o.Repo, p.HeadSHA)
		}
		prs = append(prs, x)
	}
	files := fileCache{r: r, repo: o.Repo, ref: sha, got: map[string][]byte{}}
	alerts, alertsErr := r.Alerts(ctx, o.Repo)
	if alertsErr != nil {
		rep.Unreadable = append(rep.Unreadable, "advisories: "+whyPerm(alertsErr, o.Permissions, "vulnerability_alerts"))
	}
	shared := sharedChecks(prs, defaultFailing, defaultRuns)
	rep.SharedChecks = shared
	checksUnreadable := false
	var entries []Entry
	for _, x := range prs {
		e := classify(ctx, x, prs, shared, &files, alerts, alertsErr)
		if x.checksErr != nil {
			checksUnreadable = true
		}
		if e.Class == ClassNotTriaged {
			rep.NotTriaged = append(rep.NotTriaged, e)
		} else {
			entries = append(entries, e)
		}
	}
	if checksUnreadable {
		rep.Unreadable = append(rep.Unreadable, "PR checks: "+whyPerm(ErrNoAccess, o.Permissions, "checks", "statuses"))
	}
	rep.Groups = group(entries, shared)
	for _, g := range rep.Groups {
		for _, e := range g.PRs {
			rep.Counts[e.Class]++
		}
	}
	rep.Counts[ClassNotTriaged] = len(rep.NotTriaged)
	sort.Slice(rep.NotTriaged, func(i, j int) bool { return rep.NotTriaged[i].Number < rep.NotTriaged[j].Number })
	rep.BaselineFlags = append(rep.BaselineFlags, flags(ctx, rep, &files)...)
	return rep, nil
}

func why(err error) string {
	if errors.Is(err, ErrNoAccess) {
		return "not readable with this credential"
	}
	return err.Error()
}

// AppSetting names each optional permission as the GitHub App's settings page shows it.
var AppSetting = map[string]string{"checks": "Checks", "statuses": "Commit statuses", "vulnerability_alerts": "Dependabot alerts"}

// whyPerm is why, naming the App settings to grant when the App lacks the permissions.
func whyPerm(err error, perms map[string]string, need ...string) string {
	var missing []string
	for _, p := range need {
		if perms[p] == "missing" {
			missing = append(missing, AppSetting[p]+": read")
		}
	}
	if len(missing) > 0 && errors.Is(err, ErrNoAccess) {
		return "grant the GitHub App " + strings.Join(missing, " and ")
	}
	return why(err)
}

// fileCache reads default-branch files once.
type fileCache struct {
	r    Reader
	repo string
	ref  string
	got  map[string][]byte
	errs map[string]error
}

func (f *fileCache) get(ctx context.Context, p string) ([]byte, error) {
	if b, ok := f.got[p]; ok {
		return b, f.errs[p]
	}
	b, err := f.r.File(ctx, f.repo, p, f.ref)
	f.got[p] = b
	if err != nil {
		if f.errs == nil {
			f.errs = map[string]error{}
		}
		f.errs[p] = err
	}
	return b, err
}

// classify applies research R4's rules in order.
func classify(ctx context.Context, x *pr, all []*pr, shared map[string]string, files *fileCache, alerts []Alert, alertsErr error) Entry {
	e := Entry{Number: x.Number, URL: x.URL, Author: x.Author, Ecosystem: x.bump.Ecosystem, Directory: x.bump.Directory,
		Packages: x.bump.Packages, Level: x.bump.level(), Advisories: Unknown}
	if e.Packages == nil {
		e.Packages = []Package{}
	}
	e.Checks = Checks{Failing: []string{}, Pending: []string{}}
	e.Checks.Failing, e.Checks.Pending, e.Checks.Passing = summarise(x.checks)
	if alertsErr == nil {
		e.Advisories = fixes(e.Packages, alerts)
	}
	// 1. not triaged
	switch {
	case !x.parsed:
		return notTriaged(e, "could not read the packages from the title or body")
	case x.bump.Toolchain == "":
		return notTriaged(e, fmt.Sprintf("%s: no catalog toolchain", orUnknown(x.bump.Ecosystem)))
	}
	// 2. superseded (before the checks: a superseded PR needs no CI to be closed)
	if why := superseded(ctx, x, all, files); why != "" {
		e.Class, e.Reason = ClassSuperseded, why
		return e
	}
	if x.checksErr != nil {
		return notTriaged(e, "its check results are not readable with this credential")
	}
	// 3. pending
	if len(e.Checks.Failing) == 0 && len(e.Checks.Pending) > 0 {
		e.Class, e.Reason = ClassPending, "waiting for "+strings.Join(e.Checks.Pending, ", ")
		return e
	}
	// 4. needs code changes
	var own []string
	for _, c := range e.Checks.Failing {
		if shared[c] == "" {
			own = append(own, c)
		}
	}
	if e.Level == LevelMajor || e.Level == LevelUnknown {
		e.Class, e.DependencyItem = ClassNeedsCode, true
		e.Reason = majorReason(e.Packages)
		if len(own) > 0 {
			e.Reason += "; also fails " + strings.Join(own, ", ")
		}
		return e
	}
	if len(own) > 0 {
		e.Class, e.DependencyItem = ClassNeedsCode, true
		e.Reason = "fails " + strings.Join(own, ", ") + ", which most other PRs and the default branch don't"
		return e
	}
	// 5. safe
	e.Class = ClassSafe
	if len(e.Checks.Failing) > 0 {
		e.Reason = "fails only shared checks (" + strings.Join(e.Checks.Failing, ", ") + "): fix those first"
	}
	return e
}

func notTriaged(e Entry, why string) Entry {
	e.Class, e.Reason, e.DependencyItem = ClassNotTriaged, why, false
	return e
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown ecosystem"
	}
	return s
}

func majorReason(pkgs []Package) string {
	var parts []string
	for _, p := range pkgs {
		if p.Level == LevelMajor || p.Level == LevelUnknown {
			parts = append(parts, fmt.Sprintf("%s %s → %s", p.Name, p.From, p.To))
		}
	}
	return "major bump: " + strings.Join(parts, ", ")
}

// superseded is R4 rule 2's reason, or "".
func superseded(ctx context.Context, x *pr, all []*pr, files *fileCache) string {
	// a newer open Dependabot PR bumps the same packages further, in the same directory
	for _, y := range all {
		if y == x || !y.parsed || y.bump.Directory != x.bump.Directory || y.bump.Ecosystem != x.bump.Ecosystem {
			continue
		}
		if covers(y.bump.Packages, x.bump.Packages) {
			return fmt.Sprintf("#%d bumps %s further", y.Number, x.bump.Packages[0].Name)
		}
	}
	lf := lockFile[x.bump.Toolchain]
	if lf == "" {
		return ""
	}
	p := strings.TrimPrefix(path.Join(x.bump.Directory, lf), "/")
	content, err := files.get(ctx, p)
	if err != nil || content == nil {
		return ""
	}
	var have []string
	for i := range x.bump.Packages {
		pk := &x.bump.Packages[i]
		dv := Resolved(p, content, pk.Name)
		if dv == "" {
			pk.DefaultBranchVersion = Unknown
			return ""
		}
		pk.DefaultBranchVersion = dv
		if c, ok := compareVersions(dv, pk.To); !ok || c < 0 {
			return ""
		}
		have = append(have, pk.Name+" "+dv)
	}
	return "the default branch already has " + strings.Join(have, ", ")
}

// covers reports whether newer bumps every package of older to a strictly newer version.
func covers(newer, older []Package) bool {
	if len(older) == 0 {
		return false
	}
	to := map[string]string{}
	for _, p := range newer {
		to[p.Name] = p.To
	}
	for _, p := range older {
		v, ok := to[p.Name]
		if !ok {
			return false
		}
		if c, ok := compareVersions(v, p.To); !ok || c <= 0 {
			return false
		}
	}
	return true
}

// fixes lists the advisories a PR's versions fix (research R6).
func fixes(pkgs []Package, alerts []Alert) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, a := range alerts {
		for _, p := range pkgs {
			if a.Package != p.Name || a.Patched == "" || seen[a.GHSA] {
				continue
			}
			if c, ok := compareVersions(p.To, a.Patched); ok && c >= 0 {
				out = append(out, a.GHSA)
				seen[a.GHSA] = true
			}
		}
	}
	sort.Strings(out)
	return out
}

// summarise splits a PR's checks; a name that appears more than once (several workflows with one
// job name) is failing when any run of it fails.
func summarise(checks []Check) (failing, pending []string, passing int) {
	state := map[string]string{}
	for _, c := range checks {
		switch {
		case c.State == CheckFailing:
			state[c.Name] = CheckFailing
		case c.State == CheckPending && state[c.Name] != CheckFailing:
			state[c.Name] = CheckPending
		case state[c.Name] == "":
			state[c.Name] = CheckPassing
		}
	}
	failing, pending = []string{}, []string{}
	for n, s := range state {
		switch s {
		case CheckFailing:
			failing = append(failing, n)
		case CheckPending:
			pending = append(pending, n)
		default:
			passing++
		}
	}
	sort.Strings(failing)
	sort.Strings(pending)
	return failing, pending, passing
}

// sharedChecks are the failing checks that say nothing about one PR (research R4): those that
// also fail on the default branch's head, and those that don't run there but fail on more than
// half of the PRs whose checks were read (at least 3), such as an auto-merge job blocked by a
// red required check. Each maps to why it's shared.
func sharedChecks(prs []*pr, defaultFailing, defaultRuns map[string]bool) map[string]string {
	out := map[string]string{}
	for n := range defaultFailing {
		out[n] = "also failing on the default branch"
	}
	count, read := map[string]int{}, 0
	for _, x := range prs {
		if !x.parsed || x.bump.Toolchain == "" || x.checksErr != nil {
			continue
		}
		read++
		failing, _, _ := summarise(x.checks)
		for _, n := range failing {
			count[n]++
		}
	}
	for n, c := range count {
		if !defaultRuns[n] && c >= 3 && 2*c > read {
			out[n] = fmt.Sprintf("fails on %d of %d PRs and doesn't run on the default branch", c, read)
		}
	}
	return out
}
