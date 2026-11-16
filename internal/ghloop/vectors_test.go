package ghloop_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tosin2013/specfoundry/internal/ghloop"
	"github.com/tosin2013/specfoundry/internal/reconcile"
)

// T043 / FR-011: the Go pilot passes every shared vector in specs/004-github-loop/contracts/vectors/
// (contracts/parity.md).

var vectors = filepath.Join("..", "..", "specs", "004-github-loop", "contracts", "vectors")

func load(t *testing.T, name string, v any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(vectors, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatal(err)
	}
}

func TestApprovalVectors(t *testing.T) {
	var doc struct {
		Cases []struct {
			Name  string
			Input struct {
				Decision    struct{ CommentID int64 `json:"comment_id"` }
				Step        string
				Approvers   []string
				Publisher   string
				Permissions map[string]string
				Comments    []struct {
					ID        int64  `json:"id"`
					Body      string `json:"body"`
					Author    string `json:"author"`
					CreatedAt string `json:"created_at"`
					UpdatedAt string `json:"updated_at"`
					App       bool   `json:"app"`
				}
			}
			Expect struct {
				Accepted *int64
				Log      []struct {
					CommentID int64   `json:"comment_id"`
					Reason    *string `json:"reason"`
				}
			}
		}
	}
	load(t, "approvals.json", &doc)
	if len(doc.Cases) < 12 {
		t.Fatalf("only %d approval cases", len(doc.Cases))
	}
	for _, c := range doc.Cases {
		t.Run(c.Name, func(t *testing.T) {
			in := c.Input
			var cs []ghloop.Comment
			for _, x := range in.Comments {
				cs = append(cs, ghloop.Comment{ID: x.ID, Body: x.Body, Author: x.Author, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, App: x.App})
			}
			perm := func(u string) string {
				if p, ok := in.Permissions[u]; ok {
					return p
				}
				return "none"
			}
			acc, log := ghloop.Evaluate(cs, in.Decision.CommentID, in.Step, in.Approvers, in.Publisher, perm)
			var got *int64
			if acc != nil {
				got = &acc.ID
			}
			if (got == nil) != (c.Expect.Accepted == nil) || (got != nil && *got != *c.Expect.Accepted) {
				t.Fatalf("accepted %v, want %v", got, c.Expect.Accepted)
			}
			type row struct {
				ID     int64
				Reason string
			}
			var gl, wl []row
			for _, e := range log {
				gl = append(gl, row{e.CommentID, e.Reason})
			}
			for _, e := range c.Expect.Log {
				r := ""
				if e.Reason != nil {
					r = *e.Reason
				}
				wl = append(wl, row{e.CommentID, r})
			}
			if !reflect.DeepEqual(gl, wl) {
				t.Fatalf("log %v, want %v", gl, wl)
			}
		})
	}
}

func TestStatusPRVectors(t *testing.T) {
	var doc struct {
		Cases []struct {
			Name  string
			Input struct {
				State     string   `json:"state"`
				MergedBy  *string  `json:"merged_by"`
				BaseText  string   `json:"base_text"`
				HeadText  *string  `json:"head_text"`
				Changed   []string `json:"changed"`
				Item      string   `json:"item"`
				Approvers []string `json:"approvers"`
				Publisher string   `json:"publisher"`
			}
			Expect struct {
				OK     bool    `json:"ok"`
				Reason *string `json:"reason"`
			}
		}
	}
	load(t, "status_pr.json", &doc)
	if len(doc.Cases) < 7 {
		t.Fatalf("only %d status PR cases", len(doc.Cases))
	}
	for _, c := range doc.Cases {
		t.Run(c.Name, func(t *testing.T) {
			in := c.Input
			by, head := "", "\x00none"
			if in.MergedBy != nil {
				by = *in.MergedBy
			}
			if in.HeadText != nil {
				head = *in.HeadText
			}
			ok, reason := reconcile.CheckStatusPR(in.State, by, in.BaseText, head, in.Changed, in.Item, in.Approvers, in.Publisher)
			want := ""
			if c.Expect.Reason != nil {
				want = *c.Expect.Reason
			}
			if ok != c.Expect.OK || reason != want {
				t.Fatalf("got (%v, %q), want (%v, %q)", ok, reason, c.Expect.OK, want)
			}
		})
	}
}
