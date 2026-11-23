package images

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/names"
	"github.com/tosin2013/chalkeus/internal/paths"
)

func catalog(t *testing.T) Catalog {
	t.Helper()
	root, _ := filepath.Abs("../..")
	c, err := LoadCatalog(paths.At(root))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCatalogShipped(t *testing.T) {
	c := catalog(t)
	if strings.Join(c.Names(), ",") != "go,jdk21-maven,node24,python,python3.13" {
		t.Fatal(c.Names())
	}
	if strings.Join(c.ImageKeys(), ",") != "TOOLCHAIN_GO_IMAGE,TOOLCHAIN_JDK21_MAVEN_IMAGE,TOOLCHAIN_NODE24_IMAGE,TOOLCHAIN_PYTHON313_IMAGE" {
		t.Fatal(c.ImageKeys())
	}
	for _, n := range []string{"go", "node24", "python3.13"} {
		if c[n].Proxy != "env" {
			t.Errorf("%s proxy %q", n, c[n].Proxy)
		}
	}
	j := c["jdk21-maven"]
	if j.Grader == nil || j.Grader.Memory != "2Gi" || j.Grader.DeadlineS != 900 || j.Cache == nil || j.Cache.Size != "2Gi" || j.Proxy != "maven-settings" {
		t.Fatalf("%+v", j)
	}
}

func TestToolchainImage(t *testing.T) {
	c := catalog(t)
	ref := "ghcr.io/x/specfoundry-pilot-toolchain-jdk21-maven@sha256:" + strings.Repeat("a", 64)
	if got, err := ToolchainImage(c, map[string]string{}, "python"); err != nil || got != "" {
		t.Fatal(got, err)
	}
	if got, err := ToolchainImage(c, map[string]string{"TOOLCHAIN_JDK21_MAVEN_IMAGE": ref}, "jdk21-maven"); err != nil || got != ref {
		t.Fatal(got, err)
	}
	for name, lock := range map[string]map[string]string{
		"jdk21-maven":  {},                                          // not built yet
		"node22":       {"TOOLCHAIN_JDK21_MAVEN_IMAGE": ref},        // unknown
		"jdk21-maven ": {"TOOLCHAIN_JDK21_MAVEN_IMAGE": "x:latest"}, // a tag, never accepted
	} {
		_, err := ToolchainImage(c, lock, strings.TrimSpace(name))
		if err == nil || !strings.HasPrefix(err.Error(), "TOOLCHAIN_UNAVAILABLE:") || !strings.Contains(err.Error(), "toolchain-image.yml") {
			t.Errorf("%q: %v", name, err)
		}
	}
}

func TestCheckTargetOverridesOnlyRaise(t *testing.T) {
	c := catalog(t)
	lock := map[string]string{"TOOLCHAIN_JDK21_MAVEN_IMAGE": "r/x@sha256:" + strings.Repeat("b", 64)}
	ok := config.Target{Name: "g", Toolchain: "jdk21-maven", GraderMemory: "3Gi", GraderDeadlineS: 1200, CacheSize: "4Gi"}
	if err := CheckTarget(c, lock, ok); err != nil {
		t.Fatal(err)
	}
	for key, tg := range map[string]config.Target{
		"PILOT_GRADER_MEMORY":     {Name: "g", Toolchain: "jdk21-maven", GraderMemory: "1Gi"},
		"PILOT_GRADER_DEADLINE_S": {Name: "g", Toolchain: "jdk21-maven", GraderDeadlineS: 60},
		"PILOT_CACHE_SIZE":        {Name: "g", Toolchain: "jdk21-maven", CacheSize: "512Mi"},
	} {
		err := CheckTarget(c, lock, tg)
		if err == nil || !strings.Contains(err.Error(), names.Display(key)) || !strings.HasPrefix(err.Error(), "TARGET_INVALID:") {
			t.Errorf("%s: %v", key, err)
		}
	}
}

// Owner decision 2026-11-17: a target may bring its own toolchain image, pinned by digest; the
// grader/publisher tools image stays specfoundry's.
func TestOperatorSuppliedToolchain(t *testing.T) {
	c := catalog(t)
	own := "quay.io/acme/toolchain-rust@sha256:" + strings.Repeat("c", 64)
	tg := config.Target{Name: "r", Toolchain: "rust-1.83", ToolchainImage: own}
	if err := CheckTarget(c, map[string]string{}, tg); err != nil {
		t.Fatal(err)
	}
	tc, ok := c.For(tg)
	if !ok || !tc.OperatorSupplied || tc.Grader == nil || tc.Grader.Memory != "2Gi" || tc.Cache == nil {
		t.Fatalf("%+v", tc)
	}
	for name, bad := range map[string]config.Target{
		"catalog name with own image": {Name: "r", Toolchain: "node24", ToolchainImage: own},
		"tag, not digest":             {Name: "r", Toolchain: "rust", ToolchainImage: "quay.io/acme/rust:1.83"},
		"lowers grader memory":        {Name: "r", Toolchain: "rust", ToolchainImage: own, GraderMemory: "1Gi"},
	} {
		if err := CheckTarget(c, map[string]string{}, bad); err == nil || !strings.HasPrefix(err.Error(), "TARGET_INVALID:") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// 025 FR-002: the stub harness is a layer on top of the built worker, never part of it.
func TestStubDockerfile(t *testing.T) {
	df := StubDockerfile("localhost:5001/pilot-worker@sha256:abc")
	for _, want := range []string{"FROM localhost:5001/pilot-worker@sha256:abc\n", "COPY --chmod=0755 stub-opencode /usr/local/bin/opencode\n", "COPY scenarios /usr/local/lib/pilot/stub-scenarios\n"} {
		if !strings.Contains(df, want) {
			t.Fatalf("missing %q in\n%s", want, df)
		}
	}
	root, _ := filepath.Abs("../..")
	if _, err := os.Stat(filepath.Join(root, "pilot", "images", "worker", "stub-opencode")); err == nil {
		t.Fatal("the stub must not be in the worker image's build context")
	}
	if _, err := os.Stat(filepath.Join(root, "pilot", "images", "stub", "Dockerfile")); err == nil {
		t.Fatal("pilot/images/stub must have no Dockerfile, so no image workflow builds it")
	}
}
