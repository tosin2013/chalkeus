package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/doctor"
	"github.com/tosin2013/chalkeus/internal/scm"
)

// stubHost implements the scm.Client methods doctorHostRows uses; the rest panic if called.
type stubHost struct {
	scm.Client
	perms map[string]scm.Permission
	fail  error
}

func (s stubHost) Host() scm.Host { return scm.NewHost(scm.Bitbucket, "", "") }
func (s stubHost) ResolveAccount(_ context.Context, given string) (scm.Account, error) {
	return scm.Account{ID: given, Name: "n" + given}, nil
}
func (s stubHost) Permission(_ context.Context, _ string, a scm.Account) (scm.Permission, error) {
	if s.fail != nil {
		return scm.PermNone, s.fail
	}
	return s.perms[a.ID], nil
}

func rowsText(r doctor.Report) string {
	var b strings.Builder
	for _, row := range r.Rows {
		b.WriteString(row.Status + " " + row.Check + " " + row.Detail + "\n")
	}
	return b.String()
}

func TestDoctorHostRows(t *testing.T) {
	tg := config.Target{Repo: "ws/r", PublisherEnv: t.TempDir() + "/missing.env", Approvers: []string{"{a}", "{b}"}}
	var r doctor.Report
	doctorHostRows(context.Background(), &r, stubHost{perms: map[string]scm.Permission{"{a}": scm.PermWrite, "{b}": scm.PermRead}}, tg, nil)
	got := rowsText(r)
	for _, want := range []string{"FAIL publisher credential", "PASS approver {a}", "FAIL approver {b} n{b} has read access"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	r = doctor.Report{}
	doctorHostRows(context.Background(), &r, stubHost{fail: errors.New("can't read who may write ws/r: the operator credential needs repository admin")}, tg, nil)
	if got := rowsText(r); !strings.Contains(got, "FAIL approver {a} can't verify approvers' write access") || !strings.Contains(got, "repository admin") {
		t.Errorf("%s", got)
	}
	r = doctor.Report{}
	doctorHostRows(context.Background(), &r, stubHost{}, config.Target{Repo: "ws/r", PublisherEnv: tg.PublisherEnv}, nil)
	if got := rowsText(r); !strings.Contains(got, "FAIL approvers no FORGE_APPROVERS") {
		t.Errorf("%s", got)
	}
}
