package diagnose

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Tool is a Spec Kit extension the route back to planning names (033 US5). Forge never installs
// or runs it (FR-023): the owner does, in their planning session.
type Tool struct {
	ID        string   `json:"id"`
	For       string   `json:"for"` // the route or trigger it serves
	Installed bool     `json:"installed"`
	Commands  []string `json:"commands,omitempty"` // when installed
	Install   string   `json:"install,omitempty"`  // when not
	Vetted    bool     `json:"vetted"`             // in Spec Kit's default catalog
}

// Line is the tool's line in the diagnosis text.
func (t Tool) Line() string {
	if t.Installed {
		return fmt.Sprintf("tool for %s: %s (installed)", t.For, strings.Join(t.Commands, ", then "))
	}
	note := ""
	if !t.Vetted {
		note = "; a community extension not vetted by Spec Kit: review it first"
	}
	return fmt.Sprintf("tool for %s: the %s extension, not installed (%s%s)", t.For, t.ID, t.Install, note)
}

// The planning tools (checked 2026-11-25 against Spec Kit 1.4.3's catalogs).
var (
	harnessTool = Tool{ID: "harness", For: "the research",
		Commands: []string{"/speckit.harness.explore", "/speckit.harness.report (writes research.md)"},
		Install:  "specify extension add harness --from https://github.com/formin/spec-kit-harness/archive/refs/tags/v1.0.0.zip"}
	bugTool = Tool{ID: "bug", For: "the defect", Vetted: true,
		Commands: []string{"/speckit.bug.assess with this diagnosis as the report", "/speckit.bug.fix", "/speckit.bug.test"},
		Install:  "specify extension add bug"}
)

// tools picks the planning tools for a back-to-planning diagnosis: the Research Harness for
// unknown work, Bug Triage for an unrecognised failure or failing project tests.
func tools(d Diagnosis, in Input, projectTestsFail bool) []Tool {
	installed := map[string]bool{}
	for _, e := range in.Extensions {
		installed[e] = true
	}
	var out []Tool
	add := func(t Tool) {
		t.Installed = installed[t.ID]
		if t.Installed {
			t.Install = ""
		} else {
			t.Commands = nil
		}
		out = append(out, t)
	}
	unknownReason := false
	for _, s := range d.Signals {
		if s.Kind == KindUnknownReason {
			unknownReason = true
		}
	}
	for _, tr := range d.Triggers {
		if tr == UnknownWork {
			add(harnessTool)
		}
	}
	if unknownReason || projectTestsFail {
		add(bugTool)
	}
	return out
}

// EnabledExtensions reads a Spec Kit extension registry (.specify/extensions/.registry) and
// returns the ids of the enabled extensions, sorted. Unreadable input gives none.
func EnabledExtensions(registry []byte) []string {
	var reg struct {
		Extensions map[string]struct {
			Enabled *bool `json:"enabled"`
		} `json:"extensions"`
	}
	if json.Unmarshal(registry, &reg) != nil {
		return nil
	}
	out := []string{}
	for id, e := range reg.Extensions {
		if e.Enabled == nil || *e.Enabled {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
