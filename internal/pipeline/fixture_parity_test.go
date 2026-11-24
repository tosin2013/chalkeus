package pipeline

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/k8s"
	"github.com/tosin2013/chalkeus/internal/paths"
	"github.com/tosin2013/chalkeus/internal/worker"
)

// 014 T004: the fixture run must stay byte-identical while item runs are added (FR-015,
// SC-005). These goldens were captured from main before 014's first code change. Regenerate
// only for an intended fixture change: UPDATE_FIXTURE_GOLDEN=1 go test ./internal/pipeline.

func golden(t *testing.T, name, got string) {
	t.Helper()
	p := filepath.Join("testdata", "fixture", name)
	if os.Getenv("UPDATE_FIXTURE_GOLDEN") == "1" {
		if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("fixture output %s changed (FR-015); diff it against testdata/fixture/%s", name, name)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	r, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// fixtureClone is a git repo made from pilot/fixture, as reset-fixture pushes it.
func fixtureClone(t *testing.T, root string) (string, string) {
	t.Helper()
	d := t.TempDir()
	if err := exec.Command("cp", "-r", filepath.Join(root, "pilot", "fixture")+"/.", d).Run(); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"add", "-A"},
		{"-c", "user.name=t", "-c", "user.email=t@invalid", "commit", "-qm", "fixture"}} {
		if out, err := exec.Command("git", append([]string{"-C", d}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	out, _ := exec.Command("git", "-C", d, "rev-parse", "HEAD").Output()
	return d, strings.TrimSpace(string(out))
}

func TestFixtureInputsUnchanged(t *testing.T) {
	root := repoRoot(t)
	clone, base := fixtureClone(t, root)
	l := paths.At(t.TempDir())
	l.Manifests = filepath.Join(root, "pilot", "manifests")
	rd := filepath.Join(t.TempDir(), "20990101T000000Z")
	if err := os.MkdirAll(filepath.Join(rd, "attempts"), 0o755); err != nil {
		t.Fatal(err)
	}
	run := &evidence.Run{Dir: rd, ID: filepath.Base(rd), State: map[string]any{"run_id": filepath.Base(rd)}}
	for _, a := range []struct {
		task string
		n    int
	}{{"A", 1}, {"A", 2}, {"B", 1}} {
		if _, _, err := worker.MakeInputs(worker.Deps{L: l}, run, a.task, a.n, clone, base); err != nil {
			t.Fatal(err)
		}
		adir, _ := run.AttemptDir(a.task, a.n)
		name := a.task + "-" + string(rune('0'+a.n))
		p, _ := os.ReadFile(filepath.Join(adir, "prompt.md"))
		golden(t, name+".prompt.md", string(p))
		m, _ := os.ReadFile(filepath.Join(adir, "meta.json"))
		golden(t, name+".meta.json", strings.ReplaceAll(string(m), base, "<base>"))
		if ind, err := os.ReadFile(filepath.Join(adir, "induction.json")); err == nil {
			golden(t, name+".induction.json", string(ind))
		}
	}
}

func TestFixtureManifestsUnchanged(t *testing.T) {
	m := filepath.Join(repoRoot(t), "pilot", "manifests")
	render := func(tmpl string, v map[string]string) string {
		s, err := k8s.Render(filepath.Join(m, tmpl), v)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	golden(t, "worker-ns.yaml", render("worker-ns.yaml.tmpl", map[string]string{"NS": "pilot-w-a-1", "RUN_ID": "R"}))
	golden(t, "worker-job.yaml", render("worker-job.yaml.tmpl", map[string]string{"NS": "pilot-w-a-1", "DEADLINE": "1200", "CPU": "2", "MEMORY": "2Gi",
		"WORKER_IMAGE": "w@sha256:0", "PILOT_MODEL": "p/m", "SPEND_CAP": "1.00", "TOKEN_CAP": "2000000"}))
	golden(t, "grader.yaml", render("grader.yaml.tmpl", map[string]string{"NS": "pilot-g-a-1", "RUN_ID": "R"}))
	golden(t, "grader-job.yaml", render("grader-job.yaml.tmpl", map[string]string{"NS": "pilot-g-a-1", "TOOLS_IMAGE": "t@sha256:0"}))
}

func TestFixtureModeAndSummaryUnchanged(t *testing.T) {
	p, r, out := newP(t)
	p.confirmMode(r)
	golden(t, "mode.txt", out.String())
	// The committed T057 walkthrough record's summary is regenerated from a copy of it.
	src := filepath.Join(repoRoot(t), "specs", "016-go-pilot", "evidence", "20261117T001332Z")
	dst := filepath.Join(t.TempDir(), "20261117T001332Z")
	if err := exec.Command("cp", "-r", src, dst).Run(); err != nil {
		t.Fatal(err)
	}
	cr, err := evidence.OpenRun(dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cr.WriteSummary(); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dst, "summary.md"))
	want, _ := os.ReadFile(filepath.Join(src, "summary.md"))
	if !bytes.Equal(got, want) {
		t.Error("summary.md of a committed fixture run no longer regenerates byte-identical (FR-015)")
	}
}
