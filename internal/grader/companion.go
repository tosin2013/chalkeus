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

	"github.com/tosin2013/specfoundry/internal/evidence"
)

// Hidden tests from the companion repository (014 R9). The pilot reads an item's folder on the
// host with the operator's own access; only this package turns it into the grader's overlay.
// The evidence keeps names and hashes, never contents, so nothing hidden reaches the record.

// CompanionFile is one hidden-test file as recorded in companion.json.
type CompanionFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Companion describes the overlay given to the grader.
type Companion struct {
	Repo        string          `json:"repo"`
	Commit      string          `json:"commit"`
	Item        string          `json:"item"`
	FileCount   int             `json:"file_count"`
	FilesSHA256 string          `json:"files_sha256"` // SHA-256 of the overlay tar
	Files       []CompanionFile `json:"files"`
}

// Overlay tars items/<item>/ of a companion checkout, with paths relative to that folder (so
// it lands mirrored over the project), and describes it. Entries are sorted and their times and
// owners fixed, so the same tests always give the same tar and hash.
func Overlay(companionDir, repo, commit, item string) ([]byte, Companion, error) {
	root := filepath.Join(companionDir, "items", item)
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			rel, _ := filepath.Rel(root, p)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, Companion{}, err
	}
	sort.Strings(files)
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	c := Companion{Repo: repo, Commit: commit, Item: item, FileCount: len(files)}
	for _, rel := range files {
		raw, err := os.ReadFile(filepath.Join(root, rel))
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
