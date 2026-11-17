package images

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/paths"
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
	if strings.Join(c.Names(), ",") != "jdk17-maven,python" {
		t.Fatal(c.Names())
	}
	if strings.Join(c.ImageKeys(), ",") != "TOOLCHAIN_JDK17_MAVEN_IMAGE" {
		t.Fatal(c.ImageKeys())
	}
	j := c["jdk17-maven"]
	if j.Grader == nil || j.Grader.Memory != "2Gi" || j.Grader.DeadlineS != 900 || j.Cache == nil || j.Cache.Size != "2Gi" || j.Proxy != "maven-settings" {
		t.Fatalf("%+v", j)
	}
}

func TestToolchainImage(t *testing.T) {
	c := catalog(t)
	ref := "ghcr.io/x/specfoundry-pilot-toolchain-jdk17-maven@sha256:" + strings.Repeat("a", 64)
	if got, err := ToolchainImage(c, map[string]string{}, "python"); err != nil || got != "" {
		t.Fatal(got, err)
	}
	if got, err := ToolchainImage(c, map[string]string{"TOOLCHAIN_JDK17_MAVEN_IMAGE": ref}, "jdk17-maven"); err != nil || got != ref {
		t.Fatal(got, err)
	}
	for name, lock := range map[string]map[string]string{
		"jdk17-maven":  {},                                          // not built yet
		"node22":       {"TOOLCHAIN_JDK17_MAVEN_IMAGE": ref},        // unknown
		"jdk17-maven ": {"TOOLCHAIN_JDK17_MAVEN_IMAGE": "x:latest"}, // a tag, never accepted
	} {
		_, err := ToolchainImage(c, lock, strings.TrimSpace(name))
		if err == nil || !strings.HasPrefix(err.Error(), "TOOLCHAIN_UNAVAILABLE:") || !strings.Contains(err.Error(), "toolchain-image.yml") {
			t.Errorf("%q: %v", name, err)
		}
	}
}

func TestCheckTargetOverridesOnlyRaise(t *testing.T) {
	c := catalog(t)
	lock := map[string]string{"TOOLCHAIN_JDK17_MAVEN_IMAGE": "r/x@sha256:" + strings.Repeat("b", 64)}
	ok := config.Target{Name: "g", Toolchain: "jdk17-maven", GraderMemory: "3Gi", GraderDeadlineS: 1200, CacheSize: "4Gi"}
	if err := CheckTarget(c, lock, ok); err != nil {
		t.Fatal(err)
	}
	for key, tg := range map[string]config.Target{
		"PILOT_GRADER_MEMORY":     {Name: "g", Toolchain: "jdk17-maven", GraderMemory: "1Gi"},
		"PILOT_GRADER_DEADLINE_S": {Name: "g", Toolchain: "jdk17-maven", GraderDeadlineS: 60},
		"PILOT_CACHE_SIZE":        {Name: "g", Toolchain: "jdk17-maven", CacheSize: "512Mi"},
	} {
		err := CheckTarget(c, lock, tg)
		if err == nil || !strings.Contains(err.Error(), key) || !strings.HasPrefix(err.Error(), "TARGET_INVALID:") {
			t.Errorf("%s: %v", key, err)
		}
	}
}
