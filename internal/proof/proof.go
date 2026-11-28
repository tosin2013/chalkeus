// Package proof is the proof of an item's hidden tests (041 US3, specs/041-proven-hidden-tests/
// data-model.md): the reference check's verdicts for the exact hidden-test files it covers, kept
// in the private companion at [<path>/]proofs/<item>[/<slice>].json. It holds names, verdicts and
// hashes, never a test's contents or output.
package proof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MapFile is the test author's coverage map (018); it is not a test, so the tree hash skips it.
const MapFile = ".acceptance-map.json"

// Results.
const (
	Proven = "proven"
	Failed = "failed"
	None   = "none" // the owner's decision: no reference for this item
)

// File is one hidden test file's verdict.
type File struct {
	Path      string   `json:"path"`
	Base      string   `json:"base"`
	Reference []string `json:"reference"`
	Verdict   string   `json:"verdict"`
}

// Proof is proofs/<item>[/<slice>].json.
type Proof struct {
	Version         int    `json:"version"`
	Item            string `json:"item"`
	Slice           string `json:"slice"`
	Reference       string `json:"reference"` // patch | none
	ReferencePath   string `json:"reference_path,omitempty"`
	ReferenceSHA256 string `json:"reference_sha256,omitempty"`
	TestsTreeSHA256 string `json:"tests_tree_sha256,omitempty"`
	BaseRevision    string `json:"base_revision,omitempty"`
	ManifestCommit  string `json:"manifest_commit,omitempty"`
	Files           []File `json:"files,omitempty"`
	Repeats         int    `json:"repeats,omitempty"`
	Result          string `json:"result"`
	Code            string `json:"code,omitempty"`
	Model           string `json:"model,omitempty"`
	CheckedAt       string `json:"checked_at,omitempty"`
	ForgeVersion    string `json:"forge_version,omitempty"`
	DecidedBy       string `json:"decided_by,omitempty"`
}

// Parse reads a proof and checks its shape: a version, a known result, and for a proven or failed
// proof the tests' tree hash.
func Parse(raw []byte) (*Proof, error) {
	var p Proof
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("not valid JSON: %v", err)
	}
	switch {
	case p.Version != 1:
		return nil, fmt.Errorf("version %d (want 1)", p.Version)
	case p.Result != Proven && p.Result != Failed && p.Result != None:
		return nil, fmt.Errorf("result %q (want proven, failed or none)", p.Result)
	case p.Result == None && p.Reference != None:
		return nil, fmt.Errorf(`result none needs "reference": "none"`)
	case p.Result != None && p.TestsTreeSHA256 == "":
		return nil, fmt.Errorf("no tests_tree_sha256")
	}
	return &p, nil
}

// Marshal is the proof as committed: indented, with a final newline.
func (p *Proof) Marshal() []byte {
	b, _ := json.MarshalIndent(p, "", "  ")
	return append(b, '\n')
}

// TreeSHA256 hashes the hidden-test files under dir (path relative to dir, slash-separated, and
// each file's SHA-256, sorted by path), skipping the coverage map. An empty or missing dir hashes
// to "".
func TreeSHA256(dir string) (string, error) {
	var lines []string
	err := filepath.WalkDir(dir, func(p string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if de.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if rel == MapFile {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		h := sha256.Sum256(raw)
		lines = append(lines, rel+"\x00"+hex.EncodeToString(h[:]))
		return nil
	})
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return HashLines(lines), nil
}

// HashLines is TreeSHA256's hash of "path\x00sha256" lines (sorted here).
func HashLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	sort.Strings(lines)
	h := sha256.Sum256([]byte(strings.Join(lines, "\n") + "\n"))
	return hex.EncodeToString(h[:])
}

// Status is what readiness makes of a proof against the hidden tests it finds.
type Status struct {
	OK      bool   // the run may start
	Warn    bool   // allowed with a warning (reference: none)
	Problem string // missing | failed | stale | invalid: ...
}

// Judge compares a proof (nil when missing) with the hidden tests' current tree hash.
func Judge(p *Proof, perr error, tree string) Status {
	switch {
	case perr != nil:
		return Status{Problem: "invalid: " + perr.Error()}
	case p == nil:
		return Status{Problem: "missing"}
	case p.Result == None:
		return Status{OK: true, Warn: true}
	case p.Result != Proven:
		return Status{Problem: "failed (" + p.Code + ")"}
	case p.TestsTreeSHA256 != tree:
		return Status{Problem: "stale: the hidden tests changed since the proof"}
	}
	return Status{OK: true}
}
