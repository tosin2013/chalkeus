package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/images"
	"github.com/tosin2013/specfoundry/internal/paths"
)

// loadTarget loads the --target profile (the fixture when unset) and checks its toolchain.
func loadTarget(cmd *cobra.Command, l paths.Layout, cfg *config.Config) (config.Target, error) {
	name, _ := cmd.Flags().GetString("target")
	t, err := cfg.LoadTarget(name, config.OriginRepo(l.Root))
	if err != nil || t.Fixture {
		return t, err
	}
	cat, err := images.LoadCatalog(l)
	if err != nil {
		return t, Exit(ExitError, "%v", err)
	}
	lock, _ := images.ReadLock(l)
	return t, images.CheckTarget(cat, lock, t)
}

func runTargets(cmd *cobra.Command, _ []string) error {
	l, err := layout()
	if err != nil {
		return err
	}
	cfg, err := config.Load(l.Root, false, false)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	f := cfg.Fixture()
	fmt.Fprintf(out, "%s\t%s@%s\ttoolchain=%s\t(pilot.env)\n", f.Name, f.Repo, f.Branch, f.Toolchain)
	cat, _ := images.LoadCatalog(l)
	lock, _ := images.ReadLock(l)
	origin := config.OriginRepo(l.Root)
	for _, name := range cfg.ListTargets() {
		t, err := cfg.LoadTarget(name, origin)
		if err == nil && cat != nil {
			err = images.CheckTarget(cat, lock, t)
		}
		if err != nil {
			fmt.Fprintf(out, "%s\tinvalid: %v\n", name, err)
			continue
		}
		regs := strings.Join(t.RegistryHosts, ",")
		if regs == "" {
			regs = "-"
		}
		fmt.Fprintf(out, "%s\t%s@%s\ttoolchain=%s\tregistries=%s\tcompanion=%s\n", t.Name, t.Repo, t.Branch, t.Toolchain, regs, companionRef(t))
	}
	return nil
}

// companionRef is the companion repository, plus the folder inside it when one repo serves
// several targets.
func companionRef(t config.Target) string {
	if t.CompanionPath == "" {
		return t.CompanionRepo
	}
	return t.CompanionRepo + "/" + t.CompanionPath
}
