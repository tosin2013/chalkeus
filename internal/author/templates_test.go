package author

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tosin2013/chalkeus/internal/k8s"
	"sigs.k8s.io/yaml"
)

func renderDocs(t *testing.T, name string, vars map[string]string) []map[string]any {
	t.Helper()
	root, _ := filepath.Abs("../..")
	y, err := k8s.Render(filepath.Join(root, "pilot", "manifests", name), vars)
	if err != nil {
		t.Fatal(err)
	}
	var docs []map[string]any
	for _, d := range strings.Split(y, "\n---\n") {
		var o map[string]any
		if err := yaml.Unmarshal([]byte(d), &o); err != nil {
			t.Fatalf("%s: %v\n%s", name, err, d)
		}
		if len(o) > 0 {
			docs = append(docs, o)
		}
	}
	return docs
}

func mm(v any) map[string]any { x, _ := v.(map[string]any); return x }
func ll(v any) []any          { x, _ := v.([]any); return x }

// 024 R6: the author namespace has role author, one pod, no token, and egress only to DNS and
// the author proxy.
func TestAuthorNamespaceTemplate(t *testing.T) {
	docs := renderDocs(t, "author-ns.yaml.tmpl", map[string]string{"NS": "pilot-a-stockroom-009", "DRAFT_ID": "20261119T000000Z"})
	kinds := map[string]map[string]any{}
	for _, d := range docs {
		kinds[d["kind"].(string)+"/"+mm(d["metadata"])["name"].(string)] = d
	}
	ns := kinds["Namespace/pilot-a-stockroom-009"]
	if mm(mm(ns["metadata"])["labels"])["specfoundry.io/role"] != "author" {
		t.Fatal("namespace role must be author")
	}
	if kinds["ServiceAccount/author"]["automountServiceAccountToken"] != false {
		t.Fatal("no service-account token")
	}
	if mm(mm(kinds["ResourceQuota/author-quota"]["spec"])["hard"])["pods"] != "1" {
		t.Fatal("one pod")
	}
	if _, ok := kinds["NetworkPolicy/default-deny-all"]; !ok {
		t.Fatal("default deny")
	}
	var targets []string
	for _, rule := range ll(mm(kinds["NetworkPolicy/allow-egress-dns-and-author-proxy"]["spec"])["egress"]) {
		for _, to := range ll(mm(rule)["to"]) {
			targets = append(targets, mm(mm(mm(to)["namespaceSelector"])["matchLabels"])["kubernetes.io/metadata.name"].(string))
		}
	}
	if strings.Join(targets, ",") != "kube-system,pilot-egress-a" {
		t.Fatalf("egress must reach only DNS and pilot-egress-a, got %v", targets)
	}
}

// 024 R3: the author Job runs the worker image with only the model-key secret, through the
// author proxy, with no token.
func TestAuthorJobTemplate(t *testing.T) {
	docs := renderDocs(t, "author-job.yaml.tmpl", map[string]string{"NS": "pilot-a-x-001", "DEADLINE": "1200",
		"TOOLS_IMAGE": "t@sha256:1", "WORKER_IMAGE": "w@sha256:2", "PILOT_MODEL": "p/m", "SPEND_CAP": "1.0",
		"TOKEN_CAP": "2000000", "INPUT_LIMIT": "512Mi"})
	if len(docs) != 1 || docs[0]["kind"] != "Job" {
		t.Fatalf("one Job, got %v", docs)
	}
	pod := mm(mm(mm(docs[0]["spec"])["template"])["spec"])
	if pod["automountServiceAccountToken"] != false || pod["serviceAccountName"] != "author" {
		t.Fatal("author SA without token")
	}
	var secrets []string
	for _, v := range ll(pod["volumes"]) {
		if s := mm(mm(v)["secret"]); s != nil {
			secrets = append(secrets, s["secretName"].(string))
		}
	}
	if strings.Join(secrets, ",") != "model-key" {
		t.Fatalf("only the model key, got %v", secrets)
	}
	c := mm(ll(pod["containers"])[0])
	if c["image"] != "w@sha256:2" {
		t.Fatal("worker image")
	}
	for _, e := range ll(c["env"]) {
		if n := mm(e)["name"]; n == "HTTPS_PROXY" && mm(e)["value"] != "http://proxy.pilot-egress-a.svc.cluster.local:8888" {
			t.Fatal("the author proxy")
		}
		if n := mm(e)["name"].(string); strings.Contains(n, "TOKEN") && n != "PILOT_TOKEN_CAP" || strings.Contains(n, "GITHUB") {
			t.Fatalf("no SCM-like env: %s", n)
		}
	}
}

// The isolation checks find the author's pod by this label (checks.RunAuthorChecks).
func TestAuthorJobPodLabel(t *testing.T) {
	docs := renderDocs(t, "author-job.yaml.tmpl", map[string]string{"NS": "n", "DEADLINE": "1", "TOOLS_IMAGE": "t",
		"WORKER_IMAGE": "w", "PILOT_MODEL": "m", "SPEND_CAP": "1", "TOKEN_CAP": "1", "INPUT_LIMIT": "1Mi"})
	if mm(mm(mm(mm(docs[0]["spec"])["template"])["metadata"])["labels"])["app"] != "author" {
		t.Fatal("the author pod must carry app=author")
	}
}
