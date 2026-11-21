package doctor

import (
	"strings"
	"testing"
)

func TestImagesRow(t *testing.T) {
	lock := map[string]string{"WORKER_IMAGE": "r/w@sha256:" + strings.Repeat("a", 64), "TOOLS_IMAGE": "r/t@sha256:" + strings.Repeat("b", 64),
		"EGRESS_IMAGE": "r/e@sha256:" + strings.Repeat("c", 64)}
	same := map[string]string{"worker": lock["WORKER_IMAGE"], "tools": lock["TOOLS_IMAGE"], "egress": lock["EGRESS_IMAGE"]}
	local := map[string]string{"worker": "localhost:5001/w@sha256:" + strings.Repeat("d", 64), "tools": "x", "egress": "y"}
	if st, _ := ImagesRow("pinned", same, lock); st != Pass {
		t.Errorf("pinned and matching: %s", st)
	}
	if st, det := ImagesRow("pinned", local, lock); st != Fail || !strings.Contains(det, "[egress tools worker]") {
		t.Errorf("pinned but different: %s %s", st, det)
	}
	if st, det := ImagesRow("local-build", local, lock); st != Warn || !strings.Contains(det, "local build") {
		t.Errorf("a local build is a warning, not a failure: %s %s", st, det)
	}
}
