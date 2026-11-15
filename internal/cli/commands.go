package cli

import "github.com/spf13/cobra"

// Command definitions mirror the reference pilot's argparse tree (pilot/specfoundry_pilot/cli.py).
// Bodies are filled in as each part is ported; until then they return notImplemented.

func upCmd() *cobra.Command {
	c := &cobra.Command{Use: "up", Short: "stand up kind + Calico, registry, images, proxies, publisher", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("up") }}
	c.Flags().Bool("cluster-only", false, "only kind + Calico (no config needed)")
	c.Flags().Bool("local-build", false, "build images locally into a kind-local registry instead of pulling pinned digests")
	return c
}

func downCmd() *cobra.Command {
	return &cobra.Command{Use: "down", Short: "tear down and verify", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("down") }}
}

func publisherTokenCmd() *cobra.Command {
	c := &cobra.Command{Use: "publisher-token", Short: "write publisher.env from a hidden prompt and verify its scope", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("publisher-token") }}
	c.Flags().Bool("check-only", false, "only verify the existing publisher.env")
	return c
}

func importOpencodeCmd() *cobra.Command {
	c := &cobra.Command{Use: "import-opencode", Short: "reuse your existing OpenCode config (provider + key file) for the worker", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("import-opencode") }}
	c.Flags().String("from", "~/.config/opencode", "OpenCode config dir")
	c.Flags().String("model", "", "<provider>/<model> to use (default: the config's \"model\")")
	c.Flags().String("fixture-repo", "", "owner/name of your -pilot-fixture repo, written to a new pilot.env")
	c.Flags().Bool("force", false, "replace an existing worker OpenCode config")
	return c
}

func resetFixtureCmd() *cobra.Command {
	c := &cobra.Command{Use: "reset-fixture", Short: "force-reset the throwaway fixture repo", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("reset-fixture") }}
	c.Flags().Bool("yes", false, "skip the confirmation prompt")
	return c
}

func runCmd() *cobra.Command {
	return &cobra.Command{Use: "run", Short: "start a new pilot run", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("run") }}
}

func resumeCmd() *cobra.Command {
	c := &cobra.Command{Use: "resume", Short: "resume a paused run", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("resume") }}
	c.Flags().String("run", "", "run id (default: the latest run)")
	return c
}

func approveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:       "approve {mode|retry|reconcile}",
		Short:     "approve the decision a paused run is waiting for (no-terminal use; your agent harness must ask you before this runs)",
		ValidArgs: []string{"mode", "retry", "reconcile"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE:      func(*cobra.Command, []string) error { return notImplemented("approve") },
	}
	c.Flags().String("run", "", "run id (see `pilot status`)")
	_ = c.MarkFlagRequired("run")
	return c
}

func gateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:       "gate {A|B}",
		Short:     "evaluate the gate for one task",
		ValidArgs: []string{"A", "B"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE:      func(*cobra.Command, []string) error { return notImplemented("gate") },
	}
	c.Flags().String("root", "", "evaluate this directory instead of a fresh clone")
	return c
}

func auditCmd() *cobra.Command {
	c := &cobra.Command{Use: "audit", Short: "evidence-only audit", Args: cobra.NoArgs, RunE: runAudit}
	c.Flags().String("run", "", "run id (default: the latest run)")
	return c
}

func statusCmd() *cobra.Command {
	c := &cobra.Command{Use: "status", Short: "print run state", Args: cobra.NoArgs, RunE: runStatus}
	c.Flags().String("run", "", "run id (default: the latest run)")
	return c
}

func githubCmd() *cobra.Command {
	g := &cobra.Command{Use: "github", Short: "004 GitHub loop: mirror the roadmap as issues; drift; watch for approvals"}
	sync := &cobra.Command{Use: "sync", Short: "create/update item issues and status labels from the roadmap (idempotent)", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("github sync") }}
	drift := &cobra.Command{Use: "drift", Short: "report differences between the roadmap and GitHub (exit 1 if any)", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("github drift") }}
	drift.Flags().String("run", "", "also write github/drift.json into this run's evidence")
	watch := &cobra.Command{Use: "watch", Short: "poll GitHub; resume when a valid /approve or a merge appears (Ctrl-C to stop)", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("github watch") }}
	watch.Flags().String("run", "", "run id")
	watch.Flags().Bool("detach", false, "run the watcher in the background; it survives the terminal closing")
	watch.Flags().Bool("stop", false, "stop this run's background watcher")
	_ = watch.MarkFlagRequired("run")
	watch.MarkFlagsMutuallyExclusive("detach", "stop")
	g.AddCommand(sync, drift, watch)
	return g
}

func appKeyCmd() *cobra.Command {
	return &cobra.Command{Use: "app-key", Short: "store the GitHub App private key from stdin (kept only if its check passes)", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("app-key") }}
}

func appTokenCmd() *cobra.Command {
	c := &cobra.Command{Use: "app-token", Short: "mint a GitHub App token and run the write probe", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return notImplemented("app-token") }}
	c.Flags().Bool("check", false, "mint, probe and discard a token")
	_ = c.MarkFlagRequired("check")
	return c
}
