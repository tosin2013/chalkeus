package doctor

import (
	"fmt"
	"sort"
)

// ImagesRow is doctor's "images" row. Pinned images (`./chalkeus forge up`) must match pilot/images.lock; a
// cluster stood up with `./chalkeus forge up --local-build` runs images built from this checkout, which never
// match the published digests, so it is a warning that says so, not a failure (found by the 025
// step-mode scenario: a failed doctor stops step mode from starting a run).
func ImagesRow(source string, running, lock map[string]string) (string, string) {
	if source == "local-build" {
		return Warn, "local build: the cluster runs images built from this checkout (`./chalkeus forge up --local-build`), not the published digests in pilot/images.lock"
	}
	stale := []string{}
	for k, key := range map[string]string{"worker": "WORKER_IMAGE", "tools": "TOOLS_IMAGE", "egress": "EGRESS_IMAGE"} {
		if !DigestMatches(running[k], lock[key]) {
			stale = append(stale, k)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		return Fail, fmt.Sprintf("the cluster runs older images than pilot/images.lock (%v); run `./chalkeus forge up`", stale)
	}
	return Pass, "match pilot/images.lock"
}
