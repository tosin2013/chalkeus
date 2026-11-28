package author

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"

	"github.com/tosin2013/chalkeus/internal/evidence"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/ghloop"
	"github.com/tosin2013/chalkeus/internal/grader"
	"github.com/tosin2013/chalkeus/internal/proof"
)

// CodeNoReference: author prove found no reference change in the companion.
const CodeNoReference = "NO_REFERENCE"

// ProveRecord is `author prove`'s evidence (proves/<id>/prove.json): names, verdicts and hashes.
type ProveRecord struct {
	ID        string           `json:"prove_id"`
	Target    string           `json:"target"`
	Item      string           `json:"item"`
	Slice     string           `json:"slice,omitempty"`
	Companion string           `json:"companion_commit"`
	Result    string           `json:"result"`
	Reference *ReferenceResult `json:"reference,omitempty"`
	PR        map[string]any   `json:"pr,omitempty"`
}

// Prove is `./chalkeus forge author prove` (041 T012): run the reference check on the hidden tests
// already merged in the companion, and open a companion pull request with the proof. It is run
// by the owner: the hidden tests' output is printed in the owner's terminal, never stored. A
// failed check opens no pull request.
func Prove(ctx context.Context, e Env, item string) (*proof.Proof, error) {
	if e.Reference == nil {
		return nil, fmt.Errorf("author prove needs the cluster's reference check")
	}
	res, err := e.Ready(e.Target, item, e.Slice)
	if err != nil {
		return nil, err
	}
	if res.Decision != "allow" {
		return nil, refuse(firstReason(res.Reasons), "item %s isn't ready: %s", item, strings.Join(res.Reasons, ", "))
	}
	cdir, ccommit, err := fixture.FreshClone(e.Target.Host(), e.Target.CompanionRepo, "", 1)
	if err != nil {
		return nil, fmt.Errorf("companion repository %s: %v", e.Target.CompanionRepo, err)
	}
	defer func() { _ = os.RemoveAll(cdir) }()
	dir := e.companionDir(item)
	tree, err := proof.TreeSHA256(filepath.Join(cdir, filepath.FromSlash(dir)))
	if err != nil {
		return nil, err
	}
	if tree == "" {
		return nil, refuse("ITEM_NO_HIDDEN_TESTS", "%s has no hidden tests under %s/", e.Target.CompanionRepo, dir)
	}
	rf := ReferenceFile{Path: e.Target.CompanionReferencePath(item, e.Slice), Commit: ccommit}
	if b, err := os.ReadFile(filepath.Join(cdir, filepath.FromSlash(rf.Path))); err == nil {
		rf.Content = b
	}
	if rf.Content == nil {
		return nil, refuse(CodeNoReference, "no reference change at %s:%s; add a correct implementation of the item there as a patch against %s, or record reference: none in %s",
			e.Target.CompanionRepo, rf.Path, e.Target.Branch, e.Target.CompanionProofPath(item, e.Slice))
	}
	tarb, _, err := grader.Overlay(cdir, e.Target.CompanionRepo, ccommit, dir, item)
	if err != nil {
		return nil, err
	}
	id := time.Now().UTC().Format("20060102T150405Z")
	pdir := filepath.Join(e.EvidenceRoot, "proves", id)
	rec := &ProveRecord{ID: id, Target: e.Target.Name, Item: item, Slice: e.Slice, Companion: ccommit, Result: "error"}
	defer func() { _ = evidence.WriteJSON(filepath.Join(pdir, "prove.json"), rec) }()
	fmt.Fprintf(e.Out, "proving %s item %s (%s@%s) against %s: each hidden file on the base once and on the reference %d times; this takes a while...\n",
		e.Target.Name, item, e.Target.CompanionRepo, short(ccommit), rf.Path, e.Target.Repeats())
	report, hidden, err := e.Reference(pdir, "prove-"+id, item, res, tarb, rf.Content)
	if err != nil {
		return nil, err
	}
	rr, code := ReferenceOutcome(report, rf)
	rr.GradePath = filepath.Join("reference", "grade.json")
	rec.Reference, rec.Result = rr, rr.Result
	if len(hidden) > 0 {
		fmt.Fprintf(e.Out, "----- the hidden tests' output on the base and the reference (your terminal only, not stored) -----\n%s\n-----\n", hidden)
	}
	fmt.Fprintf(e.Out, "reference check: %s\n", rr.Summary())
	if code != CodeDraftOK {
		return nil, &Error{Code: code, Detail: referenceDetail[code] + " (" + rr.Summary() + "); no pull request opened"}
	}
	p := &proof.Proof{Version: 1, Item: item, Slice: e.Slice, Reference: "patch", ReferencePath: rf.Path, ReferenceSHA256: rf.SHA256(),
		TestsTreeSHA256: tree, BaseRevision: res.Commit, Repeats: rr.Repeats, Result: proof.Proven,
		CheckedAt: evidence.Now(), ForgeVersion: ForgeVersion}
	if res.ManifestChange != nil {
		p.ManifestCommit = res.ManifestChange.Commit
	}
	for _, f := range rr.Files {
		p.Files = append(p.Files, proof.File(f))
	}
	idn, err := e.Identity(ctx, e.Target)
	if err != nil {
		return p, err
	}
	path := e.Target.CompanionProofPath(item, e.Slice)
	title := fmt.Sprintf("test(%s): proof of the hidden tests against the reference change", item)
	if e.Slice != "" {
		title = fmt.Sprintf("test(%s): proof of the %s hidden tests against the reference change", item, e.Slice)
	}
	body := fmt.Sprintf("Proof for **%s** item **%s**%s (Chalkeus 041 `author prove`).\n\n- Reference check: %s, against `%s` (`%s`).\n- Hidden tests at `%s/` (tree `%s`, companion `%s`); base `%s`.\n- Adds only `%s`.\n\n**Review before merging:** merging lets runs of this item start (readiness `HIDDEN_TESTS_UNVERIFIED`).\n",
		e.Target.Repo, item, sliceSuffix(e.Slice), rr.Summary(), rf.Path, short(rf.SHA256()), dir, short(tree), short(ccommit), short(res.Commit), path)
	pr, commit, err := openFilePR(ctx, idn.Client, e.Target.CompanionRepo, "author/"+item+"-prove-"+strings.ToLower(id), path, p.Marshal(), title, body)
	if err != nil {
		return p, err
	}
	rec.PR = map[string]any{"url": pr.GetHTMLURL(), "number": pr.GetNumber(), "commit": commit, "identity": idn.Login}
	fmt.Fprintf(e.Out, "opened %s\nReview and merge it; then readiness accepts the item's hidden tests.\n", pr.GetHTMLURL())
	return p, nil
}

func sliceSuffix(s string) string {
	if s == "" {
		return ""
	}
	return " slice **" + s + "**"
}

func firstReason(rs []string) string {
	if len(rs) == 0 {
		return "ITEM_NOT_READY"
	}
	return rs[0]
}

// openFilePR opens a companion pull request that adds or replaces one file.
func openFilePR(ctx context.Context, c *github.Client, repo, branch, path string, content []byte, title, body string) (*github.PullRequest, string, error) {
	owner, name := ghloop.SplitRepo(repo)
	r, _, err := c.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, "", err
	}
	ref, _, err := c.Git.GetRef(ctx, owner, name, "refs/heads/"+r.GetDefaultBranch())
	if err != nil {
		return nil, "", err
	}
	parent := ref.GetObject().GetSHA()
	pc, _, err := c.Git.GetCommit(ctx, owner, name, parent)
	if err != nil {
		return nil, "", err
	}
	blob, _, err := c.Git.CreateBlob(ctx, owner, name, github.Blob{Content: github.Ptr(string(content)), Encoding: github.Ptr("utf-8")})
	if err != nil {
		return nil, "", err
	}
	entries := []*github.TreeEntry{{Path: github.Ptr(path), Mode: github.Ptr("100644"), Type: github.Ptr("blob"), SHA: blob.SHA}}
	return commitAndOpen(ctx, c, owner, name, r.GetDefaultBranch(), parent, pc.GetTree().GetSHA(), entries, branch, title, body)
}
