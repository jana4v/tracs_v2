package gateway

import (
	"fmt"
	"strings"
)

// toString converts any value to a trimmed string representation.
func toString(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// firstNonEmpty returns the first non-blank string from the provided values.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

// recordOrEmpty returns rec[key] if it exists, or "" if the key is absent.
func recordOrEmpty(rec map[string]any, key string) any {
	v, ok := rec[key]
	if !ok {
		return ""
	}
	return v
}
