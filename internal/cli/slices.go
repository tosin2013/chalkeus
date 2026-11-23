package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tosin2013/chalkeus/internal/config"
	"github.com/tosin2013/chalkeus/internal/fixture"
	"github.com/tosin2013/chalkeus/internal/gate"
	"github.com/tosin2013/chalkeus/internal/slice"
)

// slicesCmd is `./chalkeus forge slices --target <name> --item NNN [--json]` (026, contracts/cli.md): a
// read-only view of an item's slices, from spec status on a fresh clone of the default branch.
func slicesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "slices --target <name> --item NNN [--json]",
		Short: "read-only: list an item's slices (from spec status per-phase counts), their state and the current one",
		Args:  cobra.NoArgs,
		RunE:  runSlices,
	}
	addTargetFlag(c)
	requireTarget(c)
	c.Flags().String("item", "", "the roadmap item NNN")
	_ = c.MarkFlagRequired("item")
	c.Flags().Bool("json", false, "print the slices as JSON")
	return c
}

func runSlices(cmd *cobra.Command, _ []string) error {
	item, _ := cmd.Flags().GetString("item")
	asJSON, _ := cmd.Flags().GetBool("json")
	l, err := layout()
	if err != nil {
		return err
	}
	cfg, err := config.Load(l.Root, false, false)
	if err != nil {
		return Exit(ExitError, "%v", err)
	}
	t, err := loadTarget(cmd, l, cfg)
	if err != nil {
		return err
	}
	if t.Fixture {
		return Exit(ExitError, "slices reads a --target repository, not the fixture")
	}
	dir, commit, err := fixture.FreshClone(t.Repo, t.Branch, 1)
	if err != nil {
		return Exit(ExitError, "could not clone %s@%s: %v", t.Repo, t.Branch, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	src := gate.ExecStatus{}
	_, stdout := src.Status(dir)
	return printSlices(cmd.OutOrStdout(), item, stdout, src.Version(), commit, asJSON)
}

// sliceRow is one line of `./chalkeus forge slices --json`.
type sliceRow struct {
	ID      string   `json:"id"`
	Index   int      `json:"index"`
	State   string   `json:"state"`
	Checked int      `json:"checked"`
	Total   int      `json:"total"`
	Phases  []string `json:"phases"`
}

// printSlices renders the slices of item from a spec status document. It exits 2 when the
// document has no tasks.phases for the item (SLICE_UNEVALUABLE).
func printSlices(out io.Writer, item, stdout, version, commit string, asJSON bool) error {
	var doc map[string]any
	_ = json.Unmarshal([]byte(stdout), &doc)
	var itemDoc map[string]any
	items, _ := doc["items"].([]any)
	for _, x := range items {
		if m, _ := x.(map[string]any); m["id"] == item {
			itemDoc = m
		}
	}
	if itemDoc == nil {
		return Exit(ExitError, "item %s is not on the roadmap (spec status)", item)
	}
	phases, ok := slice.Parse(itemDoc)
	if !ok {
		return Exit(ExitError, "SLICE_UNEVALUABLE: spec status (%s) reports no tasks.phases for item %s; slices need spec-status 0.4.0+ and a readable tasks.md", version, item)
	}
	slices, err := slice.FromPhases(phases)
	if err != nil {
		return Exit(ExitError, "SLICE_UNKNOWN: %v", err)
	}
	rows := make([]sliceRow, 0, len(slices))
	for _, s := range slices {
		rows = append(rows, sliceRow{ID: s.ID, Index: s.Index, State: slice.State(s), Checked: s.Checked, Total: s.Total, Phases: s.Phases})
	}
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}
	fmt.Fprintf(out, "item %s — %d slice(s) (from %s, %s)\n", item, len(slices), version, shortRev(commit))
	for _, r := range rows {
		fmt.Fprintf(out, "  %-4s %-9s %d/%d   %s\n", r.ID, r.State, r.Checked, r.Total, strings.Join(r.Phases, " · "))
	}
	switch cur, ok := slice.Current(slices); {
	case !slice.Sliced(slices):
		fmt.Fprintln(out, "not sliced: run the whole item without --slice")
	case ok:
		fmt.Fprintf(out, "current: %s\n", cur.ID)
	default:
		fmt.Fprintln(out, "current: none (every slice is merged)")
	}
	return nil
}

func shortRev(c string) string {
	if len(c) > 12 {
		return c[:12]
	}
	return c
}
