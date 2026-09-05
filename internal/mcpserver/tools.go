package mcpserver

import "strings"

// defaultMax caps list tool results when the caller does not specify a
// limit, keeping tool output small enough for LLM context windows.
const defaultMax = 100

const redactedSecret = "[REDACTED]"

// deref returns the value pointed to by p, or the zero value when p is nil.
func deref[T any](p *T) (v T) {
	if p != nil {
		return *p
	}
	return v
}

// nonNil returns s, or an empty slice when nil, so results marshal as []
// rather than null.
func nonNil[S ~[]E, E any](s S) S {
	if s == nil {
		return S{}
	}
	return s
}

// resolveMax applies the default result cap when max is unset or invalid.
func resolveMax(max int) int {
	if max <= 0 {
		return defaultMax
	}
	return max
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
	return strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "credential")
}
