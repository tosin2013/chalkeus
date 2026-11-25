package cnitoken

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 25, 18, 0, 0, 0, time.UTC)

const secret = "c2VjcmV0LXNpZ25hdHVyZQ"

func kubeconfig(exp time.Time) []byte {
	claims := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"iat":%d,"exp":%d,"sub":"system:serviceaccount:kube-system:calico-cni-plugin"}`,
		exp.Add(-24*time.Hour).Unix(), exp.Unix())))
	return []byte("apiVersion: v1\nkind: Config\nusers:\n- name: calico\n  user:\n    token: eyJhbGciOiJSUzI1NiJ9." + claims + "." + secret + "\n")
}

func TestExpiry(t *testing.T) {
	exp := t0.Add(3 * time.Hour)
	got, err := Expiry(kubeconfig(exp))
	if err != nil || !got.Equal(exp) {
		t.Fatalf("got %v, %v", got, err)
	}
	for name, raw := range map[string]string{
		"no token":   "apiVersion: v1\n",
		"not a jwt":  "    token: " + secret + "\n",
		"bad claims": "    token: a.!!!." + secret + "\n",
		"no exp":     "    token: a." + base64.RawURLEncoding.EncodeToString([]byte(`{"iat":1}`)) + "." + secret + "\n",
	} {
		_, err := Expiry([]byte(raw))
		if err == nil {
			t.Errorf("%s: no error", name)
		} else if strings.Contains(err.Error(), secret) {
			t.Errorf("%s: the error quotes the token: %v", name, err)
		}
	}
}

func TestLeft(t *testing.T) {
	for d, want := range map[time.Duration]string{-time.Second: "expired", 0: "expired", 59 * time.Minute: "59m",
		23*time.Hour + 5*time.Minute + 40*time.Second: "23h05m"} {
		if got := Left(d); got != want {
			t.Errorf("Left(%v) = %q, want %q", d, got, want)
		}
	}
}

// fake replaces Read, Clock and Sleep; files[i] is what the i-th read returns (the last one repeats).
func fake(t *testing.T, files ...[]byte) *int {
	oldR, oldC, oldS := Read, Clock, Sleep
	t.Cleanup(func() { Read, Clock, Sleep = oldR, oldC, oldS })
	now := t0
	Clock = func() time.Time { return now }
	Sleep = func(d time.Duration) { now = now.Add(d) }
	reads := 0
	Read = func(string) ([]byte, error) {
		f := files[min(reads, len(files)-1)]
		reads++
		if f == nil {
			return nil, errors.New("docker exec failed")
		}
		return f, nil
	}
	return &reads
}

func TestEnsureFresh(t *testing.T) {
	fake(t, kubeconfig(t0.Add(20*time.Hour)))
	called := false
	note, err := Ensure("n", time.Hour, time.Minute, func() error { called = true; return nil })
	if note != "" || err != nil || called {
		t.Fatalf("note=%q err=%v restarted=%v", note, err, called)
	}
}

func TestEnsureRefreshes(t *testing.T) {
	// The 2026-11-25 case: expired; after the restart the old file is read once, then the new one.
	fake(t, kubeconfig(t0.Add(-time.Minute)), kubeconfig(t0.Add(-time.Minute)), kubeconfig(t0.Add(24*time.Hour)))
	called := 0
	note, err := Ensure("n", 50*time.Minute, 2*time.Minute, func() error { called++; return nil })
	if err != nil || called != 1 {
		t.Fatalf("err=%v restarts=%d", err, called)
	}
	if note != "calico-node restarted: the CNI token had expired; the new one has 23h59m left" {
		t.Errorf("note %q", note)
	}
}

func TestEnsureTooShortRefreshes(t *testing.T) {
	fake(t, kubeconfig(t0.Add(30*time.Minute)), kubeconfig(t0.Add(24*time.Hour)))
	called := 0
	if _, err := Ensure("n", 50*time.Minute, time.Minute, func() error { called++; return nil }); err != nil || called != 1 {
		t.Fatalf("err=%v restarts=%d", err, called)
	}
}

func TestEnsureFailures(t *testing.T) {
	fake(t, nil)
	if _, err := Ensure("n", time.Hour, time.Minute, func() error { return nil }); err == nil {
		t.Error("unreadable file: no error")
	}
	fake(t, kubeconfig(t0))
	_, err := Ensure("n", time.Hour, time.Minute, func() error { return errors.New("forbidden") })
	if err == nil || !strings.Contains(err.Error(), "forbidden") || !strings.Contains(err.Error(), Help) {
		t.Errorf("restart failure: %v", err)
	}
	reads := fake(t, kubeconfig(t0))
	_, err = Ensure("n", time.Hour, 30*time.Second, func() error { return nil })
	if err == nil || !strings.Contains(err.Error(), "didn't write a new one") || !strings.Contains(err.Error(), Help) {
		t.Errorf("no new token: %v", err)
	}
	if *reads < 3 {
		t.Errorf("gave up after %d reads", *reads)
	}
	if strings.Contains(err.Error(), secret) {
		t.Error("an error quotes the token")
	}
}
