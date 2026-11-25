// Package cnitoken checks Calico's CNI token on the kind node (031 T034).
//
// Calico's network plugin calls the Kubernetes API with a 24-hour token that calico-node writes
// to /etc/cni/net.d/calico-kubeconfig and renews on a timer. On a Mac that timer runs on the
// Docker Desktop VM's clock, which stops while the Mac sleeps, so the token can expire unrenewed;
// every new pod then fails with "connection is unauthorized" (docs/troubleshooting.md
// #calico-token-expired). This package reads only the token's expiry. The token itself never
// leaves this package: it isn't returned, logged or put in an error.
package cnitoken

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tosin2013/chalkeus/internal/hostinfo"
)

// Path is the CNI kubeconfig on the node.
const Path = "/etc/cni/net.d/calico-kubeconfig"

// Help points to the troubleshooting entry.
const Help = "docs/troubleshooting.md#calico-token-expired"

// Expiry returns the expiry of the token in a CNI kubeconfig. Errors never quote the file.
func Expiry(kubeconfig []byte) (time.Time, error) {
	sc := bufio.NewScanner(bytes.NewReader(kubeconfig))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "token:") {
			continue
		}
		tok := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "token:")), `"'`)
		parts := strings.Split(tok, ".")
		if len(parts) != 3 {
			return time.Time{}, errors.New("the CNI token isn't a JWT")
		}
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
		if err != nil {
			return time.Time{}, errors.New("the CNI token's claims aren't base64url")
		}
		var claims struct {
			Exp int64 `json:"exp"`
		}
		if err := json.Unmarshal(raw, &claims); err != nil || claims.Exp == 0 {
			return time.Time{}, errors.New("the CNI token has no expiry")
		}
		return time.Unix(claims.Exp, 0).UTC(), nil
	}
	return time.Time{}, errors.New("no token in " + Path)
}

// Read returns the node's CNI kubeconfig (replaced in tests).
var Read = func(node string) ([]byte, error) {
	out, err := hostinfo.Command(30*time.Second, "docker", "exec", node, "cat", Path)
	if err != nil {
		return nil, fmt.Errorf("couldn't read %s on %s: %v", Path, node, err)
	}
	return out, nil
}

// NodeExpiry reads the node's token expiry.
func NodeExpiry(node string) (time.Time, error) {
	raw, err := Read(node)
	if err != nil {
		return time.Time{}, err
	}
	return Expiry(raw)
}

// Left formats the time left on the token, rounded down to minutes.
func Left(d time.Duration) string {
	if d <= 0 {
		return "expired"
	}
	d = d.Truncate(time.Minute)
	h, m := int(d.Hours()), int(d.Minutes())%60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh%02dm", h, m)
}

// Clock and Sleep are replaced in tests.
var (
	Clock = time.Now
	Sleep = time.Sleep
)

// Ensure makes sure the node's token lasts at least need from now. When it doesn't, it calls
// restart (restart calico-node and wait for it), then waits up to wait for a token that does.
// It returns a short note when it refreshed the token ("" when nothing was needed).
func Ensure(node string, need, wait time.Duration, restart func() error) (string, error) {
	exp, err := NodeExpiry(node)
	if err != nil {
		return "", err
	}
	left := exp.Sub(Clock())
	if left >= need {
		return "", nil
	}
	before := "had " + Left(left) + " left"
	if left <= 0 {
		before = "had expired"
	}
	if err := restart(); err != nil {
		return "", fmt.Errorf("the network plugin's token %s, and restarting calico-node failed: %v (%s)", before, err, Help)
	}
	end := Clock().Add(wait)
	for {
		if exp, err = NodeExpiry(node); err == nil && exp.Sub(Clock()) >= need {
			return fmt.Sprintf("calico-node restarted: the CNI token %s; the new one has %s left", before, Left(exp.Sub(Clock()))), nil
		}
		if !Clock().Before(end) {
			return "", fmt.Errorf("the network plugin's token %s, and calico-node didn't write a new one within %s (%s)", before, wait, Help)
		}
		Sleep(3 * time.Second)
	}
}
