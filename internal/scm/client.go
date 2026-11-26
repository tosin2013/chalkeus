package scm

import "context"

// Client is forge's interface to one Git host (contracts/host-interface.md). Reads run with the
// credential the client was built with: the operator's own login on the operator's machine
// (research R8), the test-author identity for companion drafts. No method merges, approves or
// accepts a change request (the no-merge guard enforces it).
type Client interface {
	// Host is the host this client talks to.
	Host() Host
	// Whoami is the account the client's credential acts as.
	Whoami(ctx context.Context) (Account, error)
	// Repository reads a repository's default branch and whether the caller can push.
	Repository(ctx context.Context, repo string) (RepoInfo, error)
	// ResolveAccount turns a username or id into an Account (Bitbucket: ids only).
	ResolveAccount(ctx context.Context, given string) (Account, error)
	// Permission is an account's effective access to repo.
	Permission(ctx context.Context, repo string, a Account) (Permission, error)
	// BranchProtection reads a branch's protection (doctor).
	BranchProtection(ctx context.Context, repo, branch string) (Protection, error)

	// OpenChangeRequests lists open requests (drafts included) whose head starts with headPrefix.
	OpenChangeRequests(ctx context.Context, repo, headPrefix string) ([]ChangeRequest, error)
	// ChangeRequest reads one request.
	ChangeRequest(ctx context.Context, repo string, id int) (ChangeRequest, error)
	// ChangedFiles lists the paths a request changes.
	ChangedFiles(ctx context.Context, repo string, cr ChangeRequest) ([]string, error)
	// FileAt reads a file at a revision.
	FileAt(ctx context.Context, repo, rev, path string) ([]byte, error)

	// CommitFiles writes files on top of base as one commit on a new branch (companion drafts
	// only) and returns the commit.
	CommitFiles(ctx context.Context, repo, base, branch string, files []File, message string) (string, error)
	// OpenDraft opens a draft request from head into base (companion drafts only).
	OpenDraft(ctx context.Context, repo, head, base, title, body string) (ChangeRequest, error)
	// CloseChangeRequest closes a request without merging and deletes its branch (reset-fixture
	// only, when the human asked).
	CloseChangeRequest(ctx context.Context, repo string, id int, comment string) error

	// ProbePublisher checks that the client's credential is a publisher credential reaching only
	// repo (research R4). It only reads (GitHub's probe creates an unreferenced blob, as before).
	ProbePublisher(ctx context.Context, repo string) TokenProbeResult
}
