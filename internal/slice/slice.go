// Package slice derives an item's slices from spec-status's per-phase task counts (026). A slice
// is one or more phases of the item's approved tasks.md. Everything here is pure: the input is
// the `spec status --json` item object, never tasks.md itself (C-01; research R1, R2).
package slice

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Phase is one `## Phase …` section as spec-status reports it (tasks.phases[], 0.4.0+).
type Phase struct {
	Heading string `json:"heading"`
	Line    int    `json:"line"`
	Story   *int   `json:"story"`
	Checked int    `json:"checked"`
	Total   int    `json:"total"`
}

// Slice is a group of phases that one run builds. ID is "US<n>", or All for an unsliced item.
type Slice struct {
	ID      string   `json:"id"`
	Index   int      `json:"index"`
	Phases  []string `json:"phases"`
	Checked int      `json:"checked"`
	Total   int      `json:"total"`
}

// All is the id of the single slice of an item with no user-story phases.
const All = "ALL"

// States of a slice on the target's default branch (data-model.md). "merged" here is the tick
// half only; readiness also requires that the slice has no open pilot PR (research R3).
const (
	Merged    = "merged"
	Untouched = "untouched"
	Started   = "started"
)

// Parse reads tasks.phases from a spec status item. ok is false when the field is absent (an older
// spec-status) or null (the tasks file was not read): slices are then unevaluable.
func Parse(item map[string]any) (phases []Phase, ok bool) {
	tasks, _ := item["tasks"].(map[string]any)
	raw, present := tasks["phases"]
	if !present || raw == nil {
		return nil, false
	}
	list, isList := raw.([]any)
	if !isList {
		return nil, false
	}
	phases = []Phase{}
	for _, x := range list {
		m, _ := x.(map[string]any)
		if m == nil {
			return nil, false
		}
		p := Phase{Heading: str(m["heading"]), Line: num(m["line"]), Checked: num(m["checked"]), Total: num(m["total"])}
		if s, has := m["story"]; has && s != nil {
			n := num(s)
			p.Story = &n
		}
		phases = append(phases, p)
	}
	return phases, true
}

// FromPhases groups phases into slices (spec FR-003, contracts/slices.md):
//   - a phase with a story starts slice US<story>;
//   - phases before the first story phase join the first story's slice;
//   - any other phase without a story joins the slice before it;
//   - with no story phases there is one slice, All.
//
// A story number used by two phases is an error naming both headings.
func FromPhases(phases []Phase) ([]Slice, error) {
	var out []Slice
	var lead []Phase // non-story phases before the first story
	seen := map[int]string{}
	for _, p := range phases {
		if p.Story == nil {
			if len(out) == 0 {
				lead = append(lead, p)
			} else {
				add(&out[len(out)-1], p)
			}
			continue
		}
		if prev, dup := seen[*p.Story]; dup {
			return nil, fmt.Errorf("story %d appears in two phases: %q and %q", *p.Story, prev, p.Heading)
		}
		seen[*p.Story] = p.Heading
		s := Slice{ID: "US" + strconv.Itoa(*p.Story), Index: len(out)}
		if len(out) == 0 {
			for _, l := range lead {
				add(&s, l)
			}
			lead = nil
		}
		add(&s, p)
		out = append(out, s)
	}
	if len(out) == 0 {
		s := Slice{ID: All, Phases: []string{}}
		for _, l := range lead {
			add(&s, l)
		}
		return []Slice{s}, nil
	}
	return out, nil
}

func add(s *Slice, p Phase) {
	s.Phases = append(s.Phases, p.Heading)
	s.Checked += p.Checked
	s.Total += p.Total
}

// State is a slice's tick state: merged (every task ticked), untouched (none) or started.
func State(s Slice) string {
	switch s.Checked {
	case s.Total: // first, so a slice with no tasks (0 of 0) counts as merged
		return Merged
	case 0:
		return Untouched
	default:
		return Started
	}
}

// Current is the first slice that is not merged (by ticks), the one step mode and the test author
// work on next. ok is false when every slice is merged.
func Current(slices []Slice) (Slice, bool) {
	for _, s := range slices {
		if State(s) != Merged {
			return s, true
		}
	}
	return Slice{}, false
}

// Sliced reports whether the item has two or more slices.
func Sliced(slices []Slice) bool { return len(slices) > 1 }

// Normalize upper-cases a slice id given on the command line ("us2" → "US2").
func Normalize(id string) string { return strings.ToUpper(strings.TrimSpace(id)) }

// Find returns the slice with id (case-insensitive). All is never a valid id to ask for: the whole
// item is run by giving no slice.
func Find(slices []Slice, id string) (Slice, bool) {
	id = Normalize(id)
	if id == "" || id == All {
		return Slice{}, false
	}
	for _, s := range slices {
		if s.ID == id {
			return s, true
		}
	}
	return Slice{}, false
}

// Earlier returns the slices before s, in order.
func Earlier(slices []Slice, s Slice) []Slice {
	if s.Index <= 0 || s.Index > len(slices) {
		return nil
	}
	return slices[:s.Index]
}

// Last reports whether s is the item's last slice.
func Last(slices []Slice, s Slice) bool { return len(slices) > 0 && s.Index == len(slices)-1 }

// IDs lists slice ids in order.
func IDs(slices []Slice) []string {
	out := make([]string, len(slices))
	for i, s := range slices {
		out[i] = s.ID
	}
	return out
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func num(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

// CurrentOf is an item's current slice from a `spec status --json` document: the first slice not
// merged, or "" when the item isn't sliced, every slice is merged, or slices can't be read (an
// older spec-status). Step mode and the test author use it to work slice by slice (026 R9).
func CurrentOf(statusJSON, item string) string {
	var doc map[string]any
	if json.Unmarshal([]byte(statusJSON), &doc) != nil {
		return ""
	}
	items, _ := doc["items"].([]any)
	for _, x := range items {
		m, _ := x.(map[string]any)
		if m["id"] != item {
			continue
		}
		phases, ok := Parse(m)
		if !ok {
			return ""
		}
		slices, err := FromPhases(phases)
		if err != nil || !Sliced(slices) {
			return ""
		}
		if cur, ok := Current(slices); ok {
			return cur.ID
		}
	}
	return ""
}
