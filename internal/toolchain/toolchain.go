// Package toolchain reads and checks a toolchain image's /opt/toolchain/toolchain.json and turns
// it into a pod's environment (014 R5, specs/014-any-project/contracts/toolchain-image.md). The
// same rules hold for specfoundry's catalog images and for operator-supplied ones.
package toolchain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Mount is where the init container's copy of /opt/toolchain is mounted in every pod.
const Mount = "/opt/toolchain"

// Spec is a toolchain.json.
type Spec struct {
	Name           string            `json:"name"`
	VersionCommand []string          `json:"version_command"`
	Path           []string          `json:"path"`
	Env            map[string]string `json:"env"`
	SetupCommand   []string          `json:"setup_command,omitempty"`
	Proxy          string            `json:"proxy"`
}

var (
	reservedEnv = regexp.MustCompile(`(?i)^(https?_proxy|no_proxy|all_proxy|path|pilot_deps)$`)
	secretEnv   = regexp.MustCompile(`(?i)token|key|secret|password`)
	envName     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// Parse reads and checks a toolchain.json (the probe's rule checks; TOOLCHAIN_CONTRACT on failure).
func Parse(raw []byte) (Spec, error) {
	var s Spec
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("TOOLCHAIN_CONTRACT: toolchain.json is not valid JSON: %v", err)
	}
	bad := func(format string, a ...any) (Spec, error) {
		return s, fmt.Errorf("TOOLCHAIN_CONTRACT: toolchain.json "+format, a...)
	}
	if strings.TrimSpace(s.Name) == "" {
		return bad("needs a name")
	}
	if len(s.VersionCommand) == 0 {
		return bad("needs a version_command")
	}
	for _, p := range s.Path {
		if !strings.HasPrefix(p, Mount+"/") && !strings.HasPrefix(p, "${PILOT_DEPS}/") {
			return bad("path entry %q must be under %s or ${PILOT_DEPS}", p, Mount)
		}
	}
	for k := range s.Env {
		switch {
		case !envName.MatchString(k):
			return bad("env name %q is not a valid variable name", k)
		case reservedEnv.MatchString(k):
			return bad("env may not set %s: the pilot sets it", k)
		case secretEnv.MatchString(k):
			return bad("env may not set %s: no secret-like names", k)
		}
	}
	if s.Proxy != "env" && s.Proxy != "maven-settings" {
		return bad("proxy must be env or maven-settings, got %q", s.Proxy)
	}
	return s, nil
}

// expand replaces ${PILOT_DEPS} with the role's dependency directory.
func expand(v, deps string) string { return strings.ReplaceAll(v, "${PILOT_DEPS}", deps) }

// PodEnv is the pod environment for a role: PILOT_DEPS (the role's dependency directory: the
// worker's own, the prefetch's writable cache, the grader's read-only cache), the toolchain's
// variables, and PATH with the toolchain first. Sorted by name, so renders are stable.
func (s Spec) PodEnv(deps, basePath string) [][2]string {
	out := [][2]string{{"PILOT_DEPS", deps}}
	keys := make([]string, 0, len(s.Env))
	for k := range s.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, [2]string{k, expand(s.Env[k], deps)})
	}
	var path []string
	for _, p := range s.Path {
		path = append(path, expand(p, deps))
	}
	return append(out, [2]string{"PATH", strings.Join(append(path, basePath), ":")})
}

// SetupScript is the shell script the worker's and the prefetch's setup step runs from the
// project root: the toolchain's setup command, then the target's install command.
func (s Spec) SetupScript(install string) string {
	var b strings.Builder
	b.WriteString("set -eu\n")
	if len(s.SetupCommand) > 0 {
		b.WriteString(shellJoin(s.SetupCommand) + "\n")
	}
	if strings.TrimSpace(install) != "" {
		b.WriteString(install + "\n")
	}
	return b.String()
}

func shellJoin(args []string) string {
	q := make([]string, len(args))
	for i, a := range args {
		q[i] = "'" + strings.ReplaceAll(a, "'", `'"'"'`) + "'"
	}
	return strings.Join(q, " ")
}

// EnvYAML renders env entries for a container spec at the given indent.
func EnvYAML(env [][2]string, indent string) string {
	var b strings.Builder
	for _, kv := range env {
		v, _ := json.Marshal(kv[1])
		fmt.Fprintf(&b, "%s- {name: %s, value: %s}\n", indent, kv[0], v)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// Blocks are the YAML snippets an item-run pod template takes for its toolchain (014 R5):
// the copy init container, the environment, the mount and the volume. All empty for a
// toolchain built into the worker and tools images (the fixture's python).
type Blocks struct{ Inits, Env, Mounts, Volumes string }

// BasePath is the worker and tools images' PATH, after which the toolchain's entries go first.
const BasePath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

// PodBlocks renders a pod's toolchain blocks: image is the toolchain image (digest), spec its
// checked toolchain.json, deps the role's dependency directory.
func PodBlocks(image string, spec *Spec, deps string) Blocks {
	if image == "" || spec == nil {
		return Blocks{}
	}
	return Blocks{
		Inits: `        - name: toolchain
          image: ` + image + `
          command: [cp, -R, /opt/toolchain/., /dest/]
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: [ALL]}}
          resources: {requests: {cpu: 50m, memory: 32Mi}, limits: {cpu: 500m, memory: 256Mi}}
          volumeMounts:
            - {name: toolchain, mountPath: /dest}`,
		Env:     EnvYAML(spec.PodEnv(deps, BasePath), "            "),
		Mounts:  `            - {name: toolchain, mountPath: ` + Mount + `, readOnly: true}`,
		Volumes: `        - {name: toolchain, emptyDir: {sizeLimit: 2Gi}}`,
	}
}

// SetupInit is the worker's setup init container (014 R7): with the worker's proxy, it checks
// out the snapshot in a scratch directory and runs /in/setup.sh there (the toolchain's setup and
// the target's install command), installing into the worker's own dependency directory.
func SetupInit(toolsImage, proxyURL string, spec *Spec, deps string) string {
	env := []([2]string){{"HOME", "/home/worker"}, {"HTTPS_PROXY", proxyURL}, {"https_proxy", proxyURL}, {"NO_PROXY", ""}}
	if spec != nil {
		env = append(env, spec.PodEnv(deps, BasePath)...)
	}
	mounts := `            - {name: inputs, mountPath: /in, readOnly: true}
            - {name: home, mountPath: /home/worker}
            - {name: tmp, mountPath: /tmp}`
	if spec != nil {
		mounts += "\n            - {name: toolchain, mountPath: " + Mount + ", readOnly: true}"
	}
	return `        - name: setup
          image: ` + toolsImage + `
          command:
            - sh
            - -c
            - >-
              set -eu;
              git clone --quiet /in/base.bundle /tmp/.setup;
              cd /tmp/.setup;
              git checkout --quiet "$(/usr/bin/python3 -c 'import json; print(json.load(open("/in/meta.json"))["base_revision"])')";
              sh /in/setup.sh
          env:
` + EnvYAML(env, "            ") + `
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: [ALL]}}
          resources: {requests: {cpu: 500m, memory: 512Mi}, limits: {cpu: "2", memory: 2Gi}}
          volumeMounts:
` + mounts
}
