// Package secret holds credential values in memory without ever printing them
// (specs/016-go-pilot/research.md R7).
package secret

import (
	"fmt"

	"github.com/tosin2013/chalkeus/internal/verbose"
)

const redacted = "[redacted]"

// Value is a credential. Every formatting path prints "[redacted]"; only Reveal returns the
// value, for the HTTP and Kubernetes clients that must send it.
type Value struct{ v string }

// New wraps a credential value, and registers it so verbose diagnostics never print it.
func New(v string) Value {
	verbose.AddRedaction(v)
	return Value{v: v}
}

// Reveal returns the raw value. Never log or persist the result.
func (s Value) Reveal() string { return s.v }

// Empty reports whether no value is held.
func (s Value) Empty() bool { return s.v == "" }

func (s Value) String() string   { return redacted }
func (s Value) GoString() string { return redacted }

// Format makes %v, %+v, %#v, %s, %q and %x all print "[redacted]".
func (s Value) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(redacted)) }

// MarshalJSON never emits the value.
func (s Value) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }

// MarshalText never emits the value.
func (s Value) MarshalText() ([]byte, error) { return []byte(redacted), nil }
