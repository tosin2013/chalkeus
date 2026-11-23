package checks

import (
	"strings"
	"testing"
)

func TestAuthorRules(t *testing.T) {
	if ok, _ := AuthorPodRule(1, 1, false, false); !ok {
		t.Fatal("A01 pass")
	}
	for _, c := range []struct {
		pods, containers int
		sa, pod          any
	}{{2, 1, false, false}, {1, 2, false, false}, {1, 1, nil, false}, {1, 1, false, true}} {
		if ok, _ := AuthorPodRule(c.pods, c.containers, c.sa, c.pod); ok {
			t.Fatalf("A01 must fail: %+v", c)
		}
	}
	if ok, _ := AuthorSecretsRule([]string{"model-key"}, []string{"PILOT_MODEL", "PILOT_TOKEN_CAP", "HTTPS_PROXY"}); !ok {
		t.Fatal("A02 pass")
	}
	if ok, d := AuthorSecretsRule([]string{"model-key", "scm-token"}, nil); ok {
		t.Fatal("A02: a second secret fails: " + d)
	}
	if ok, d := AuthorSecretsRule([]string{"model-key"}, []string{"GITHUB_TOKEN"}); ok || !strings.Contains(d, "GITHUB_TOKEN") {
		t.Fatal("A02: an SCM-like env fails: " + d)
	}
	ingress := func(sel map[string]any) map[string]any {
		return map[string]any{"spec": map[string]any{"ingress": []any{map[string]any{"from": []any{map[string]any{"namespaceSelector": map[string]any{"matchLabels": sel}}}}}}}
	}
	if ok, _ := AuthorProxyIngressRule(ingress(map[string]any{"chalkeus.io/role": "author"})); !ok {
		t.Fatal("A05 pass")
	}
	if ok, _ := AuthorProxyIngressRule(ingress(map[string]any{"chalkeus.io/role": "worker"})); ok {
		t.Fatal("A05: a worker role fails")
	}
	if ok, _ := AuthorProxyIngressRule(ingress(map[string]any{"kubernetes.io/metadata.name": "x"})); ok {
		t.Fatal("A05: any other selector fails")
	}
	if ok, _ := AuthorProxyIngressRule(nil); ok {
		t.Fatal("A05: no policy fails")
	}
}
