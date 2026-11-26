package names

import (
	"bytes"
	"strings"
	"testing"
)

func quiet(t *testing.T) *bytes.Buffer {
	t.Helper()
	b := &bytes.Buffer{}
	oldOut, oldDir := Out, NoticeDir
	dir := t.TempDir()
	Out, NoticeDir = b, func() string { return dir }
	shown = map[string]bool{}
	t.Cleanup(func() { Out, NoticeDir = oldOut, oldDir })
	return b
}

func TestEnvNewWinsOldWorks(t *testing.T) {
	b := quiet(t)
	t.Setenv("PILOT_ROOT", "/old")
	if v := Env("ROOT"); v != "/old" || !strings.Contains(b.String(), "PILOT_ROOT is the old name of FORGE_ROOT") {
		t.Fatalf("old name: %q %q", v, b.String())
	}
	t.Setenv("FORGE_ROOT", "/new")
	if v := Env("ROOT"); v != "/new" || !strings.Contains(b.String(), "FORGE_ROOT is set and wins; PILOT_ROOT is ignored") {
		t.Fatalf("both: %q %q", v, b.String())
	}
	if _, ok := LookupEnv("NOT_SET_ANYWHERE"); ok {
		t.Fatal("unset")
	}
}

func TestNoticeOnce(t *testing.T) {
	b := quiet(t)
	Notice("x", "hello")
	Notice("x", "hello")
	if strings.Count(b.String(), "hello") != 1 {
		t.Fatalf("%q", b.String())
	}
}

func TestCanonical(t *testing.T) {
	b := quiet(t)
	got := Canonical(map[string]string{"FORGE_MODEL": "new", "PILOT_MODEL": "old", "PILOT_APPROVERS": "me", "RETRY_LIMIT": "2"}, "forge.env")
	if got["PILOT_MODEL"] != "new" || got["PILOT_APPROVERS"] != "me" || got["RETRY_LIMIT"] != "2" || len(got) != 3 {
		t.Fatalf("%v", got)
	}
	out := b.String()
	if !strings.Contains(out, "both names for PILOT_MODEL") || !strings.Contains(out, "old PILOT_ names (PILOT_APPROVERS)") {
		t.Fatalf("%q", out)
	}
	b.Reset()
	if got := Canonical(map[string]string{"FORGE_MODEL": "m"}, "f2"); got["PILOT_MODEL"] != "m" || b.Len() != 0 {
		t.Fatalf("new names only: %v %q", got, b.String())
	}
}

// Under go test, the default notice folder is empty: tests never leave markers on a machine.
func TestNoMarkersUnderGoTest(t *testing.T) {
	old := NoticeDir
	t.Cleanup(func() { NoticeDir = old })
	if d := NoticeDir(); d != "" {
		t.Fatalf("default notice folder under go test: %q", d)
	}
}

func TestDisplay(t *testing.T) {
	for in, want := range map[string]string{"PILOT_MODEL": "FORGE_MODEL", "PILOT_SCM_TOKEN": "PILOT_SCM_TOKEN", "PILOT_SCM_KIND": "PILOT_SCM_KIND", "PILOT_SCM_HOST": "FORGE_SCM_HOST", "PILOT_SCM": "FORGE_SCM",
		"FORGE_MODEL": "FORGE_MODEL", "CHALKEUS_CONFIG": "CHALKEUS_CONFIG"} {
		if got := Display(in); got != want {
			t.Errorf("Display(%s) = %s, want %s", in, got, want)
		}
	}
}
