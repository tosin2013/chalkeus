package readiness

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/tosin2013/specfoundry/internal/config"
	"github.com/tosin2013/specfoundry/internal/slice"
)

// SliceInfo is what readiness records about the slice a run builds (026, data-model.md).
type SliceInfo struct {
	ID                 string     `json:"id"`
	Index              int        `json:"index"`
	Of                 int        `json:"of"`
	Phases             []string   `json:"phases"`
	Earlier            []string   `json:"earlier"`
	EarlierPhases      [][]string `json:"earlier_phases"`
	AllowedPathsSource string     `json:"allowed_paths_source"`
	DefaultRevision    string     `json:"default_revision"`
}

// PriorSlice is the slicing a local slice run of the item recorded when it was built: its phases
// and the item's slice count then. SLICE_MAP_CHANGED compares it with today's (026 G1).
type PriorSlice struct {
	Phases []string
	Of     int
}

// sliceState is readiness's view of the item's slices on the fresh clone.
type sliceState struct {
	known  bool          // tasks.phases was readable (spec-status 0.4.0+, tasks file read)
	err    error         // the phases could not be grouped (a repeated story number)
	slices []slice.Slice // in file order
}

func readSlices(itemDoc map[string]any) sliceState {
	phases, ok := slice.Parse(itemDoc)
	if !ok {
		return sliceState{}
	}
	s, err := slice.FromPhases(phases)
	return sliceState{known: true, err: err, slices: s}
}

// headRE matches the head branch of a pilot PR for the item or one of its slices:
// pilot/<run>/<item> or pilot/<run>/<item>-us<n>.
func headRE(item string) *regexp.Regexp {
	return regexp.MustCompile(`^pilot/[^/]+/` + regexp.QuoteMeta(item) + `(-us[0-9]+)?$`)
}

// headSuffix is the last path part of a pilot PR head for item and slice ("" = the whole item).
func headSuffix(item, sliceID string) string {
	if sliceID == "" {
		return item
	}
	return item + "-" + strings.ToLower(sliceID)
}

// openHeadsFor returns the open pilot PR heads among heads that block the run (contracts/slices.md,
// OPEN_ITEM_PR widened): for a whole-item run any head of the item or its slices; for a slice run a
// head of the whole item, of this slice, or of an earlier slice.
func openHeadsFor(heads []string, item string, sl *slice.Slice, earlier []slice.Slice) []string {
	re := headRE(item)
	block := map[string]bool{}
	if sl != nil {
		block[headSuffix(item, "")] = true
		block[headSuffix(item, sl.ID)] = true
		for _, e := range earlier {
			block[headSuffix(item, e.ID)] = true
		}
	}
	var out []string
	for _, h := range heads {
		if !re.MatchString(h) {
			continue
		}
		if sl == nil || block[h[strings.LastIndex(h, "/")+1:]] {
			out = append(out, h)
		}
	}
	return out
}

// openSlices lists the slice ids (among ids) that have an open pilot PR in heads.
func openSlices(heads []string, item string, ids []string) map[string]bool {
	out := map[string]bool{}
	for _, h := range heads {
		last := h[strings.LastIndex(h, "/")+1:]
		for _, id := range ids {
			if headRE(item).MatchString(h) && last == headSuffix(item, id) {
				out[id] = true
			}
		}
	}
	return out
}

// companionLayout lists the top-level entries of the item's companion folder (name -> is dir).
func companionLayout(itemDir string) map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(itemDir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.Name() == MapFile {
			continue
		}
		out[e.Name()] = e.IsDir()
	}
	return out
}

// overlayDirs decides which companion folders a run overlays (research R6), as companion-relative
// slash paths: overlay (this slice, or the whole item: every slice folder in order) and earlier
// (each earlier slice's folder, in order; slice runs only). stray is set for a mixed layout.
func overlayDirs(t config.Target, item string, ss sliceState, sl *slice.Slice, earlier []slice.Slice, entries map[string]bool) (overlay, earlierDirs, stray []string) {
	perSlice := false
	if ss.known && ss.err == nil {
		perSlice, stray = config.SliceLayout(entries, slice.IDs(ss.slices))
	}
	switch {
	case sl != nil && perSlice:
		overlay = []string{t.CompanionSliceDir(item, sl.ID)}
		for _, e := range earlier {
			earlierDirs = append(earlierDirs, t.CompanionSliceDir(item, e.ID))
		}
	case sl != nil:
		// A slice run needs per-slice folders: with a flat folder there is nothing to tell slices
		// apart, so the run is refused by the hidden-tests rule (no files in the slice's folder).
		overlay = []string{t.CompanionSliceDir(item, sl.ID)}
		for _, e := range earlier {
			earlierDirs = append(earlierDirs, t.CompanionSliceDir(item, e.ID))
		}
	case perSlice:
		for _, s := range ss.slices {
			if entries[s.ID] {
				overlay = append(overlay, t.CompanionSliceDir(item, s.ID))
			}
		}
	default:
		overlay = []string{t.CompanionItemDir(item)}
	}
	return overlay, earlierDirs, stray
}

func countDirs(cdir string, dirs []string) int {
	n := 0
	for _, d := range dirs {
		n += countFiles(filepath.Join(cdir, filepath.FromSlash(d)))
	}
	return n
}

// mapChanged reports a slice whose phases (or the item's slice count) changed since a local run of
// it was recorded, for slices that are merged now (026 G1, SLICE_MAP_CHANGED).
func mapChanged(prior map[string]PriorSlice, slices []slice.Slice) string {
	ids := make([]string, 0, len(prior))
	for id := range prior {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		p := prior[id]
		cur, ok := slice.Find(slices, id)
		if !ok {
			return fmt.Sprintf("slice %s was built locally but is no longer in tasks.md", id)
		}
		if slice.State(cur) != slice.Merged {
			continue
		}
		if !reflect.DeepEqual(p.Phases, cur.Phases) {
			return fmt.Sprintf("slice %s was built with phases %q, and tasks.md now groups %q", id, p.Phases, cur.Phases)
		}
		if p.Of != 0 && p.Of != len(slices) {
			return fmt.Sprintf("the item had %d slices when %s was built, and has %d now", p.Of, id, len(slices))
		}
	}
	return ""
}

var acceptanceRE = regexp.MustCompile(`^##\s+Item acceptance\s*$`)

// Acceptance lists the item-level acceptance checks a spec declares under "## Item acceptance"
// (026 FR-018): one per list item, its text without the bullet.
func Acceptance(specText string) []string {
	var out []string
	in := false
	bullet := regexp.MustCompile(`^\s*[-*]\s+(.+)$`)
	for _, line := range strings.Split(specText, "\n") {
		switch {
		case acceptanceRE.MatchString(line):
			in = true
		case in && headingRE.MatchString(line):
			in = false
		case in:
			if m := bullet.FindStringSubmatch(line); m != nil {
				out = append(out, strings.TrimSpace(m[1]))
			}
		}
	}
	return out
}
