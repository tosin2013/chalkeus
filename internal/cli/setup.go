package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tosin2013/specfoundry/internal/cluster"
	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/images"
	"github.com/tosin2013/specfoundry/internal/k8s"
	"github.com/tosin2013/specfoundry/internal/paths"
)

// credentialEnv is the env file holding the publisher credential (token or App token).
func credentialEnv(cfg *config.Config) (string, error) {
	cred, err := cfg.Credential()
	if err != nil {
		return "", err
	}
	if cred == "app" {
		return "", Exit(ExitError, "GitHub App credential: not implemented in the Go pilot yet (use PILOT_IMPL=python)")
	}
	return cfg.PublisherEnv, nil
}

func verifyPublisher(cmd *cobra.Command, l paths.Layout, cfg *config.Config, envFile string) ghloop.ProbeResult {
	res := ghloop.VerifyFile(envFile, cfg.Repo(), config.OriginRepo(l.Root), cfg.GithubIssues())
	ghloop.Report(cmd.OutOrStdout(), res)
	return res
}

func runUp(cmd *cobra.Command, _ []string) error {
	l, err := layout()
	if err != nil {
		return err
	}
	clusterOnly, _ := cmd.Flags().GetBool("cluster-only")
	localBuild, _ := cmd.Flags().GetBool("local-build")
	e := cluster.Env{L: l, Out: cmd.OutOrStdout()}
	_ = os.Remove(l.UpOK)
	var cfg *config.Config
	var credEnv string
	if !clusterOnly {
		if cfg, err = config.Load(l.Root, true, true); err != nil {
			return err
		}
		if credEnv, err = credentialEnv(cfg); err != nil {
			return err
		}
		if res := verifyPublisher(cmd, l, cfg, credEnv); !res.OK {
			return Exit(ExitError, "publisher token check failed; fix it with `pilot publisher-token`")
		}
	}
	if err := cluster.UpCluster(e, localBuild); err != nil {
		return Exit(ExitError, "%v", err)
	}
	if clusterOnly {
		fmt.Fprintln(e.Out, "== cluster only: skipping images, proxies and publisher")
		return nil
	}
	var imgs map[string]string
	source := "pinned"
	if localBuild {
		source = "local-build"
		if imgs, err = images.BuildAll(l, func(s string) { fmt.Fprintln(e.Out, s) }); err != nil {
			return Exit(ExitError, "%v", err)
		}
	} else {
		lock, err := images.ReadLock(l)
		if err != nil {
			return Exit(ExitError, "%v", err)
		}
		if imgs, err = images.LoadPinned(lock); err != nil {
			return Exit(ExitError, "%v", err)
		}
		fmt.Fprintln(e.Out, "== images: pulled by digest from pilot/images.lock")
	}
	if err := images.Save(l, imgs); err != nil {
		return Exit(ExitError, "%v", err)
	}
	if err := os.WriteFile(l.ImageSource, []byte(source+"\n"), 0o644); err != nil {
		return Exit(ExitError, "%v", err)
	}
	c, err := k8s.New(paths.Context)
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	if err := cluster.ApplyAll(e, c, cfg.ModelHosts(), imgs, credEnv); err != nil {
		return Exit(ExitError, "%v", err)
	}
	if err := cluster.WriteUpOK(l, imgs, source); err != nil {
		return Exit(ExitError, "%v", err)
	}
	fmt.Fprintln(e.Out, "== up complete")
	return nil
}

func runDown(cmd *cobra.Command, _ []string) error {
	l, err := layout()
	if err != nil {
		return err
	}
	code, err := cluster.Down(cluster.Env{L: l, Out: cmd.OutOrStdout()})
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	return Code(code)
}

// readHidden reads one line from the terminal without echo (the reference used `read -rs`
// from /dev/tty). The value goes straight into a 0600 file and is never printed.
func readHidden() (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", err
	}
	defer func() { _ = tty.Close() }()
	raw, err := term.ReadPassword(int(tty.Fd()))
	_, _ = tty.WriteString("\n")
	return strings.TrimSpace(string(raw)), err
}

func runPublisherToken(cmd *cobra.Command, _ []string) error {
	l, err := layout()
	if err != nil {
		return err
	}
	cfg, err := config.Load(l.Root, false, true)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if checkOnly, _ := cmd.Flags().GetBool("check-only"); checkOnly {
		if res := verifyPublisher(cmd, l, cfg, cfg.PublisherEnv); !res.OK {
			return Code(ExitFail)
		}
		return nil
	}
	staging := cfg.PublisherEnv + ".new"
	defer func() { _ = os.Remove(staging) }()
	fmt.Fprintln(out, "Paste the fine-grained token for the fixture repo (input hidden):")
	tok, err := readHidden()
	if err != nil || tok == "" {
		fmt.Fprintln(out, "no token written")
		return Code(ExitError)
	}
	if err := os.MkdirAll(cfg.ConfigDir, 0o700); err != nil {
		return Exit(ExitError, "%v", err)
	}
	if err := os.WriteFile(staging, []byte("PILOT_SCM_TOKEN="+tok+"\n"), 0o600); err != nil {
		return Exit(ExitError, "%v", err)
	}
	if res := verifyPublisher(cmd, l, cfg, staging); !res.OK {
		kept := "no token stored"
		if _, err := os.Stat(cfg.PublisherEnv); err == nil {
			kept = "your previous token is unchanged"
		}
		fmt.Fprintf(out, "  not saved: %s. Fix the token's scope and run this again.\n", kept)
		return Code(ExitFail)
	}
	if err := os.Rename(staging, cfg.PublisherEnv); err != nil {
		return Exit(ExitError, "%v", err)
	}
	fmt.Fprintf(out, "  saved: %s\n", cfg.PublisherEnv)
	return nil
}
