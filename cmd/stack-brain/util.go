package main

import "strings"

func lowercase(s string) string {
	return strings.ToLower(s)
}

func containsLower(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// normalizeVersion strips the "v" prefix and "(local)" suffix for comparison.
func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimSuffix(v, " (local)")
	return strings.TrimSpace(v)
}
