package secret

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestNeverPrinted(t *testing.T) {
	s := New("ghp_TOPSECRETVALUE1234567890")
	type holder struct {
		Token Value
		Name  string
	}
	h := holder{Token: s, Name: "x"}
	outs := []string{
		fmt.Sprint(s), fmt.Sprintf("%v %+v %#v %s %q %x", s, s, s, s, s, s),
		fmt.Sprintf("%v %+v %#v", h, h, h),
	}
	j, _ := json.Marshal(h)
	outs = append(outs, string(j))
	txt, _ := s.MarshalText()
	outs = append(outs, string(txt))
	for _, o := range outs {
		if strings.Contains(o, "TOPSECRET") {
			t.Fatalf("secret leaked: %s", o)
		}
	}
	if s.Reveal() != "ghp_TOPSECRETVALUE1234567890" || s.Empty() || !New("").Empty() {
		t.Fatal("Reveal/Empty")
	}
}
