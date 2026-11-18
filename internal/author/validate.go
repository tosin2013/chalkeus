package author

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MapFile is the coverage map in the item folder (grader.MapFile, readiness.MapFile).
const MapFile = ".acceptance-map.json"

// DraftFiles lists the drafted test files (map excluded) with their hashes, and the draft's
// stable hash (paths and hashes, map included). Symlinks and special files are refused.
func DraftFiles(draftDir string) ([]File, string, error) {
	var files []File
	var mapHash string
	err := filepath.WalkDir(draftDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(draftDir, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return refuse(CodeDraftOutsideFolder, "%s is not a regular file (symlinks and special files are not allowed)", rel)
		}
		if strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
			return refuse(CodeDraftOutsideFolder, "%s is outside the draft folder", rel)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		h := hex.EncodeToString(sum[:])
		if rel == MapFile {
			mapHash = h
			return nil
		}
		files = append(files, File{Path: rel, SHA256: h})
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	if len(files) == 0 {
		return nil, "", refuse(CodeDraftEmpty, "%s has no test files", draftDir)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	h := sha256.New()
	for _, f := range files {
		h.Write([]byte(f.Path + "\x00" + f.SHA256 + "\n"))
	}
	h.Write([]byte(MapFile + "\x00" + mapHash + "\n"))
	return files, hex.EncodeToString(h.Sum(nil)), nil
}
