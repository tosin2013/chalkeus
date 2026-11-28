package author

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/readiness"
)

// Reference runs the grader's reference mode (041, pipeline.AuthorReference): the report (per-file
// verdicts, no output) and the hidden tests' output for the terminal only.
type Reference func(dir, id, item string, res readiness.Result, overlay, reference []byte) (map[string]any, []byte, error)

// ReferenceFile is an item's reference change read from the companion.
type ReferenceFile struct {
	Path    string // in the companion, e.g. documcp/references/004.patch
	Commit  string // the companion commit it was read at
	Content []byte // nil when the companion has none
}

// SHA256 is the reference's content hash.
func (r ReferenceFile) SHA256() string {
	h := sha256.Sum256(r.Content)
	return hex.EncodeToString(h[:])
}

// LoadReference reads an item's reference change from a fresh clone of the private companion
// (research R4). It is read into memory and staged only to the grader, never written to a
// workspace the test author or a worker can read.
func LoadReference(t config.Target, item, slice string) (ReferenceFile, error) {
	rf := ReferenceFile{Path: t.CompanionReferencePath(item, slice)}
	dir, commit, err := fixture.FreshClone(t.Host(), t.CompanionRepo, "", 1)
	if err != nil {
		return rf, fmt.Errorf("companion repository %s: %v", t.CompanionRepo, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	rf.Commit = commit
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rf.Path)))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return rf, nil
	case err != nil:
		return rf, err
	}
	rf.Content = b
	return rf, nil
}

// referenceCodes maps the grader's reference failure to the draft code.
var referenceCodes = map[string]string{
	"DRAFT_FAILS_ON_REFERENCE": CodeDraftFailsOnReference, "DRAFT_PASSES_ON_BASE": CodeDraftPassesOnBase,
	"DRAFT_FLAKY": CodeDraftFlaky, "REFERENCE_BROKEN": CodeReferenceBroken, "REFERENCE_DOESNT_APPLY": CodeReferenceDoesntApply,
}

// referenceDetail is what each code tells the author or owner.
var referenceDetail = map[string]string{
	CodeDraftFailsOnReference: "a hidden test file did not pass with the reference change applied; it may not compile or load, or it asserts something the spec doesn't ask for",
	CodeDraftPassesOnBase:     "a hidden test file already passes on the unchanged base, so it can't tell a correct change from no change",
	CodeDraftFlaky:            "a hidden test file passed and failed across runs on the reference change; make it deterministic",
	CodeReferenceBroken:       "the project's own tests fail with the reference change applied; the owner fixes the reference",
	CodeReferenceDoesntApply:  "the reference patch doesn't apply to the base; the owner updates the reference",
	CodeDraftGraderError:      "the grader could not finish the reference check; see its grade.json and try again",
}

// ReferenceOutcome turns a reference-mode report into a ReferenceResult and its code (DRAFT_OK
// when every file is proven).
func ReferenceOutcome(report map[string]any, rf ReferenceFile) (*ReferenceResult, string) {
	rr := &ReferenceResult{Result: "failed", ReferencePath: rf.Path, ReferenceSHA256: rf.SHA256(), CompanionCommit: rf.Commit}
	if n, ok := report["repeats"].(float64); ok {
		rr.Repeats = int(n)
	}
	files, _ := report["files"].([]any)
	for _, x := range files {
		f, _ := x.(map[string]any)
		fv := FileVerdict{Path: str(f["path"]), Base: str(f["base"]), Reference: strs(f["reference"]), Verdict: str(f["verdict"])}
		rr.Files = append(rr.Files, fv)
	}
	code := CodeDraftOK
	if report["result"] == "pass" && report["mode"] == "reference" {
		rr.Result = "proven"
	} else if c := referenceCodes[str(report["failure"])]; c != "" {
		code = c
	} else {
		code = CodeDraftGraderError
	}
	rr.Code = code
	if code == CodeDraftOK {
		rr.Code = ""
	}
	return rr, code
}

// Summary is the one-line result for the PR body and the terminal.
func (rr *ReferenceResult) Summary() string {
	if rr == nil {
		return "not run (no reference change)"
	}
	proven := 0
	var bad []string
	for _, f := range rr.Files {
		if f.Verdict == "proven" {
			proven++
		} else {
			bad = append(bad, f.Path+": "+f.Verdict)
		}
	}
	s := fmt.Sprintf("fails on the base · passes on the reference (%d of %d files, %d runs)", proven, len(rr.Files), rr.Repeats)
	if rr.Result != "proven" {
		s = fmt.Sprintf("FAILED %s (%d of %d files proven, %d runs)", rr.Code, proven, len(rr.Files), rr.Repeats)
		if len(bad) > 0 {
			s += ": " + strings.Join(bad, "; ")
		}
	}
	return s
}

// referenceCheck runs the reference check for a draft whose baseline passed. It returns the result
// (nil when the companion has no reference) and the code (DRAFT_OK when proven or skipped).
func referenceCheck(e Env, d *Draft, item, dir, id string, res readiness.Result, tarb []byte) (*ReferenceResult, string, string, error) {
	rf, err := e.LoadReference(e.Target, item, d.Slice)
	if err != nil {
		return nil, "", "", err
	}
	if rf.Content == nil {
		fmt.Fprintf(e.Out, "no reference change at %s:%s, so the reference check was skipped; the hidden tests will need one (or the owner's reference: none) before a run (HIDDEN_TESTS_UNVERIFIED)\n", e.Target.CompanionRepo, rf.Path)
		return nil, CodeDraftOK, "", nil
	}
	fmt.Fprintf(e.Out, "reference check: each hidden file on the base once and on the reference change %d times; this takes a while...\n", e.Target.Repeats())
	report, hidden, err := e.Reference(dir, id, item, res, tarb, rf.Content)
	if err != nil {
		return nil, "", "", err
	}
	rr, code := ReferenceOutcome(report, rf)
	rr.GradePath = filepath.Join(filepath.Base(dir), "reference", "grade.json")
	return rr, code, string(hidden), nil
}
