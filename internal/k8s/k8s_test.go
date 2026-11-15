package k8s

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestRenderLikePythonTemplate(t *testing.T) {
	out, err := RenderString("image: ${IMG}\nname: $JOB\ncost: $$5\n", map[string]string{"IMG": "x@sha256:1", "JOB": "w"})
	if err != nil || out != "image: x@sha256:1\nname: w\ncost: $5\n" {
		t.Fatalf("%q %v", out, err)
	}
	if _, err := RenderString("a: ${MISSING}", nil); err == nil {
		t.Error("missing variable must fail")
	}
	if _, err := RenderString("a: $ b", nil); err == nil {
		t.Error("lone $ must fail")
	}
}

func TestRenderRealTemplates(t *testing.T) {
	m, _ := filepath.Glob("../../pilot/manifests/*.tmpl")
	if len(m) == 0 {
		t.Fatal("no manifests found")
	}
	for _, p := range m {
		raw, _ := os.ReadFile(p)
		vars := map[string]string{}
		for _, g := range tmplRE.FindAllStringSubmatch(string(raw), -1) {
			if n := g[2] + g[3]; n != "" {
				vars[n] = "v"
			}
		}
		if _, err := Render(p, vars); err != nil {
			t.Errorf("%s: %v", filepath.Base(p), err)
		}
	}
}

func TestParseEnvFile(t *testing.T) {
	d, err := ParseEnvFile([]byte("# c\n\n  A=1\nB=x=y\nC=\"q\"\n"))
	if err != nil || string(d["A"]) != "1" || string(d["B"]) != "x=y" || string(d["C"]) != `"q"` || len(d) != 3 {
		t.Fatalf("%v %v", d, err)
	}
	if _, err := ParseEnvFile([]byte("NOEQUALS\n")); err == nil {
		t.Error("line without = must fail")
	}
}

func TestSplitYAML(t *testing.T) {
	docs := splitYAML("a: 1\n---\nb: 2\n--- \n\n---\nc: 3\n")
	if len(docs) != 3 {
		t.Fatalf("%d docs: %q", len(docs), docs)
	}
}

func fakeClient(objs ...runtime.Object) (*Client, *fake.Clientset) {
	cs := fake.NewSimpleClientset(objs...)
	return &Client{CS: cs}, cs
}

func TestConfigMapBinaryData(t *testing.T) {
	c, cs := fakeClient()
	dir := t.TempDir()
	txt, bin := filepath.Join(dir, "pr.json"), filepath.Join(dir, "change.bundle")
	_ = os.WriteFile(txt, []byte(`{"a":1}`), 0o644)
	_ = os.WriteFile(bin, []byte{0xff, 0xfe, 0x00}, 0o644)
	if err := c.CreateConfigMap("ns", "cm", map[string]string{"pr.json": txt, "change.bundle": bin}, map[string]string{"x": "y"}); err != nil {
		t.Fatal(err)
	}
	cm, _ := cs.CoreV1().ConfigMaps("ns").Get(t.Context(), "cm", metav1.GetOptions{})
	if cm.Data["pr.json"] != `{"a":1}` || len(cm.BinaryData["change.bundle"]) != 3 || cm.Labels["x"] != "y" {
		t.Fatalf("%+v", cm)
	}
}

func TestSecretErrorsNeverEchoValues(t *testing.T) {
	c, cs := fakeClient()
	dir := t.TempDir()
	env := filepath.Join(dir, "publisher.env")
	_ = os.WriteFile(env, []byte("PILOT_SCM_TOKEN=supersecret123\n"), 0o600)
	if err := c.CreateSecretFromEnvFile("ns", "scm-token", env); err != nil {
		t.Fatal(err)
	}
	s, _ := cs.CoreV1().Secrets("ns").Get(t.Context(), "scm-token", metav1.GetOptions{})
	if string(s.Data["PILOT_SCM_TOKEN"]) != "supersecret123" {
		t.Fatal("secret data")
	}
	err := c.CreateSecretFromEnvFile("ns", "scm-token", env) // already exists
	if err == nil || strings.Contains(err.Error(), "supersecret") {
		t.Fatalf("error must not contain the value: %v", err)
	}
}

func TestCanI(t *testing.T) {
	c, cs := fakeClient()
	cs.PrependReactor("create", "subjectaccessreviews", func(a k8stesting.Action) (bool, runtime.Object, error) {
		sar := a.(k8stesting.CreateAction).GetObject().(*authv1.SubjectAccessReview)
		allowed := sar.Spec.ResourceAttributes.Resource == "configmaps" && sar.Spec.User == "system:serviceaccount:pilot-w:worker"
		hasGroup := false
		for _, g := range sar.Spec.Groups {
			hasGroup = hasGroup || g == "system:serviceaccounts:pilot-w"
		}
		sar.Status.Allowed = allowed && hasGroup
		return true, sar, nil
	})
	// No mapper here: CanI falls back to the core group.
	if got := c.CanI("get", "configmaps", "g", "system:serviceaccount:pilot-w:worker"); got != "yes" {
		t.Errorf("got %s", got)
	}
	if got := c.CanI("create", "pods/exec", "g", "system:serviceaccount:pilot-w:worker"); got != "no" {
		t.Errorf("got %s", got)
	}
}

func TestPullErrorsAndJobs(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "ns", Labels: map[string]string{"job-name": "w"}},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Image: "img@sha256:x",
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}}}}}
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "ns"},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: "Failed", Status: corev1.ConditionTrue, Reason: "DeadlineExceeded"}}}}
	c, _ := fakeClient(pod, job)
	if got := c.PullErrors("ns"); len(got) != 1 || got[0] != "p1: img@sha256:x (ImagePullBackOff)" {
		t.Errorf("%v", got)
	}
	if got := c.WaitJob("ns", "w", 5*time.Second); got != "DeadlineExceeded" {
		t.Errorf("%s", got)
	}
	if got := c.JobPod("ns", "w"); got != "p1" {
		t.Errorf("%s", got)
	}
	if err := c.DeleteJob("ns", "w", true); err != nil {
		t.Error(err)
	}
	if err := c.DeleteJob("ns", "absent", true); err != nil {
		t.Error("deleting a missing job is not an error")
	}
}
