package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

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

	stackfilePath := filepath.Join(projectDir, "Stackfile.md")
	pinned, err := parseStackfile(stackfilePath)
	if err != nil {
		return fmt.Errorf("reading Stackfile.md: %w", err)
	}

	if len(pinned) == 0 {
		fmt.Fprintln(os.Stderr, "no stack components found in Stackfile.md")
		return nil
	}

	// Discover all components to map names to locations
	brainDir := expandHome(os.Getenv("STACK_BRAIN_DIR"))
	if brainDir == "" {
		home, _ := os.UserHomeDir()
		brainDir = filepath.Join(home, "newstack", "brain")
	}
	stackRoot := filepath.Dir(brainDir)

	components, err := DiscoverComponents(stackRoot)
	if err != nil {
		return fmt.Errorf("discovering components: %w", err)
	}

	// Build lookup by component name (lowercased)
	compByName := make(map[string]*Component)
	for _, c := range components {
		compByName[strings.ToLower(c.Name)] = c
	}

	var results []StaleEntry
	for _, p := range pinned {
		entry := StaleEntry{
			Component:      p.Name,
			Module:         p.Module,
			CurrentVersion: p.Version,
		}

		comp, ok := compByName[strings.ToLower(p.Name)]
		if !ok {
			// Component not found in stack — skip
			entry.Stale = false
			entry.LatestVersion = "unknown"
			results = append(results, entry)
			continue
		}

		entry.Location = comp.Location
		entry.LatestVersion = comp.Version

		// Get git HEAD of component
		latestRef := gitHeadShort(comp.Location)
		entry.LatestRef = latestRef

		// Compare versions (normalize v prefix)
		entry.Stale = normalizeVersion(entry.CurrentVersion) != normalizeVersion(entry.LatestVersion)

		results = append(results, entry)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
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
