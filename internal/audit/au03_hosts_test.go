package audit

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/evidence"
)

// 037: AU03 checks each PR's URL against the run's Git host, and a recorded author id against the
// run's publisher id.
func TestAU03PerHost(t *testing.T) {
	withHost := func(kind, host string, urls func(n int) string, extra map[string]any) func(string, *[]map[string]any, *[]map[string]any) {
		return func(d string, _ *[]map[string]any, prs *[]map[string]any) {
			st := map[string]any{"run_id": "r", "kind": "item", "item": "002", "state": "succeeded",
				"target": map[string]any{"name": "g", "repo": "me/greeting-service"}, "scm": map[string]any{"kind": kind, "host": host}}
			for k, v := range extra {
				st[k] = v
			}
			if err := evidence.WriteJSON(filepath.Join(d, "run.json"), st); err != nil {
				t.Fatal(err)
			}
			for i := range *prs {
				(*prs)[i]["url"] = urls(5 + i)
			}
		}
	}
	bb := func(n int) string { return "https://bitbucket.org/me/greeting-service/pull-requests/" + itoa(n) }
	gl := func(n int) string { return "https://git.corp/me/greeting-service/-/merge_requests/" + itoa(n) }
	gh := func(n int) string { return "https://github.com/me/greeting-service/pull/" + itoa(n) }
	for name, c := range map[string]struct {
		mutate func(string, *[]map[string]any, *[]map[string]any)
		pass   bool
	}{
		"bitbucket URLs":           {withHost("bitbucket", "bitbucket.org", bb, nil), true},
		"github URL on bitbucket":  {withHost("bitbucket", "bitbucket.org", gh, nil), false},
		"self-managed gitlab URLs": {withHost("gitlab", "git.corp", gl, nil), true},
		"gitlab.com URL on git.corp": {withHost("gitlab", "git.corp", func(n int) string {
			return "https://gitlab.com/me/greeting-service/-/merge_requests/" + itoa(n)
		}, nil), false},
		"author id is the publisher": {func(d string, ev *[]map[string]any, prs *[]map[string]any) {
			withHost("bitbucket", "bitbucket.org", bb, map[string]any{"publisher_id": "{BOT}"})(d, ev, prs)
			for i := range *prs {
				(*prs)[i]["author_id"] = "{bot}"
			}
		}, true},
		"author id is someone else": {func(d string, ev *[]map[string]any, prs *[]map[string]any) {
			withHost("bitbucket", "bitbucket.org", bb, map[string]any{"publisher_id": "{bot}"})(d, ev, prs)
			(*prs)[0]["author_id"] = "{human}"
		}, false},
	} {
		t.Run(name, func(t *testing.T) {
			got := itemResults(t, itemRecord(t, c.mutate))["AU03"]
			if strings.HasPrefix(got, "pass") != c.pass {
				t.Fatalf("AU03 %s", got)
			}
		})
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
