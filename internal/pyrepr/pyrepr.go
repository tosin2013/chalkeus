// Package pyrepr formats values the way Python's repr() and json.dumps() do, so detail strings
// in the evidence and audit match the reference pilot byte for byte (016 research R10).
package pyrepr

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Str is repr(str): single quotes unless the string contains ' and no ", with Python's escapes.
func Str(s string) string {
	q := byte('\'')
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		q = '"'
	}
	var b strings.Builder
	b.WriteByte(q)
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == rune(q):
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\x%02x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte(q)
	return b.String()
}

// Value is repr() for the JSON-decoded values the audit handles: nil → None, bool → True/False,
// numbers (integral floats print as ints), strings, []any as a list, Tuple as a tuple.
func Value(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case bool:
		if x {
			return "True"
		}
		return "False"
	case string:
		return Str(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'g', -1, 64)
	case json.Number:
		return x.String()
	case Tuple:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = Value(e)
		}
		if len(parts) == 1 {
			return "(" + parts[0] + ",)"
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = Value(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []string:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = Str(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return fmt.Sprint(v)
}

// Tuple is a Python tuple for repr purposes.
type Tuple []any

// Strings is repr(list[str]).
func Strings(xs []string) string { return Value(xs) }

// OrderedMap is a JSON object with a fixed key order (Python dicts keep insertion order).
type OrderedMap struct {
	Keys   []string
	Values map[string]any
}

// JSONDumps is Python's json.dumps(obj) with default settings: ", " and ": " separators,
// ensure_ascii=True, keys in insertion order for OrderedMap and sorted for plain maps
// only when sortKeys is set.
func JSONDumps(v any) string {
	var b strings.Builder
	dump(&b, v)
	return b.String()
}

func dump(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case string:
		b.WriteString(jsonStr(x))
	case int:
		b.WriteString(strconv.Itoa(x))
	case float64:
		if x == float64(int64(x)) {
			b.WriteString(strconv.FormatInt(int64(x), 10))
		} else {
			b.WriteString(strconv.FormatFloat(x, 'g', -1, 64))
		}
	case json.Number:
		b.WriteString(x.String())
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			dump(b, e)
		}
		b.WriteByte(']')
	case []OrderedMap:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			dump(b, e)
		}
		b.WriteByte(']')
	case OrderedMap:
		b.WriteByte('{')
		for i, k := range x.Keys {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(jsonStr(k))
			b.WriteString(": ")
			dump(b, x.Values[k])
		}
		b.WriteByte('}')
	default:
		raw, _ := json.Marshal(x)
		b.Write(raw)
	}
}

// jsonStr encodes a string like Python's json.dumps with ensure_ascii=True.
func jsonStr(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			switch {
			case r < 0x20:
				fmt.Fprintf(&b, `\u%04x`, r)
			case r < 0x80:
				b.WriteRune(r)
			case r < 0x10000:
				fmt.Fprintf(&b, `\u%04x`, r)
			default:
				r -= 0x10000
				fmt.Fprintf(&b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// Dict is repr() of a dict whose keys arrive sorted (Kubernetes API JSON sorts map keys, so the
// reference's dicts read from `kubectl -o json` are in key order).
func Dict(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = Str(k) + ": " + Value(m[k])
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// SortedJSON is json.dumps(m, sort_keys=True).
func SortedJSON(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return JSONDumps(OrderedMap{Keys: keys, Values: m})
}
