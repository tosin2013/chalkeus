package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// 027 US1a: every command is classified, and every ask command has a harness rule for each
// command form, so the harness (not only the skill's text) asks the human first.
func TestCommandAudit(t *testing.T) {
	forge := Forge(NewRoot("test"))
	if forge == nil {
		t.Fatal("no forge command under the chalkeus root")
	}
	seen := map[string]bool{}
	for _, c := range forge.Commands() {
		name := c.Name()
		if name == "help" || name == "completion" {
			continue
		}
		seen[name] = true
		if _, ok := CommandAudit[name]; !ok {
			t.Errorf("command %q is not in CommandAudit: classify it as ask or free (specs/027-chalkeus-rename/contracts/command-audit.md)", name)
		}
	}
	for name := range CommandAudit {
		if !seen[name] {
			t.Errorf("CommandAudit lists %q, which is not a command", name)
		}
	}

	raw, err := os.ReadFile(filepath.Join("..", "..", ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Permissions struct {
			Ask []string `json:"ask"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	rules := map[string]bool{}
	for _, r := range s.Permissions.Ask {
		rules[r] = true
	}
	var ask []string
	for name, class := range CommandAudit {
		if class == ClassAsk {
			ask = append(ask, name)
		}
	}
	sort.Strings(ask)
	for _, name := range ask {
		for _, p := range AskRulePrefixes {
			if want := "Bash(" + p + " " + name + ":*)"; !rules[want] {
				t.Errorf("ask command %q has no harness rule %s in .claude/settings.json", name, want)
			}
		}
	}
	if len(ask) != 10 {
		t.Errorf("want the 10 ask commands of the 027 audit, got %d: %v", len(ask), ask)
	}
}
