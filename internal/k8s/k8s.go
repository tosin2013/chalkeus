// Package k8s talks to the pilot cluster through client-go (016 research R3), replacing the
// reference's kubectl subprocesses (pilot/specfoundry_pilot/k8s.py) with the same semantics.
package k8s

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/yaml"
)

// FieldManager names the pilot in server-side apply.
const FieldManager = "specfoundry-pilot"

// Client is a connection to one kubeconfig context.
type Client struct {
	CS     kubernetes.Interface
	Dyn    dynamic.Interface
	Mapper meta.RESTMapper
	cfg    *rest.Config
}

// New connects using the default kubeconfig loading rules and the given context.
func New(kubeContext string) (*Client, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules,
		&clientcmd.ConfigOverrides{CurrentContext: kubeContext}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("kubeconfig context %s: %w", kubeContext, err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(cs.Discovery()))
	return &Client{CS: cs, Dyn: dyn, Mapper: mapper, cfg: cfg}, nil
}

func ctxT(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// ---------------------------------------------------------------- templates

var tmplRE = regexp.MustCompile(`(?i)\$(?:(\$)|([_a-z][_a-z0-9]*)|\{([_a-z][_a-z0-9]*)\}|())`)

// Render substitutes $VAR and ${VAR} like Python's string.Template.substitute: a missing
// variable or a lone "$" is an error; "$$" is a literal "$".
func Render(templatePath string, vars map[string]string) (string, error) {
	raw, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	return RenderString(string(raw), vars)
}

// RenderString is Render on a string.
func RenderString(text string, vars map[string]string) (string, error) {
	var firstErr error
	out := tmplRE.ReplaceAllStringFunc(text, func(m string) string {
		g := tmplRE.FindStringSubmatch(m)
		switch {
		case g[1] != "":
			return "$"
		case g[2] != "" || g[3] != "":
			name := g[2] + g[3]
			v, ok := vars[name]
			if !ok && firstErr == nil {
				firstErr = fmt.Errorf("template variable %s is not set", name)
			}
			return v
		default:
			if firstErr == nil {
				firstErr = errors.New("invalid placeholder in template")
			}
			return m
		}
	})
	return out, firstErr
}

// ---------------------------------------------------------------- generic reads and apply

var aliases = map[string]string{
	"sa": "serviceaccounts", "netpol": "networkpolicies", "resourcequota": "resourcequotas",
	"job": "jobs", "pod": "pods", "secret": "secrets", "configmap": "configmaps", "deploy": "deployments",
	"deployment": "deployments", "ns": "namespaces", "namespace": "namespaces",
}

func (c *Client) gvr(resource string) (schema.GroupVersionResource, bool, error) {
	if a, ok := aliases[resource]; ok {
		resource = a
	}
	gvr, err := c.Mapper.ResourceFor(schema.GroupVersionResource{Resource: resource})
	if err != nil {
		return gvr, false, err
	}
	gvk, err := c.Mapper.KindFor(gvr)
	if err != nil {
		return gvr, false, err
	}
	m, err := c.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return gvr, false, err
	}
	return gvr, m.Scope.Name() == meta.RESTScopeNameNamespace, nil
}

// Get returns one object as a map (nil, nil when it does not exist): `kubectl get -o json`.
func (c *Client) Get(ns, resource, name string) (map[string]any, error) {
	gvr, namespaced, err := c.gvr(resource)
	if err != nil {
		return nil, err
	}
	ctx, cancel := ctxT(60 * time.Second)
	defer cancel()
	var obj *unstructured.Unstructured
	if namespaced {
		obj, err = c.Dyn.Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = c.Dyn.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return obj.Object, nil
}

// List returns the items of a resource; ns "" lists all namespaces; selector may be empty.
func (c *Client) List(ns, resource, selector string) ([]map[string]any, error) {
	gvr, namespaced, err := c.gvr(resource)
	if err != nil {
		return nil, err
	}
	ctx, cancel := ctxT(60 * time.Second)
	defer cancel()
	opts := metav1.ListOptions{LabelSelector: selector}
	var l *unstructured.UnstructuredList
	if namespaced && ns != "" {
		l, err = c.Dyn.Resource(gvr).Namespace(ns).List(ctx, opts)
	} else {
		l, err = c.Dyn.Resource(gvr).List(ctx, opts)
	}
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(l.Items))
	for _, it := range l.Items {
		out = append(out, it.Object)
	}
	return out, nil
}

// Apply server-side-applies every document in a multi-document YAML (`kubectl apply -f -`).
// Namespaced objects without a namespace go to "default", as kubectl does.
func (c *Client) Apply(yamlText string) error {
	for _, doc := range splitYAML(yamlText) {
		var obj map[string]any
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			return fmt.Errorf("apply: %w", err)
		}
		if len(obj) == 0 {
			continue
		}
		u := &unstructured.Unstructured{Object: obj}
		if u.GetKind() == "List" {
			items, _ := obj["items"].([]any)
			for _, it := range items {
				m, _ := it.(map[string]any)
				if err := c.applyOne(&unstructured.Unstructured{Object: m}); err != nil {
					return err
				}
			}
			continue
		}
		if err := c.applyOne(u); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) applyOne(u *unstructured.Unstructured) error {
	gvk := u.GroupVersionKind()
	m, err := c.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		if r, ok := c.Mapper.(interface{ Reset() }); ok { // a CRD applied earlier in the same call
			r.Reset()
			m, err = c.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		}
		if err != nil {
			return fmt.Errorf("apply %s %s: %w", gvk.Kind, u.GetName(), err)
		}
	}
	ctx, cancel := ctxT(120 * time.Second)
	defer cancel()
	raw, err := u.MarshalJSON()
	if err != nil {
		return err
	}
	opts := metav1.PatchOptions{FieldManager: FieldManager, Force: ptr(true)}
	var ri dynamic.ResourceInterface = c.Dyn.Resource(m.Resource)
	if m.Scope.Name() == meta.RESTScopeNameNamespace {
		ns := u.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		ri = c.Dyn.Resource(m.Resource).Namespace(ns)
	}
	if _, err := ri.Patch(ctx, u.GetName(), types.ApplyPatchType, raw, opts); err != nil {
		return fmt.Errorf("apply %s %s: %w", gvk.Kind, u.GetName(), err)
	}
	return nil
}

func ptr[T any](v T) *T { return &v }

func splitYAML(text string) []string {
	var docs []string
	var cur strings.Builder
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimRight(line, " \t") == "---" || strings.HasPrefix(line, "--- ") {
			docs = append(docs, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteString(line + "\n")
	}
	docs = append(docs, cur.String())
	var out []string
	for _, d := range docs {
		if strings.TrimSpace(d) != "" {
			out = append(out, d)
		}
	}
	return out
}

// ---------------------------------------------------------------- jobs, pods, logs

// WaitJob returns Complete | Failed | DeadlineExceeded | Timeout, polling every 3 s.
func (c *Client) WaitJob(ns, name string, timeout time.Duration) string {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		ctx, cancel := ctxT(30 * time.Second)
		job, err := c.CS.BatchV1().Jobs(ns).Get(ctx, name, metav1.GetOptions{})
		cancel()
		if err == nil {
			for _, cnd := range job.Status.Conditions {
				if cnd.Status != corev1.ConditionTrue {
					continue
				}
				switch string(cnd.Type) {
				case "Complete":
					return "Complete"
				case "Failed":
					if cnd.Reason == "DeadlineExceeded" {
						return "DeadlineExceeded"
					}
					return "Failed"
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
	return "Timeout"
}

// JobPod returns the name of the first pod of a job ("" if none).
func (c *Client) JobPod(ns, job string) string {
	ctx, cancel := ctxT(30 * time.Second)
	defer cancel()
	pods, err := c.CS.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + job})
	if err != nil || len(pods.Items) == 0 {
		return ""
	}
	return pods.Items[0].Name
}

// PodLogs returns a pod's logs ("" on error); container and since are optional.
func (c *Client) PodLogs(ns, pod, container string, since *time.Time) string {
	opts := &corev1.PodLogOptions{Container: container}
	if since != nil {
		t := metav1.NewTime(*since)
		opts.SinceTime = &t
	}
	ctx, cancel := ctxT(60 * time.Second)
	defer cancel()
	rc, err := c.CS.CoreV1().Pods(ns).GetLogs(pod, opts).Stream(ctx)
	if err != nil {
		return ""
	}
	defer func() { _ = rc.Close() }()
	raw, _ := io.ReadAll(rc)
	return string(raw)
}

// Logs returns the logs of a job's pod (the reference k8s.logs).
func (c *Client) Logs(ns, job, container string) string {
	pod := c.JobPod(ns, job)
	if pod == "" {
		return ""
	}
	return c.PodLogs(ns, pod, container, nil)
}

// DeploymentLogs returns the logs of the first pod of a deployment since a time
// (`kubectl logs deploy/<name> --since-time=…`).
func (c *Client) DeploymentLogs(ns, name string, since *time.Time) string {
	ctx, cancel := ctxT(30 * time.Second)
	d, err := c.CS.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	cancel()
	if err != nil || d.Spec.Selector == nil {
		return ""
	}
	sel, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return ""
	}
	ctx, cancel = ctxT(30 * time.Second)
	pods, err := c.CS.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
	cancel()
	if err != nil || len(pods.Items) == 0 {
		return ""
	}
	sort.Slice(pods.Items, func(i, j int) bool { return pods.Items[i].Name < pods.Items[j].Name })
	return c.PodLogs(ns, pods.Items[0].Name, "", since)
}

// RolloutStatus waits until a deployment has all replicas updated and available.
func (c *Client) RolloutStatus(ns, name string, timeout time.Duration) bool {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		ctx, cancel := ctxT(30 * time.Second)
		d, err := c.CS.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		cancel()
		if err == nil && d.Status.ObservedGeneration >= d.Generation {
			want := int32(1)
			if d.Spec.Replicas != nil {
				want = *d.Spec.Replicas
			}
			if d.Status.UpdatedReplicas == want && d.Status.AvailableReplicas == want && d.Status.Replicas == want {
				return true
			}
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// WaitPodReady waits for a pod's Ready condition.
func (c *Client) WaitPodReady(ns, pod string, timeout time.Duration) bool {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		ctx, cancel := ctxT(30 * time.Second)
		p, err := c.CS.CoreV1().Pods(ns).Get(ctx, pod, metav1.GetOptions{})
		cancel()
		if err == nil {
			for _, cnd := range p.Status.Conditions {
				if cnd.Type == corev1.PodReady && cnd.Status == corev1.ConditionTrue {
					return true
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

// PullErrors lists "pod: image (reason)" for containers stuck on an image pull.
func (c *Client) PullErrors(ns string) []string {
	ctx, cancel := ctxT(30 * time.Second)
	defer cancel()
	pods, err := c.CS.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range pods.Items {
		for _, cs := range append(append([]corev1.ContainerStatus{}, p.Status.ContainerStatuses...), p.Status.InitContainerStatuses...) {
			if w := cs.State.Waiting; w != nil {
				switch w.Reason {
				case "ImagePullBackOff", "ErrImagePull", "InvalidImageName", "ErrImageNeverPull":
					out = append(out, fmt.Sprintf("%s: %s (%s)", p.Name, cs.Image, w.Reason))
				}
			}
		}
	}
	return out
}

// ---------------------------------------------------------------- deletes

func ignoreNotFound(err error) error {
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// DeleteNS deletes a namespace and, when wait is set, waits until it is gone (up to 300 s).
func (c *Client) DeleteNS(ns string, wait bool) {
	ctx, cancel := ctxT(60 * time.Second)
	_ = ignoreNotFound(c.CS.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{}))
	cancel()
	if !wait {
		return
	}
	end := time.Now().Add(300 * time.Second)
	for time.Now().Before(end) && c.NSExists(ns) {
		time.Sleep(2 * time.Second)
	}
}

// NSExists reports whether a namespace exists.
func (c *Client) NSExists(ns string) bool {
	ctx, cancel := ctxT(30 * time.Second)
	defer cancel()
	_, err := c.CS.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	return err == nil
}

// DeleteJob deletes a job and its pods; with wait, it waits until the job is gone.
func (c *Client) DeleteJob(ns, name string, wait bool) error {
	ctx, cancel := ctxT(60 * time.Second)
	bg := metav1.DeletePropagationBackground // kubectl's default cascade
	err := ignoreNotFound(c.CS.BatchV1().Jobs(ns).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &bg}))
	cancel()
	if err != nil || !wait {
		return err
	}
	end := time.Now().Add(120 * time.Second)
	for time.Now().Before(end) {
		ctx, cancel := ctxT(30 * time.Second)
		_, gerr := c.CS.BatchV1().Jobs(ns).Get(ctx, name, metav1.GetOptions{})
		cancel()
		if apierrors.IsNotFound(gerr) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("job %s/%s still present after delete", ns, name)
}

// DeleteSecret deletes a secret if it exists.
func (c *Client) DeleteSecret(ns, name string) error {
	ctx, cancel := ctxT(30 * time.Second)
	defer cancel()
	return ignoreNotFound(c.CS.CoreV1().Secrets(ns).Delete(ctx, name, metav1.DeleteOptions{}))
}

// DeleteConfigMap deletes a configmap if it exists.
func (c *Client) DeleteConfigMap(ns, name string) error {
	ctx, cancel := ctxT(30 * time.Second)
	defer cancel()
	return ignoreNotFound(c.CS.CoreV1().ConfigMaps(ns).Delete(ctx, name, metav1.DeleteOptions{}))
}

// ---------------------------------------------------------------- secrets and configmaps

// ParseEnvFile reads KEY=VALUE pairs the way `kubectl create secret --from-env-file` does:
// leading whitespace trimmed, blank and # lines skipped, the value kept verbatim.
func ParseEnvFile(raw []byte) (map[string][]byte, error) {
	out := map[string][]byte{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimLeft(sc.Text(), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("env file line %d: expected KEY=VALUE", n)
		}
		out[k] = []byte(v)
	}
	return out, sc.Err()
}

// CreateSecretFromEnvFile creates a generic secret from an env file (values read in-process
// and sent only to the API server; never logged: 016 research R7).
func (c *Client) CreateSecretFromEnvFile(ns, name, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data, err := ParseEnvFile(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return c.createSecret(ns, name, data)
}

// CreateSecretFromFiles creates a generic secret with one key per file (key = base name).
func (c *Client) CreateSecretFromFiles(ns, name string, files []string) error {
	data := map[string][]byte{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		data[filepath.Base(f)] = raw
	}
	return c.createSecret(ns, name, data)
}

func (c *Client) createSecret(ns, name string, data map[string][]byte) error {
	ctx, cancel := ctxT(30 * time.Second)
	defer cancel()
	_, err := c.CS.CoreV1().Secrets(ns).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}, Type: corev1.SecretTypeOpaque, Data: data},
		metav1.CreateOptions{FieldManager: FieldManager})
	if err != nil {
		return fmt.Errorf("create secret %s/%s: %s", ns, name, apiReason(err)) // never echo values
	}
	return nil
}

// CreateConfigMap creates a configmap from files (key → path); non-UTF-8 content goes to
// binaryData, as `kubectl create configmap --from-file` does.
func (c *Client) CreateConfigMap(ns, name string, files map[string]string, labels map[string]string) error {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
		Data: map[string]string{}, BinaryData: map[string][]byte{}}
	for k, p := range files {
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if utf8.Valid(raw) {
			cm.Data[k] = string(raw)
		} else {
			cm.BinaryData[k] = raw
		}
	}
	ctx, cancel := ctxT(30 * time.Second)
	defer cancel()
	if _, err := c.CS.CoreV1().ConfigMaps(ns).Create(ctx, cm, metav1.CreateOptions{FieldManager: FieldManager}); err != nil {
		return fmt.Errorf("create configmap %s/%s: %w", ns, name, err)
	}
	return nil
}

func apiReason(err error) string {
	var st apierrors.APIStatus
	if errors.As(err, &st) {
		return string(st.Status().Reason)
	}
	return "error"
}

// ---------------------------------------------------------------- authorization and exec

// CanI answers "yes" or "no" for a service account, like `kubectl auth can-i --as=<sa>`.
// asSA is "system:serviceaccount:<ns>:<name>"; resource may be "pods/exec".
func (c *Client) CanI(verb, resource, ns, asSA string) string {
	res, sub, _ := strings.Cut(resource, "/")
	groups := []string{"system:serviceaccounts", "system:authenticated"}
	if parts := strings.Split(asSA, ":"); len(parts) == 4 {
		groups = append(groups, "system:serviceaccounts:"+parts[2])
	}
	group := ""
	if c.Mapper != nil {
		if gvr, _, err := c.gvr(res); err == nil {
			group = gvr.Group
		}
	}
	return c.sar(verb, res, sub, ns, asSA, group, groups)
}

func (c *Client) sar(verb, res, sub, ns, asSA, group string, groups []string) string {
	sar := &authv1.SubjectAccessReview{Spec: authv1.SubjectAccessReviewSpec{
		User: asSA, Groups: groups,
		ResourceAttributes: &authv1.ResourceAttributes{Namespace: ns, Verb: verb, Group: group, Resource: res, Subresource: sub},
	}}
	ctx, cancel := ctxT(30 * time.Second)
	defer cancel()
	out, err := c.CS.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return "error: " + apiReason(err)
	}
	if out.Status.Allowed {
		return "yes"
	}
	return "no"
}

// ExecResult is the outcome of a command run in a pod.
type ExecResult struct {
	Code           int
	Stdout, Stderr string
}

// Exec runs a command in a pod's first container (`kubectl exec <pod> -- cmd…`).
func (c *Client) Exec(ns, pod string, timeout time.Duration, cmd ...string) ExecResult {
	req := c.CS.CoreV1().RESTClient().Post().Resource("pods").Name(pod).Namespace(ns).SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{Command: cmd, Stdout: true, Stderr: true}, scheme.ParameterCodec)
	ex, err := remotecommand.NewSPDYExecutor(c.cfg, "POST", req.URL())
	if err != nil {
		return ExecResult{Code: 1, Stderr: err.Error()}
	}
	ctx, cancel := ctxT(timeout)
	defer cancel()
	var so, se bytes.Buffer
	err = ex.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &so, Stderr: &se})
	code := 0
	if err != nil {
		code = 1
		var ce interface{ ExitStatus() int }
		if errors.As(err, &ce) {
			code = ce.ExitStatus()
		}
		if se.Len() == 0 {
			se.WriteString(err.Error())
		}
	}
	return ExecResult{Code: code, Stdout: so.String(), Stderr: se.String()}
}

// DaemonSetReady waits until a daemonset has all pods updated and ready
// (`kubectl rollout status ds/<name>`).
func (c *Client) DaemonSetReady(ns, name string, timeout time.Duration) bool {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		ctx, cancel := ctxT(30 * time.Second)
		ds, err := c.CS.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
		cancel()
		if err == nil && ds.Status.ObservedGeneration >= ds.Generation && ds.Status.DesiredNumberScheduled > 0 &&
			ds.Status.UpdatedNumberScheduled == ds.Status.DesiredNumberScheduled &&
			ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
			return true
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// NodesReady waits until every node reports Ready (`kubectl wait --for=condition=Ready node --all`).
func (c *Client) NodesReady(timeout time.Duration) bool {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		ctx, cancel := ctxT(30 * time.Second)
		nodes, err := c.CS.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		cancel()
		if err == nil && len(nodes.Items) > 0 {
			all := true
			for _, n := range nodes.Items {
				ready := false
				for _, cnd := range n.Status.Conditions {
					if cnd.Type == corev1.NodeReady && cnd.Status == corev1.ConditionTrue {
						ready = true
					}
				}
				all = all && ready
			}
			if all {
				return true
			}
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

// ---------------------------------------------------------------- diagnostics (016 FR-018)

// PodState summarises a pod for verbose output: its phase and each container's state.
func (c *Client) PodState(ns, pod string) string {
	ctx, cancel := ctxT(15 * time.Second)
	defer cancel()
	p, err := c.CS.CoreV1().Pods(ns).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return ""
	}
	parts := []string{string(p.Status.Phase)}
	for _, cs := range append(p.Status.InitContainerStatuses, p.Status.ContainerStatuses...) {
		switch {
		case cs.State.Waiting != nil:
			parts = append(parts, fmt.Sprintf("%s waiting (%s)", cs.Name, cs.State.Waiting.Reason))
		case cs.State.Running != nil:
			parts = append(parts, cs.Name+" running")
		case cs.State.Terminated != nil:
			parts = append(parts, fmt.Sprintf("%s exited %d (%s)", cs.Name, cs.State.Terminated.ExitCode, cs.State.Terminated.Reason))
		}
	}
	return strings.Join(parts, ", ")
}

// PodEvent is one Kubernetes event about a pod.
type PodEvent struct{ Key, Text string }

// PodEvents lists the events whose involved object is the pod, oldest first.
func (c *Client) PodEvents(ns, pod string) []PodEvent {
	ctx, cancel := ctxT(15 * time.Second)
	defer cancel()
	evs, err := c.CS.CoreV1().Events(ns).List(ctx, metav1.ListOptions{FieldSelector: "involvedObject.name=" + pod})
	if err != nil {
		return nil
	}
	sort.SliceStable(evs.Items, func(i, j int) bool { return evs.Items[i].LastTimestamp.Before(&evs.Items[j].LastTimestamp) })
	var out []PodEvent
	for _, e := range evs.Items {
		out = append(out, PodEvent{Key: fmt.Sprintf("%s/%d", e.UID, e.Count),
			Text: fmt.Sprintf("%s %s: %s", e.Type, e.Reason, e.Message)})
	}
	return out
}

// ExecStream runs a command in a pod's first container and copies its stdout to w until the
// command ends or ctx is cancelled.
func (c *Client) ExecStream(ctx context.Context, ns, pod string, w io.Writer, cmd ...string) error {
	req := c.CS.CoreV1().RESTClient().Post().Resource("pods").Name(pod).Namespace(ns).SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{Command: cmd, Stdout: true, Stderr: true}, scheme.ParameterCodec)
	ex, err := remotecommand.NewSPDYExecutor(c.cfg, "POST", req.URL())
	if err != nil {
		return err
	}
	return ex.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: w, Stderr: io.Discard})
}
