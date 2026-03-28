package main

import (
	"os/exec"
	"strconv"
	"strings"
)

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

// gitCountCommits counts the number of commits between two refs in a given directory.
// Returns 0 if the count cannot be determined (e.g., refs don't exist, not a git repo).
func gitCountCommits(dir, fromRef, toRef string) int {
	cmd := exec.Command("git", "-C", dir, "rev-list", "--count", fromRef+".."+toRef)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return n
}
