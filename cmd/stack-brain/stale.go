package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/panyam/stack-brain/internal/env"
	"github.com/spf13/cobra"
)

// StaleEntry reports staleness for a single component (from Stackfile.md).
type StaleEntry struct {
	Component      string `json:"component"`
	Module         string `json:"module"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Location       string `json:"location"`
	CurrentRef     string `json:"current_ref,omitempty"`
	LatestRef      string `json:"latest_ref,omitempty"`
	Stale          bool   `json:"stale"`
}

// ExternalStaleEntry reports commit drift for an external repo.
type ExternalStaleEntry struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	LastChecked  string `json:"last_checked"`  // Commit hash when we last looked
	CurrentHead  string `json:"current_head"`  // Current HEAD commit
	Relationship string `json:"relationship,omitempty"`
	Stale        bool   `json:"stale"`
	CommitsBehind int   `json:"commits_behind,omitempty"` // 0 if we can't count
}

// StaleOutput combines Stackfile-based staleness and external repo drift.
type StaleOutput struct {
	Components []StaleEntry         `json:"components,omitempty"`
	Externals  []ExternalStaleEntry `json:"externals,omitempty"`
}

func newStaleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stale [project-dir]",
		Short: "Check which stack deps are outdated in a project",
		Long: `Reads a project's Stackfile.md, compares pinned versions against
the HEAD ref of each stack component's worktree, and reports which are stale.

If no project-dir is given, uses the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runStale,
	}
}

func runStale(cmd *cobra.Command, args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = expandHome(args[0])
	}

	output := StaleOutput{}

	// Check Stackfile.md-based staleness (existing behavior)
	stackfilePath := filepath.Join(projectDir, "Stackfile.md")
	pinned, err := parseStackfile(stackfilePath)
	if err == nil && len(pinned) > 0 {
		roots := discoveryRoots()
		components, err := DiscoverComponents(roots...)
		if err != nil {
			return fmt.Errorf("discovering components: %w", err)
		}

		compByName := make(map[string]*Component)
		for _, c := range components {
			compByName[strings.ToLower(c.Name)] = c
		}

		for _, p := range pinned {
			entry := StaleEntry{
				Component:      p.Name,
				Module:         p.Module,
				CurrentVersion: p.Version,
			}

			comp, ok := compByName[strings.ToLower(p.Name)]
			if !ok {
				entry.Stale = false
				entry.LatestVersion = "unknown"
				output.Components = append(output.Components, entry)
				continue
			}

			entry.Location = comp.Location
			entry.LatestVersion = comp.Version
			entry.LatestRef = gitHeadShort(comp.Location)
			entry.Stale = normalizeVersion(entry.CurrentVersion) != normalizeVersion(entry.LatestVersion)

			output.Components = append(output.Components, entry)
		}
	}

	// Check external repos for commit drift (when env is active)
	output.Externals = checkExternalStaleness()

	if len(output.Components) == 0 && len(output.Externals) == 0 {
		fmt.Fprintln(os.Stderr, "no stack components or external repos to check")
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// checkExternalStaleness compares each external repo's last_checked commit
// against its current HEAD to detect drift. Also counts commits behind if possible.
func checkExternalStaleness() []ExternalStaleEntry {
	envName, err := env.Detect()
	if err != nil || envName == "" {
		return nil
	}

	e, err := env.Load(envName)
	if err != nil {
		return nil
	}

	externals, err := e.ListExternals()
	if err != nil || len(externals) == 0 {
		return nil
	}

	var results []ExternalStaleEntry
	for _, ext := range externals {
		entry := ExternalStaleEntry{
			Name:         ext.Name,
			Path:         ext.Path,
			LastChecked:  ext.LastChecked,
			Relationship: ext.Relationship,
		}

		// Get current HEAD
		currentHead := gitHeadShortAt(ext.Path)
		entry.CurrentHead = currentHead

		if ext.LastChecked == "" || currentHead == "" {
			entry.Stale = currentHead != ""
		} else {
			entry.Stale = ext.LastChecked != currentHead
		}

		// Count commits between last_checked and HEAD
		if entry.Stale && ext.LastChecked != "" {
			entry.CommitsBehind = gitCountCommits(ext.Path, ext.LastChecked, "HEAD")
		}

		results = append(results, entry)
	}

	return results
}

type pinnedComponent struct {
	Name    string
	Module  string
	Version string
}

// parseStackfile reads a Stackfile.md and extracts the component table.
func parseStackfile(path string) ([]pinnedComponent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pinned []pinnedComponent
	scanner := bufio.NewScanner(f)
	inTable := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Detect markdown table by header separator
		if strings.HasPrefix(line, "|---") || strings.HasPrefix(line, "| ---") {
			inTable = true
			continue
		}

		// Skip the header row
		if strings.Contains(line, "| Component |") {
			continue
		}

		if inTable && strings.HasPrefix(line, "|") {
			cols := strings.Split(line, "|")
			// Trim and filter empty
			var fields []string
			for _, c := range cols {
				c = strings.TrimSpace(c)
				if c != "" {
					fields = append(fields, c)
				}
			}
			if len(fields) >= 3 {
				pinned = append(pinned, pinnedComponent{
					Name:    fields[0],
					Module:  fields[1],
					Version: fields[2],
				})
			}
		} else if inTable && !strings.HasPrefix(line, "|") {
			inTable = false
		}
	}

	return pinned, scanner.Err()
}
