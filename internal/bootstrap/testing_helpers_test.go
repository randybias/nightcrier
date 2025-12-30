package bootstrap

import "strings"

// contains is a test helper that checks if a string contains a substring.
// This is used in test files to verify error messages.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
