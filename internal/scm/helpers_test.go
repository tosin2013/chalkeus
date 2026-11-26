package scm

import (
	"encoding/json"
	"os"
)

func jsonUnmarshal(raw []byte, v any) error { return json.Unmarshal(raw, v) }

func writeFile(p, s string) error { return os.WriteFile(p, []byte(s), 0o600) }
