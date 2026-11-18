package author

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/tosin2013/specfoundry/internal/evidence"
)

// File is one drafted file: its path in the item folder and its SHA-256, never its contents.
type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// CheckRecord is one `author check`.
type CheckRecord struct {
	At        string `json:"at"`
	DraftHash string `json:"draft_hash"`
	Result    string `json:"result"` // DRAFT_OK or a code
	GradePath string `json:"grade_json,omitempty"`
}

// Draft is a draft's evidence (data-model.md). It holds names, hashes and results only.
type Draft struct {
	ID             string         `json:"draft_id"`
	Target         string         `json:"target"`
	Repo           string         `json:"repo"`
	Item           string         `json:"item"`
	Title          string         `json:"title,omitempty"`
	SpecSHA256     string         `json:"spec_sha256"`
	BaseRevision   string         `json:"base_revision"`
	Level          string         `json:"level"`
	Identity       string         `json:"identity"`
	CanWriteTarget bool           `json:"can_write_target"`
	Companion      string         `json:"companion"`
	CompanionDir   string         `json:"companion_dir"`
	Workspace      string         `json:"workspace"`
	Files          []File         `json:"files"`
	Checks         []CheckRecord  `json:"checks"`
	Mapping        map[string]any `json:"mapping,omitempty"`
	PR             map[string]any `json:"pr,omitempty"`
	State          string         `json:"state"` // started | checked(pass) | checked(fail) | submitted
	Readiness      map[string]any `json:"readiness,omitempty"`

	dir string
}

// Root is where draft evidence lives: <evidence root>/drafts.
func Root(evidenceRoot string) string { return filepath.Join(evidenceRoot, "drafts") }

// NewDraft creates a draft folder named by the UTC time.
func NewDraft(root string) (*Draft, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	for i := 0; i < 5; i++ {
		id := time.Now().UTC().Format("20060102T150405Z")
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0o755); err == nil {
			return &Draft{ID: id, State: "started", dir: dir, Files: []File{}, Checks: []CheckRecord{}}, nil
		}
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("could not create a draft folder in %s", root)
}

// LoadDraft reads a draft by id, or the latest draft of target/item when id is "".
func LoadDraft(root, id, target, item string) (*Draft, error) {
	if id == "" {
		entries, _ := os.ReadDir(root)
		var ids []string
		for _, e := range entries {
			if e.IsDir() {
				ids = append(ids, e.Name())
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(ids)))
		for _, x := range ids {
			d, err := LoadDraft(root, x, "", "")
			if err == nil && d.Target == target && d.Item == item {
				return d, nil
			}
		}
		return nil, fmt.Errorf("no draft for target %s item %s: run `pilot/pilot author start` first", target, item)
	}
	raw, err := os.ReadFile(filepath.Join(root, id, "draft.json"))
	if err != nil {
		return nil, err
	}
	var d Draft
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, err
	}
	d.dir = filepath.Join(root, id)
	return &d, nil
}

// Dir is the draft's evidence folder.
func (d *Draft) Dir() string { return d.dir }

// Save writes draft.json.
func (d *Draft) Save() error { return evidence.WriteJSON(filepath.Join(d.dir, "draft.json"), d) }

// Event appends one line to events.jsonl (names, codes and counts only).
func (d *Draft) Event(event, detail string) error {
	f, err := os.OpenFile(filepath.Join(d.dir, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	raw, _ := json.Marshal(map[string]any{"at": evidence.Now(), "event": event, "detail": detail})
	_, err = f.Write(append(raw, '\n'))
	return err
}
