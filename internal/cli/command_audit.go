package cli

// CommandClass says whether the harness must ask the human before a command runs (027,
// specs/027-chalkeus-rename/contracts/command-audit.md). Every command under `forge` is listed; a
// command's subcommands share its class. TestCommandAudit fails on a command missing here, and on
// an ask command without a matching rule in the repository's .claude/settings.json.
const (
	ClassAsk  = "ask"  // decides, destroys or holds credentials: the harness asks the human first
	ClassFree = "free" // reads, or acts only after a human's decision recorded elsewhere
)

// CommandAudit is the class of each command under `forge`.
var CommandAudit = map[string]string{
	"approve":         ClassAsk, // records the human's decision (every step, including 026's acceptance)
	"reset-fixture":   ClassAsk, // force-pushes the fixture repository
	"down":            ClassAsk, // deletes the cluster
	"publisher-token": ClassAsk, // writes the publisher credential
	"app-key":         ClassAsk, // stores the GitHub App private key
	"app-token":       ClassAsk, // mints a GitHub App token
	"author":          ClassAsk, // the test author's session only, never the driving session
	"step":            ClassAsk, // the owner's manager loop, never the driving session
	"run":             ClassAsk, // starts a run (owner decision 2026-11-22)
	"resume":          ClassAsk, // continues a run (owner decision 2026-11-22)

	"up":              ClassFree, // stands up the local cluster; no decision
	"status":          ClassFree,
	"audit":           ClassFree,
	"doctor":          ClassFree,
	"gate":            ClassFree,
	"targets":         ClassFree,
	"slices":          ClassFree,
	"debt":            ClassFree, // 029: reads only; in App mode a read-only token, in memory
	"version":         ClassFree,
	"github":          ClassFree, // sync and drift mirror; watch resumes only after a human's approval or merge
	"import-opencode": ClassFree, // copies the operator's own model config into the config folder
}

// AskRulePrefixes are the command forms the repository's harness rules must cover for each ask
// command (027 stage 3): the in-repository shim and the installed binary (013).
var AskRulePrefixes = []string{"./chalkeus forge", "chalkeus forge"}
