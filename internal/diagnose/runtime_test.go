package diagnose

import "testing"

func TestRuntimeMismatches(t *testing.T) {
	tasks := Tasks("## Phase 1: a\n- [ ] T001 Set `.nvmrc` to Node 24\n- [ ] T002 Support Node 22 too\n- [ ] T003 Pin Python 3.12 in the docs tooling\n")
	got := RuntimeMismatches(tasks, "node24", "")
	if len(got) != 1 || got[0].Task != "T002" || got[0].Named != "node 22" || got[0].Provides != "node 24" || got[0].Line != 3 {
		t.Fatalf("%+v", got)
	}
	if got := RuntimeMismatches(tasks, "python3.13", ""); len(got) != 1 || got[0].Task != "T003" {
		t.Fatalf("%+v", got)
	}
	if got := RuntimeMismatches(Tasks("- [ ] T001 Build with Go 1.26\n"), "go", "go1.27.4"); len(got) != 1 {
		t.Fatalf("%+v", got)
	}
	if got := RuntimeMismatches(tasks, "python", ""); got != nil {
		t.Fatalf("unknown toolchain version: %+v", got)
	}
	if f, v, ok := ToolchainRuntime("jdk21-maven", ""); !ok || f != "java" || v != "21" {
		t.Fatal(f, v, ok)
	}
}
