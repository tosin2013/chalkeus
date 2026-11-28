package readiness

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/proof"
)

// 041 FR-014: HIDDEN_TESTS_UNVERIFIED unless the hidden tests are proven, or the owner recorded
// reference: none.
func TestHiddenTestsUnverified(t *testing.T) {
	refuse := func(name string, w *world, want string) {
		t.Helper()
		res, _ := w.check(t, "002", 50)
		if res.Decision != "refuse" || res.Reasons[0] != "HIDDEN_TESTS_UNVERIFIED" {
			t.Fatalf("%s: %s", name, codes(res))
		}
		d := ruleDetail(res, "HIDDEN_TESTS_UNVERIFIED")
		if !strings.Contains(d, want) || !strings.Contains(d, "references/002.patch") || !strings.Contains(d, "author prove --target t --item 002") ||
			!strings.Contains(d, "proofs/002.json") {
			t.Fatalf("%s: %s", name, d)
		}
	}
	w := newWorld()
	w.noProof = true
	refuse("missing", w, "002: missing")

	stale := newWorld()
	stale.companion = withProofs(t, stale.companion, "")
	stale.companion["items/002/src/test/java/AcceptanceTest.java"] = "class AcceptanceTest { /* edited after the proof */ }\n"
	stale.noProof = true
	refuse("stale", stale, "002: stale")

	failed := newWorld()
	failed.noProof = true
	failed.companion["proofs/002.json"] = string((&proof.Proof{Version: 1, Item: "002", Reference: "patch", TestsTreeSHA256: "x",
		Result: proof.Failed, Code: "DRAFT_FAILS_ON_REFERENCE"}).Marshal())
	refuse("failed", failed, "failed (DRAFT_FAILS_ON_REFERENCE)")

	bad := newWorld()
	bad.noProof = true
	bad.companion["proofs/002.json"] = "{not json"
	refuse("invalid", bad, "002: invalid")

	none := newWorld()
	none.noProof = true
	none.companion["proofs/002.json"] = `{"version": 1, "item": "002", "reference": "none", "result": "none", "decided_by": "owner"}`
	res, _ := none.check(t, "002", 50)
	if res.Decision != "allow" || !res.ProofNone || !strings.Contains(ruleDetail(res, "hidden_tests_proof"), "WARN the owner recorded reference: none for 002") {
		t.Fatalf("reference: none allows with a warning: %s %+v", codes(res), res.Rules)
	}

	res, _ = newWorld().check(t, "002", 50)
	if res.Decision != "allow" || res.ProofNone || ruleDetail(res, "hidden_tests_proof") != "proven against the reference change" {
		t.Fatalf("%s", codes(res))
	}
}

func TestHiddenTestsUnverifiedPerSlice(t *testing.T) {
	// A whole-item run of a sliced item needs every slice's proof.
	w := slicedWorld(0, 0, 0)
	w.companion = withProofs(t, w.companion, "")
	delete(w.companion, "proofs/002/US2.json")
	w.noProof = true
	res, _ := w.check(t, "002", 50)
	if res.Decision != "refuse" || !strings.Contains(ruleDetail(res, "HIDDEN_TESTS_UNVERIFIED"), "002/US2: missing") ||
		strings.Contains(ruleDetail(res, "HIDDEN_TESTS_UNVERIFIED"), "002/US1") {
		t.Fatalf("%s %s", codes(res), ruleDetail(res, "HIDDEN_TESTS_UNVERIFIED"))
	}
	// The US1 slice run needs only US1's.
	w.slice = "US1"
	if res, _ := w.check(t, "002", 50); res.Decision != "allow" {
		t.Fatalf("%s", codes(res))
	}
}

func TestSkipHiddenSkipsTheProof(t *testing.T) {
	w := newWorld()
	w.noProof, w.skipHidden = true, true
	if res, _ := w.check(t, "002", 50); res.Decision != "allow" {
		t.Fatalf("the test author's readiness never needs a proof: %s", codes(res))
	}
}

func ruleDetail(res Result, id string) string {
	for _, r := range res.Rules {
		if r.ID == id {
			return r.Detail
		}
	}
	return ""
}
