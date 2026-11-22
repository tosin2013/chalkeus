package grader

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tosin2013/specfoundry/internal/evidence"
)

// Hidden tests from the companion repository (014 R9). The pilot reads an item's folder on the
// host with the operator's own access; only this package turns it into the grader's overlay.
// The evidence keeps names and hashes, never contents, so nothing hidden reaches the record.

// MapFile is the test author's coverage map in an item's companion folder (018). It is not a
// test: it never goes into the grader overlay and isn't counted.
const MapFile = ".acceptance-map.json"

// CompanionFile is one hidden-test file as recorded in companion.json.
type CompanionFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Companion describes the overlay given to the grader.
type Companion struct {
	Repo        string          `json:"repo"`
	Commit      string          `json:"commit"`
	Dir         string          `json:"dir"` // the item's folder in the companion, e.g. cli-schema-mcp/items/006
	Dirs        []string        `json:"dirs,omitempty"` // 026: the folders overlaid, in order, when more than one
	Item        string          `json:"item"`
	FileCount   int             `json:"file_count"`
	FilesSHA256 string          `json:"files_sha256"` // SHA-256 of the overlay tar
	Files       []CompanionFile `json:"files"`
}

// Overlay tars dir (the item's folder, e.g. items/<item> or <path>/items/<item>) of a companion checkout, with paths relative to that folder (so
// it lands mirrored over the project), and describes it. Entries are sorted and their times and
// owners fixed, so the same tests always give the same tar and hash.
func Overlay(companionDir, repo, commit, dir, item string) ([]byte, Companion, error) {
	return OverlayDirs(companionDir, repo, commit, []string{dir}, item)
}

// OverlayDirs tars several companion folders as one overlay, in order (026: earlier slices first).
// Paths are relative to each folder; when two folders hold the same path, the later one wins. With
// one folder the tar and the description are exactly Overlay's.
func OverlayDirs(companionDir, repo, commit string, dirs []string, item string) ([]byte, Companion, error) {
	from := map[string]string{} // relative path -> the root it is read from (the last folder wins)
	for _, dir := range dirs {
		root := filepath.Join(companionDir, filepath.FromSlash(dir))
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type().IsRegular() && d.Name() != MapFile {
				rel, _ := filepath.Rel(root, p)
				from[filepath.ToSlash(rel)] = root
			}
			return nil
		})
		if err != nil {
			return nil, Companion{}, err
		}
	}
	files := make([]string, 0, len(from))
	for rel := range from {
		files = append(files, rel)
	}
	sort.Strings(files)
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	c := Companion{Repo: repo, Commit: commit, Item: item, FileCount: len(files)}
	if len(dirs) == 1 {
		c.Dir = dirs[0]
	} else {
		c.Dir, c.Dirs = strings.Join(dirs, ","), append([]string(nil), dirs...)
	}
	for _, rel := range files {
		raw, err := os.ReadFile(filepath.Join(from[rel], filepath.FromSlash(rel)))
		if err != nil {
			return nil, Companion{}, err
		}
		if err := tw.WriteHeader(&tar.Header{Name: rel, Mode: 0o644, Size: int64(len(raw)), Typeflag: tar.TypeReg, Format: tar.FormatPAX}); err != nil {
			return nil, Companion{}, err
		}
		if _, err := tw.Write(raw); err != nil {
			return nil, Companion{}, err
		}
		sum := sha256.Sum256(raw)
		c.Files = append(c.Files, CompanionFile{Path: rel, SHA256: hex.EncodeToString(sum[:])})
	}
	if err := tw.Close(); err != nil {
		return nil, Companion{}, err
	}
	sum := sha256.Sum256(buf.Bytes())
	c.FilesSHA256 = hex.EncodeToString(sum[:])
	return buf.Bytes(), c, nil
}

// WriteCompanion records companion.json in the run directory.
func WriteCompanion(runDir string, c Companion) error {
	return evidence.WriteJSON(filepath.Join(runDir, "companion.json"), c)
}
