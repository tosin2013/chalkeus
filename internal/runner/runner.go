// Package runner runs pilot units (worker, grader, publisher) as plain Kubernetes Jobs
// (reference: pilot/specfoundry_pilot/runner/base.py and jobs.py).
package runner

import (
	"time"

	"github.com/tosin2013/specfoundry/internal/k8s"
)

// Handle identifies a submitted unit.
type Handle struct {
	Kind, Namespace, Name string
	OwnNamespace          bool // Delete removes the whole namespace
}

// Runner is the backend interface (constitution VI, PRD F10).
type Runner interface {
	Submit(kind, namespace, name, manifest string, ownNamespace bool) (Handle, error)
	Wait(h Handle, timeout time.Duration) string // Complete | Failed | DeadlineExceeded | Timeout
	Collect(h Handle) string                     // the unit's stdout (where the result frame lives)
	Status(h Handle) map[string]any
	Delete(h Handle)
}

// Jobs is the plain Kubernetes Jobs backend.
type Jobs struct{ C *k8s.Client }

func (j Jobs) Submit(kind, ns, name, manifest string, own bool) (Handle, error) {
	return Handle{Kind: kind, Namespace: ns, Name: name, OwnNamespace: own}, j.C.Apply(manifest)
}

func (j Jobs) Wait(h Handle, timeout time.Duration) string {
	return j.C.WaitJob(h.Namespace, h.Name, timeout)
}

func (j Jobs) Collect(h Handle) string { return j.C.Logs(h.Namespace, h.Name, "") }

func (j Jobs) Status(h Handle) map[string]any {
	job, _ := j.C.Get(h.Namespace, "job", h.Name)
	pod := j.C.JobPod(h.Namespace, h.Name)
	var pj map[string]any
	if pod != "" {
		pj, _ = j.C.Get(h.Namespace, "pod", pod)
	}
	st, _ := pj["status"].(map[string]any)
	var term any
	if css, ok := st["containerStatuses"].([]any); ok {
		for _, x := range css {
			cs, _ := x.(map[string]any)
			state, _ := cs["state"].(map[string]any)
			if t, ok := state["terminated"]; ok && t != nil {
				term = t
			}
		}
	}
	var podV any
	if pod != "" {
		podV = pod
	}
	return map[string]any{"job_status": job["status"], "pod": podV, "pod_phase": st["phase"], "pod_reason": st["reason"],
		"container_terminated": term}
}

func (j Jobs) Delete(h Handle) {
	if h.OwnNamespace {
		j.C.DeleteNS(h.Namespace, true)
		return
	}
	_ = j.C.DeleteJob(h.Namespace, h.Name, true)
}
