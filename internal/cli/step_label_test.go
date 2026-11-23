package cli

import (
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/ghloop"
)

// 027 FR-007: a profile on the default label is told when open issues still carry the old one.
func TestOldLabelNotice(t *testing.T) {
	issues := []ghloop.Issue{
		{Number: 3, State: "open", Labels: []string{"status:specced", "pilot/go"}},
		{Number: 4, State: "closed", Labels: []string{"pilot/go"}},
		{Number: 5, State: "open", Labels: []string{"chalkeus:proposed"}},
	}
	msg := oldLabelNotice(issues, "shop", "me/shop")
	for _, want := range []string{`"pilot/go" (#3)`, `"chalkeus:proposed"`, "FORGE_PROPOSE_LABEL=pilot/go in targets/shop.env"} {
		if !strings.Contains(msg, want) {
			t.Errorf("notice %q lacks %q", msg, want)
		}
	}
	if strings.Contains(msg, "#4") {
		t.Errorf("closed issue named: %q", msg)
	}
	if got := oldLabelNotice(issues[2:], "shop", "me/shop"); got != "" {
		t.Errorf("want no notice, got %q", got)
	}
}
