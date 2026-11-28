package config

import (
	"strings"
	"testing"
)

// 041 US3: the reference check's profile keys and companion paths.
func TestReferenceKeys(t *testing.T) {
	c := targetCfg(t)
	writeTarget(t, c, "greeting", goodTarget)
	tg, err := c.LoadTarget("greeting", "tosin2013/specfoundry")
	if err != nil {
		t.Fatal(err)
	}
	if tg.Repeats() != 3 || tg.HiddenTestCommand != "" {
		t.Fatalf("defaults: %d %q", tg.Repeats(), tg.HiddenTestCommand)
	}
	writeTarget(t, c, "greeting", goodTarget+"FORGE_REFERENCE_REPEATS=5\nFORGE_HIDDEN_TEST_COMMAND=npx jest {files}\n")
	if tg, err = c.LoadTarget("greeting", "tosin2013/specfoundry"); err != nil || tg.Repeats() != 5 || tg.HiddenTestCommand != "npx jest {files}" {
		t.Fatalf("%v %d %q", err, tg.Repeats(), tg.HiddenTestCommand)
	}
	for body, want := range map[string]string{
		"FORGE_REFERENCE_REPEATS=1\n":          "at least 2",
		"FORGE_REFERENCE_REPEATS=0\n":          "positive integer",
		"FORGE_HIDDEN_TEST_COMMAND=npx jest\n": "{files}",
	} {
		writeTarget(t, c, "greeting", goodTarget+body)
		if _, err := c.LoadTarget("greeting", "tosin2013/specfoundry"); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v", body, err)
		}
	}
}

func TestCompanionReferenceAndProofPaths(t *testing.T) {
	tg := Target{}
	if tg.CompanionReferencePath("004", "") != "references/004.patch" || tg.CompanionProofPath("004", "US2") != "proofs/004/US2.json" {
		t.Fatal(tg.CompanionReferencePath("004", ""), tg.CompanionProofPath("004", "US2"))
	}
	tg.CompanionPath = "documcp"
	if tg.CompanionReferencePath("004", "US1") != "documcp/references/004/US1.patch" || tg.CompanionProofPath("004", "") != "documcp/proofs/004.json" {
		t.Fatal(tg.CompanionReferencePath("004", "US1"), tg.CompanionProofPath("004", ""))
	}
}
