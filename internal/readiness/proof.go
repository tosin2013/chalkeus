package readiness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/proof"
	"github.com/tosin2013/chalkeus/internal/slice"
)

func sliceOf(sl *slice.Slice) string {
	if sl == nil {
		return ""
	}
	return sl.ID
}

// proofProblems judges each overlaid folder's proof (041 data model): one proof per item, or per
// slice folder. It returns what is wrong per folder, and a warning when the owner recorded
// reference: none (allowed).
func proofProblems(cdir string, t config.Target, item string, overlay []string) (problems []string, warn string, refs []Reference) {
	itemDir := t.CompanionItemDir(item)
	var none []string
	for _, o := range overlay {
		sl := ""
		if o != itemDir {
			sl = strings.TrimPrefix(o, itemDir+"/")
		}
		st, p := ProofStatus(cdir, t, item, sl)
		if st.OK && p != nil && p.Reference == "patch" {
			refs = append(refs, Reference{Path: p.ReferencePath, SHA256: p.ReferenceSHA256})
		}
		label := item
		if sl != "" {
			label = item + "/" + sl
		}
		switch {
		case !st.OK:
			problems = append(problems, label+": "+st.Problem)
		case st.Warn:
			none = append(none, label)
		}
	}
	if len(none) > 0 {
		warn = fmt.Sprintf("the owner recorded reference: none for %s; the hidden tests are not proven", strings.Join(none, ", "))
	}
	return problems, warn, refs
}

// Reference is a proven folder's reference change (041): its companion path and hash, recorded so
// P11r can show a run's worker never sees either.
type Reference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ProofStatus reads [<path>/]proofs/<item>[/<slice>].json in the companion clone cdir and judges
// it against the hidden tests' current tree hash.
func ProofStatus(cdir string, t config.Target, item, sl string) (proof.Status, *proof.Proof) {
	dir := t.CompanionItemDir(item)
	if sl != "" {
		dir = t.CompanionSliceDir(item, sl)
	}
	tree, err := proof.TreeSHA256(filepath.Join(cdir, filepath.FromSlash(dir)))
	if err != nil {
		return proof.Status{Problem: "the hidden tests could not be read: " + err.Error()}, nil
	}
	raw, err := os.ReadFile(filepath.Join(cdir, filepath.FromSlash(t.CompanionProofPath(item, sl))))
	if errors.Is(err, fs.ErrNotExist) {
		return proof.Judge(nil, nil, tree), nil
	}
	if err != nil {
		return proof.Judge(nil, err, tree), nil
	}
	p, perr := proof.Parse(raw)
	return proof.Judge(p, perr, tree), p
}
