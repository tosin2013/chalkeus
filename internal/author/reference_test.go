package author

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/proof"
	"github.com/tosin2013/chalkeus/internal/readiness"
)

const refMarker = "REFERENCE_PATCH_BODY_42"

// refReport is a grader reference-mode report.
func refReport(failure string, verdict string) map[string]any {
	r := map[string]any{"mode": "reference", "result": "pass", "failure": nil, "repeats": float64(3),
		"files": []any{map[string]any{"path": "tests/test_bundled.py", "base": "fail",
			"reference": []any{"pass", "pass", "pass"}, "verdict": verdict}}}
	if failure != "" {
		r["result"], r["failure"] = "fail", failure
	}
	return r
}

// referenceWorld is newWorld with a reference check wired in: the companion has a reference
// (unless none) and the grader returns report.
func referenceWorld(t *testing.T, report map[string]any, none bool) (*world, *int) {
	w := newWorld(t, config.LevelMinimal)
	calls := 0
	w.env.LoadReference = func(tg config.Target, item, slice string) (ReferenceFile, error) {
		rf := ReferenceFile{Path: tg.CompanionReferencePath(item, slice), Commit: "c0ffee"}
		if !none {
			rf.Content = []byte("diff --git a/app.py b/app.py\n" + refMarker + "\n")
		}
		return rf, nil
	}
	w.env.Reference = func(dir, id, item string, res readiness.Result, overlay, ref []byte) (map[string]any, []byte, error) {
		calls++
		if !bytes.Contains(ref, []byte(refMarker)) || !bytes.Contains(overlay, []byte(marker)) {
			t.Fatal("the reference check got the wrong inputs")
		}
		return report, []byte("on the reference: " + marker), nil
	}
	return w, &calls
}

func TestReferenceCheckProvesAndSubmitsTheProof(t *testing.T) {
	w, calls := referenceWorld(t, refReport("", "proven"), false)
	d, err := draftAndCheck(t, w, "pass")
	if err != nil || *calls != 1 {
		t.Fatalf("%v calls=%d", err, *calls)
	}
	rr := d.Checks[len(d.Checks)-1].Reference
	if rr == nil || rr.Result != "proven" || rr.ReferencePath != "proj/references/006.patch" || len(rr.Files) != 1 {
		t.Fatalf("%+v", rr)
	}
	if !strings.Contains(w.out.String(), "pass with the reference change") || !strings.Contains(w.out.String(), "1 of 1 files, 3 runs") {
		t.Fatal(w.out.String())
	}
	if _, err := Submit(context.Background(), w.env, "006"); err != nil {
		t.Fatal(err)
	}
	var proofBlob string
	for i, e := range w.fake.tree {
		if e["path"] == "proj/proofs/006.json" {
			proofBlob = w.fake.blobs[i]
		}
	}
	p, err := proof.Parse([]byte(proofBlob))
	if err != nil {
		t.Fatalf("proof: %v\n%s", err, proofBlob)
	}
	want, _ := proof.TreeSHA256(filepath.Join(d.Workspace, "draft"))
	if p.Result != proof.Proven || p.TestsTreeSHA256 != want || p.Reference != "patch" || p.Repeats != 3 || p.Files[0].Verdict != "proven" {
		t.Fatalf("%+v", p)
	}
	if !strings.Contains(w.fake.prBody, "Reference check: fails on the base · passes on the reference (1 of 1 files, 3 runs)") {
		t.Fatal(w.fake.prBody)
	}
	// Neither the reference nor the tests' output reaches the proof, the PR body or the draft's evidence.
	for _, s := range []string{proofBlob, w.fake.prBody, string(mustJSON(d))} {
		if strings.Contains(s, refMarker) || strings.Contains(s, marker) {
			t.Fatal("reference or test contents leaked")
		}
	}
}

func TestReferenceCheckFailureCodes(t *testing.T) {
	for failure, want := range map[string]string{
		"DRAFT_FAILS_ON_REFERENCE": CodeDraftFailsOnReference, "DRAFT_PASSES_ON_BASE": CodeDraftPassesOnBase,
		"DRAFT_FLAKY": CodeDraftFlaky, "REFERENCE_BROKEN": CodeReferenceBroken, "REFERENCE_DOESNT_APPLY": CodeReferenceDoesntApply,
		"something_else": CodeDraftGraderError} {
		t.Run(failure, func(t *testing.T) {
			w, _ := referenceWorld(t, refReport(failure, "fails_on_reference"), false)
			d, err := draftAndCheck(t, w, "pass")
			if code(err) != want || d.State != "checked(fail)" {
				t.Fatalf("%v %s", err, d.State)
			}
			if _, err := Submit(context.Background(), w.env, "006"); code(err) != CodeDraftNotChecked {
				t.Fatalf("a draft that failed the reference check can't be submitted: %v", err)
			}
		})
	}
}

func TestNoReferenceSkipsTheCheckAndSaysSo(t *testing.T) {
	w, calls := referenceWorld(t, nil, true)
	if _, err := draftAndCheck(t, w, "pass"); err != nil || *calls != 0 {
		t.Fatalf("%v calls=%d", err, *calls)
	}
	if !strings.Contains(w.out.String(), "reference check was skipped") {
		t.Fatal(w.out.String())
	}
	if _, err := Submit(context.Background(), w.env, "006"); err != nil {
		t.Fatal(err)
	}
	for _, e := range w.fake.tree {
		if strings.Contains(e["path"].(string), "proofs/") {
			t.Fatal("no proof without a reference check")
		}
	}
	if !strings.Contains(w.fake.prBody, "Reference check: not run") {
		t.Fatal(w.fake.prBody)
	}
}

func TestAFailedBaselineSkipsTheReferenceCheck(t *testing.T) {
	w, calls := referenceWorld(t, refReport("", "proven"), false)
	if _, err := draftAndCheck(t, w, "hidden_passes"); code(err) != CodeDraftHiddenPasses || *calls != 0 {
		t.Fatalf("%v calls=%d", err, *calls)
	}
}

func TestReferenceSummary(t *testing.T) {
	rr, code := ReferenceOutcome(map[string]any{"mode": "reference", "result": "fail", "failure": "DRAFT_FLAKY", "repeats": float64(3),
		"files": []any{map[string]any{"path": "a", "verdict": "proven"}, map[string]any{"path": "b", "verdict": "flaky"}}}, ReferenceFile{Path: "p"})
	if code != CodeDraftFlaky || rr.Summary() != "FAILED DRAFT_FLAKY (1 of 2 files proven, 3 runs): b: flaky" {
		t.Fatalf("%s %q", code, rr.Summary())
	}
	if (*ReferenceResult)(nil).Summary() != "not run (no reference change)" {
		t.Fatal("nil summary")
	}
	// A baseline-shaped report is never a proof.
	if _, code := ReferenceOutcome(map[string]any{"mode": "baseline", "result": "pass"}, ReferenceFile{}); code != CodeDraftGraderError {
		t.Fatal(code)
	}
}

func TestManifestChangeReachesTheCheck(t *testing.T) {
	w := newWorld(t, config.LevelMinimal)
	w.res.ManifestChange = &readiness.ManifestChange{Commit: "abc123", Files: []string{"package.json"}}
	d, err := Start(context.Background(), w.env, "006", false)
	if err != nil {
		t.Fatal(err)
	}
	d2, _ := LoadDraft(Root(w.env.EvidenceRoot), "", "proj", "006")
	if mc := manifestOf(d2.Readiness["manifest_change"]); mc == nil || mc.Commit != "abc123" || d.ID != d2.ID {
		t.Fatalf("the manifest change must survive the draft record: %+v", d2.Readiness)
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
