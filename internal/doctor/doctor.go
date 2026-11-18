// Package doctor is `pilot doctor` (014): a read-only preflight that reports every problem with a
// target (and optionally one item) in one pass, before anyone prepares tests or starts a run.
// Runs and `pilot author start` still refuse at their own checks; doctor only reports.
package doctor

import (
	"fmt"
	"io"
	"strings"

	"github.com/tosin2013/specfoundry/internal/readiness"
)

// Status of one check.
const (
	Pass = "PASS"
	Warn = "WARN"
	Fail = "FAIL"
	Skip = "SKIP"
)

// Row is one check.
type Row struct {
	Area   string `json:"area"`
	Check  string `json:"check"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Report collects rows in order.
type Report struct{ Rows []Row }

// Add records a check.
func (r *Report) Add(area, check, status, detail string) {
	r.Rows = append(r.Rows, Row{Area: area, Check: check, Status: status, Detail: detail})
}

// OK records Pass when err is nil, else Fail with the error.
func (r *Report) OK(area, check string, err error, passDetail string) bool {
	if err != nil {
		r.Add(area, check, Fail, err.Error())
		return false
	}
	r.Add(area, check, Pass, passDetail)
	return true
}

// Failed reports whether any check failed (warnings don't count).
func (r *Report) Failed() bool {
	for _, x := range r.Rows {
		if x.Status == Fail {
			return true
		}
	}
	return false
}

// Readiness turns an all-rules readiness result into rows. Failures carry the rule's detail.
func (r *Report) Readiness(res readiness.Result) {
	for _, x := range res.Rules {
		st := map[string]string{"pass": Pass, "fail": Fail, "skipped": Skip}[x.Result]
		r.Add("item", x.ID, st, x.Detail)
	}
	if m := res.Mapping; m != nil {
		switch {
		case m["present"] != true:
			if res.HiddenFiles > 0 {
				r.Add("item", "coverage", Warn, "the hidden tests have no coverage map (written by hand?); fine, but uncovered criteria can't be shown")
			}
		case m["valid"] != true:
			r.Add("item", "coverage", Warn, "the coverage map can't be read")
		case m["stale"] == true:
			r.Add("item", "coverage", Warn, fmt.Sprintf("the coverage map was written for an older spec; uncovered: %v", m["uncovered"]))
		case len(fmt.Sprint(m["uncovered"])) > 2:
			r.Add("item", "coverage", Warn, fmt.Sprintf("uncovered criteria: %v", m["uncovered"]))
		default:
			r.Add("item", "coverage", Pass, "every criterion covered")
		}
	}
}

// DigestMatches compares the digests (the part after '@') of a running image and its lock entry.
func DigestMatches(running, locked string) bool {
	_, a, ok1 := strings.Cut(running, "@")
	_, b, ok2 := strings.Cut(locked, "@")
	return ok1 && ok2 && a == b
}

// Print writes the report as aligned lines and a summary.
func (r *Report) Print(w io.Writer) {
	area := ""
	for _, x := range r.Rows {
		if x.Area != area {
			area = x.Area
			fmt.Fprintf(w, "== %s\n", area)
		}
		line := fmt.Sprintf("  %-4s %s", x.Status, x.Check)
		if x.Detail != "" {
			line += ": " + x.Detail
		}
		fmt.Fprintln(w, line)
	}
	n := map[string]int{}
	for _, x := range r.Rows {
		n[x.Status]++
	}
	fmt.Fprintf(w, "doctor: %d pass, %d warn, %d fail, %d skipped\n", n[Pass], n[Warn], n[Fail], n[Skip])
}
