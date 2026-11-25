package k8s

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func TestReplaceSecretFromEnvFileUpdatesInPlace(t *testing.T) {
	cs := fake.NewSimpleClientset()
	c := &Client{CS: cs}
	env := filepath.Join(t.TempDir(), "a.env")
	_ = os.WriteFile(env, []byte("PILOT_SCM_TOKEN=one\n"), 0o600)
	if err := c.ReplaceSecretFromEnvFile("pilot-publish", "scm-token", env); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(env, []byte("PILOT_SCM_TOKEN=two\n"), 0o600)
	cs.ClearActions()
	if err := c.ReplaceSecretFromEnvFile("pilot-publish", "scm-token", env); err != nil {
		t.Fatal(err)
	}
	for _, a := range cs.Actions() {
		if a.GetVerb() == "delete" || a.GetVerb() == "create" {
			t.Fatalf("unexpected %s: replace must update in place", a.GetVerb())
		}
	}
	s, _ := cs.CoreV1().Secrets("pilot-publish").Get(context.Background(), "scm-token", metav1.GetOptions{})
	if string(s.Data["PILOT_SCM_TOKEN"]) != "two" {
		t.Fatal(string(s.Data["PILOT_SCM_TOKEN"]))
	}
}

func TestCreateSecretWaitsForQuota(t *testing.T) {
	cs := fake.NewSimpleClientset()
	refusals := 2
	cs.PrependReactor("create", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
		if refusals > 0 {
			refusals--
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "scm-token",
				errors.New("status unknown for quota: publisher-quota, resources: count/secrets"))
		}
		return false, nil, nil
	})
	defer func(w, p time.Duration) { quotaWait, quotaPoll = w, p }(quotaWait, quotaPoll)
	quotaWait, quotaPoll = time.Second, time.Millisecond
	c := &Client{CS: cs}
	if err := c.createSecret("pilot-publish", "scm-token", map[string][]byte{"K": []byte("v")}); err != nil {
		t.Fatal(err)
	}
	if refusals != 0 {
		t.Fatal("did not retry")
	}
	// Other Forbidden errors fail at once.
	cs.PrependReactor("create", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "x", errors.New("RBAC: denied"))
	})
	if err := c.createSecret("pilot-publish", "x", nil); err == nil || !strings.Contains(err.Error(), "Forbidden") {
		t.Fatal(err)
	}
}

func TestReceiverReady(t *testing.T) {
	cs := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-abc", Namespace: "pilot-w-002-1", Labels: map[string]string{"job-name": "worker"}},
		Status: corev1.PodStatus{Phase: corev1.PodPending, InitContainerStatuses: []corev1.ContainerStatus{
			{Name: "receiver", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}},
	}, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "done-x", Namespace: "ended", Labels: map[string]string{"job-name": "worker"}},
		Status:     corev1.PodStatus{Phase: corev1.PodFailed},
	})
	c := &Client{CS: cs}
	if got := c.ReceiverReady("pilot-w-002-1", "worker", "receiver", 2*time.Second); got != "worker-abc" {
		t.Fatal(got)
	}
	if got := c.ReceiverReady("ended", "worker", "receiver", 2*time.Second); got != "" {
		t.Fatal("a pod that already ended can't receive")
	}
	if got := c.ReceiverReady("pilot-w-002-1", "worker", "other", 1500*time.Millisecond); got != "" {
		t.Fatal(got)
	}
}

// 031 T034: RestartDaemonSet stamps the pod template, like `kubectl rollout restart`.
func TestRestartDaemonSet(t *testing.T) {
	ds := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "calico-node", Namespace: "kube-system", Generation: 1},
		Status: appsv1.DaemonSetStatus{ObservedGeneration: 1, DesiredNumberScheduled: 1, UpdatedNumberScheduled: 1, NumberReady: 1}}
	c, cs := fakeClient(ds)
	if err := c.RestartDaemonSet("kube-system", "calico-node", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	got, err := cs.AppsV1().DaemonSets("kube-system").Get(context.Background(), "calico-node", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] == "" {
		t.Fatalf("no restartedAt annotation: %v", got.Spec.Template.Annotations)
	}
	if err := c.RestartDaemonSet("kube-system", "missing", time.Second); err == nil {
		t.Error("a missing daemonset: no error")
	}
}
