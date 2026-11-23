package doctor

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/readiness"
)

func TestReportReadinessAndPrint(t *testing.T) {
	var r Report
	r.OK("profile", "loads", nil, "level=minimal")
	r.Readiness(readiness.Result{HiddenFiles: 0, Rules: []readiness.Rule{
		{ID: "spec", Result: "pass", Detail: "specs/006/spec.md"},
		{ID: "ITEM_NO_HIDDEN_TESTS", Result: "fail", Detail: "no files"},
		{ID: "INPUT_TOO_LARGE", Result: "fail", Detail: "snapshot is 61 MB, over PILOT_MAX_INPUT_MB=50"},
	}, Mapping: map[string]any{"present": false}})
	r.Add("github", "branch protection", Warn, "main is not protected")
	if !r.Failed() {
		t.Fatal("two failed rules must fail the report")
	}
	var b bytes.Buffer
	r.Print(&b)
	out := b.String()
	for _, want := range []string{"== profile", "== item", "FAIL INPUT_TOO_LARGE: snapshot is 61 MB", "== github", "doctor: 2 pass, 1 warn, 2 fail, 0 skipped"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	var ok Report
	ok.OK("x", "y", nil, "")
	ok.Add("x", "z", Warn, "w")
	if ok.Failed() {
		t.Fatal("warnings don't fail")
	}
	var bad Report
	if bad.OK("x", "y", errors.New("boom"), "") || !bad.Failed() {
		t.Fatal("an error is a failure")
	}
}

func TestCoverageRows(t *testing.T) {
	for m, want := range map[string]string{
		"stale":     "Warn",
		"uncovered": "Warn",
		"covered":   "Pass",
	} {
		var r Report
		mapping := map[string]any{"present": true, "valid": true, "uncovered": []any{}}
		switch m {
		case "stale":
			mapping["stale"] = true
		case "uncovered":
			mapping["uncovered"] = []any{"US2-AS1"}
		}
		r.Readiness(readiness.Result{HiddenFiles: 2, Mapping: mapping})
		got := r.Rows[len(r.Rows)-1].Status
		if (want == "Warn" && got != Warn) || (want == "Pass" && got != Pass) {
			t.Errorf("%s: %s", m, got)
		}
	}
}

func TestDigestMatches(t *testing.T) {
	if !DigestMatches("ghcr.io/a/b@sha256:1", "ghcr.io/a/b@sha256:1") || DigestMatches("x@sha256:1", "x@sha256:2") || DigestMatches("x:tag", "x@sha256:1") {
		t.Fatal("digest comparison")
	}
}
