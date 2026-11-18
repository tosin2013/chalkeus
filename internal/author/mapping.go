package author

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Map is the coverage map (contracts/acceptance-map.md).
type Map struct {
	Version    int                 `json:"version"`
	SpecSHA256 string              `json:"spec_sha256"`
	Criteria   map[string][]string `json:"criteria"`
	Uncovered  []string            `json:"uncovered"`
}

// CheckMap validates draftDir/.acceptance-map.json against the spec's criteria and the drafted
// files. It returns a summary for evidence (counts and ids only).
func CheckMap(draftDir string, criteria []Criterion, specSHA string, files []File) (map[string]any, error) {
	raw, err := os.ReadFile(filepath.Join(draftDir, MapFile))
	if err != nil {
		return nil, refuse(CodeMapInvalid, "%s is missing: map each test to the criteria in criteria.json", MapFile)
	}
	var m Map
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, refuse(CodeMapInvalid, "%s is not valid JSON: %v", MapFile, err)
	}
	if m.SpecSHA256 != specSHA {
		return nil, refuse(CodeMapInvalid, "spec_sha256 is %q; the spec's is %s", m.SpecSHA256, specSHA)
	}
	known := map[string]bool{}
	for _, c := range criteria {
		known[c.ID] = true
	}
	have := map[string]bool{}
	for _, f := range files {
		have[f.Path] = true
	}
	for id, refs := range m.Criteria {
		if !known[id] {
			return nil, refuse(CodeMapInvalid, "criterion %s is not in the spec (see criteria.json)", id)
		}
		if len(refs) == 0 {
			return nil, refuse(CodeMapInvalid, "criterion %s lists no test; move it to uncovered", id)
		}
		for _, r := range refs {
			file := strings.SplitN(r, "::", 2)[0]
			if !have[file] {
				return nil, refuse(CodeMapInvalid, "criterion %s refers to %s, which is not a drafted file", id, file)
			}
		}
	}
	var want []string
	for _, c := range criteria {
		if _, ok := m.Criteria[c.ID]; !ok {
			want = append(want, c.ID)
		}
	}
	got := append([]string{}, m.Uncovered...)
	sort.Strings(got)
	sortedWant := append([]string{}, want...)
	sort.Strings(sortedWant)
	if strings.Join(got, ",") != strings.Join(sortedWant, ",") {
		return nil, refuse(CodeMapInvalid, "uncovered must list exactly the criteria with no test: %s", strings.Join(want, ", "))
	}
	return map[string]any{"criteria": len(criteria), "covered": len(m.Criteria), "uncovered": want}, nil
}
