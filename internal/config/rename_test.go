package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/names"
)

// notices captures the rename notices (027) with markers in a temporary folder.
func notices(t *testing.T) *bytes.Buffer {
	t.Helper()
	b := &bytes.Buffer{}
	oldOut, oldDir := names.Out, names.NoticeDir
	dir := t.TempDir()
	names.Out, names.NoticeDir = b, func() string { return dir }
	t.Cleanup(func() { names.Out, names.NoticeDir = oldOut, oldDir })
	return b
}

// snapshot lists a folder's entries and modes, to prove the tool wrote nothing in it.
func snapshot(t *testing.T, d string) string {
	t.Helper()
	var s []string
	_ = filepath.Walk(d, func(p string, info os.FileInfo, err error) error {
		if err == nil {
			s = append(s, p+" "+info.Mode().String())
		}
		return nil
	})
	return strings.Join(s, "\n")
}

// 027 FR-005, FR-006: the config folder's lookup order, its notices, and no writes.
func TestConfigFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CHALKEUS_CONFIG", "")
	t.Setenv("SPECFOUNDRY_CONFIG", "")
	_ = os.Unsetenv("CHALKEUS_CONFIG")
	_ = os.Unsetenv("SPECFOUNDRY_CONFIG")
	cur, old := filepath.Join(home, ".config", "chalkeus"), filepath.Join(home, ".config", "specfoundry")

	b := notices(t)
	if d := Dir(); d != cur || b.Len() != 0 {
		t.Fatalf("neither folder: %s %q", d, b.String())
	}
	if _, err := os.Stat(cur); err == nil {
		t.Fatal("Dir must not create the folder")
	}

	_ = os.MkdirAll(old, 0o700)
	_ = os.WriteFile(filepath.Join(old, "pilot.env"), []byte("PILOT_MODEL=m\n"), 0o600)
	before := snapshot(t, home)
	if d := Dir(); d != old || !strings.Contains(b.String(), "mv "+old+" "+cur) {
		t.Fatalf("only the old folder: %s %q", d, b.String())
	}
	n := strings.Count(b.String(), "using the old config folder")
	Dir()
	if strings.Count(b.String(), "using the old config folder") != n {
		t.Error("the move notice is shown once")
	}
	if snapshot(t, home) != before {
		t.Error("the tool must not move, copy or create anything in either folder")
	}

	_ = os.MkdirAll(cur, 0o700)
	if d := Dir(); d != cur || !strings.Contains(b.String(), "the old "+old+" is still there") {
		t.Fatalf("both folders: %s %q", d, b.String())
	}

	t.Setenv("SPECFOUNDRY_CONFIG", "/legacy")
	if d := Dir(); d != "/legacy" || !strings.Contains(b.String(), "SPECFOUNDRY_CONFIG is the old name of CHALKEUS_CONFIG") {
		t.Fatalf("old setting: %s %q", d, b.String())
	}
	t.Setenv("CHALKEUS_CONFIG", "/new")
	if d := Dir(); d != "/new" || !strings.Contains(b.String(), "CHALKEUS_CONFIG is set and wins") {
		t.Fatalf("both settings: %s %q", d, b.String())
	}
}

// 027 FR-013: forge.env before pilot.env, FORGE_ names before PILOT_ names, in one Load.
func TestSettingsFileAndNames(t *testing.T) {
	b := notices(t)
	d := t.TempDir()
	if f := SettingsFile(d); f != filepath.Join(d, "forge.env") {
		t.Fatalf("no file yet: %s", f)
	}
	_ = os.WriteFile(filepath.Join(d, "pilot.env"), []byte("PILOT_FIXTURE_REPO=me/me-pilot-fixture\n"), 0o600)
	if f := SettingsFile(d); f != filepath.Join(d, "pilot.env") || !strings.Contains(b.String(), "pilot.env is the old name of forge.env") {
		t.Fatalf("old file: %s %q", f, b.String())
	}
	_ = os.WriteFile(filepath.Join(d, "forge.env"), []byte(
		"FORGE_FIXTURE_REPO=me/new-pilot-fixture\nFORGE_MODEL=p/m\nPILOT_MODEL=p/old\nPILOT_MODEL_EGRESS_HOSTS=h.example\n"), 0o600)
	t.Setenv("CHALKEUS_CONFIG", d)
	c, err := Load(t.TempDir(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	if c.Repo() != "me/new-pilot-fixture" || c.Get("PILOT_MODEL") != "p/m" || len(c.ModelHosts()) != 1 {
		t.Fatalf("forge.env wins and FORGE_ names win: %v", c.Values)
	}
	out := b.String()
	for _, want := range []string{"both forge.env and pilot.env", "both names for PILOT_MODEL", "old PILOT_ names (PILOT_MODEL_EGRESS_HOSTS)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing notice %q in %q", want, out)
		}
	}
}
