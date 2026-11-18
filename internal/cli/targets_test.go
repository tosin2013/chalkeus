package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/paths"
)

// configDir writes a pilot.env (and optional target profiles) into a temp SPECFOUNDRY_CONFIG.
func configDir(t *testing.T, targets map[string]string) string {
	t.Helper()
	d := t.TempDir()
	env := "PILOT_FIXTURE_REPO=o/x-pilot-fixture\nPILOT_MODEL=p/m\nPILOT_MODEL_EGRESS_HOSTS=secret-model.example\n"
	if err := os.WriteFile(filepath.Join(d, "pilot.env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	if len(targets) > 0 {
		_ = os.MkdirAll(filepath.Join(d, "targets"), 0o700)
		for n, body := range targets {
			if err := os.WriteFile(filepath.Join(d, "targets", n+".env"), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Setenv("SPECFOUNDRY_CONFIG", d)
	return d
}

func TestTargetsListsFixtureAndProfiles(t *testing.T) {
	configDir(t, map[string]string{
		"pysample": "PILOT_TARGET_REPO=me/pysample\nPILOT_TOOLCHAIN=python\nPILOT_TEST_COMMAND=python3 -m pytest -q\nPILOT_COMPANION_REPO=me/pysample-acceptance\nPILOT_REGISTRY_HOSTS=pypi.org,files.pythonhosted.org\n",
		"broken":   "PILOT_TARGET_REPO=nope\n",
		"java":     "PILOT_TARGET_REPO=me/j\nPILOT_TOOLCHAIN=jdk21-maven\nPILOT_TEST_COMMAND=mvn -B test\nPILOT_COMPANION_REPO=me/j-acceptance\n",
		"cobol":    "PILOT_TARGET_REPO=me/c\nPILOT_TOOLCHAIN=cobol\nPILOT_TEST_COMMAND=make test\nPILOT_COMPANION_REPO=me/c-acceptance\n",
	})
	code, out, errs := run("targets")
	if code != 0 {
		t.Fatalf("%d %s", code, errs)
	}
	for _, want := range []string{
		"fixture\to/x-pilot-fixture@main\ttoolchain=python\t(pilot.env)\n",
		"pysample\tme/pysample@main\ttoolchain=python\tregistries=pypi.org,files.pythonhosted.org\tcompanion=me/pysample-acceptance\tlevel=minimal\n",
		"broken\tinvalid: TARGET_INVALID:",
		"java\tme/j@main\ttoolchain=jdk21-maven\t", // a catalog toolchain with a locked image
		"cobol\tinvalid: TOOLCHAIN_UNAVAILABLE:",   // not in the catalog
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if strings.Contains(out, "secret-model.example") {
		t.Error("targets printed the model host")
	}
}

func TestResetFixtureRefusesARealTarget(t *testing.T) {
	configDir(t, nil)
	code, _, errs := run("reset-fixture", "--target", "greeting", "--yes")
	if code != ExitError || !strings.Contains(errs, "NOT_A_FIXTURE:") {
		t.Fatalf("%d %q", code, errs)
	}
}

func TestGateItemFlagRules(t *testing.T) {
	configDir(t, nil)
	if code, _, errs := run("gate", "--item", "002"); code != ExitError || !strings.Contains(errs, "--item needs --target") {
		t.Fatalf("%d %q", code, errs)
	}
	if code, _, errs := run("gate", "--target", "nope", "--item", "002"); code != ExitError || !strings.Contains(errs, "TARGET_UNKNOWN:") {
		t.Fatalf("%d %q", code, errs)
	}
	if code, _, errs := run("gate", "--target", "greeting", "A"); code != ExitError || !strings.Contains(errs, "needs --item") {
		t.Fatalf("%d %q", code, errs)
	}
	if code, _, _ := run("gate", "--item", "002", "A"); code != ExitError {
		t.Fatalf("an item check takes no A/B argument: %d", code)
	}
}

func TestRunItemFlagRules(t *testing.T) {
	configDir(t, nil)
	if code, _, errs := run("run", "--item", "002"); code != ExitError || !strings.Contains(errs, "--item needs --target") {
		t.Fatalf("%d %q", code, errs)
	}
	if code, _, errs := run("run", "--target", "greeting"); code != ExitError || !strings.Contains(errs, "needs --item NNN") {
		t.Fatalf("%d %q", code, errs)
	}
}

// 014 FR-004a: a paused item run continues against the target its run.json names.
func TestTargetForRunFollowsTheRecord(t *testing.T) {
	configDir(t, map[string]string{"greeting": "PILOT_TARGET_REPO=me/greeting-service\nPILOT_TOOLCHAIN=python\nPILOT_TEST_COMMAND=pytest\nPILOT_COMPANION_REPO=me/g-acceptance\n"})
	cfg, err := config.Load(t.TempDir(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	l := paths.At(t.TempDir())
	rd := filepath.Join(l.EvidenceRoot(), "20990101T000000Z")
	_ = os.MkdirAll(rd, 0o755)
	write := func(repo string) {
		raw := `{"run_id": "20990101T000000Z", "kind": "item", "item": "002", "target": {"name": "greeting", "repo": "` + repo + `"}}`
		_ = os.WriteFile(filepath.Join(rd, "run.json"), []byte(raw), 0o644)
	}
	cmd := resumeCmd()
	_ = cmd.Flags().Set("run", "20990101T000000Z")
	write("me/greeting-service")
	tg, err := targetFor(cmd, l, cfg)
	if err != nil || tg.Name != "greeting" || tg.Fixture {
		t.Fatalf("%+v %v", tg, err)
	}
	write("me/somewhere-else")
	if _, err := targetFor(cmd, l, cfg); err == nil || !strings.Contains(err.Error(), "restore the profile") {
		t.Fatalf("a changed profile must not silently redirect a paused run: %v", err)
	}
	if tg, _ := targetFor(downCmd(), l, cfg); !tg.Fixture {
		t.Fatal("no --run flag and no --target: the fixture")
	}
	if tg, _ := targetFor(statusCmd(), l, cfg); tg.Fixture {
		t.Fatal("no --run value means the latest run, here the item run")
	}
}
