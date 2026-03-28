package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type UpdateResult struct {
	Component  string `json:"component"`
	Module     string `json:"module"`
	OldVersion string `json:"old_version"`
	NewVersion string `json:"new_version"`
	Method     string `json:"method"` // "go-get", "replace-local", "skipped"
	Error      string `json:"error,omitempty"`
}

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [project-dir]",
		Short: "Update stale stack deps in a project's go.mod",
		Long: `For each stale stack dependency, runs 'go get module@version'
to bump to the latest version, then runs 'go mod tidy'.
Updates the project's Stackfile.md with new versions.

If no project-dir is given, uses the current directory.

Use --dry-run to see what would be updated without making changes.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runUpdate,
	}

	cmd.Flags().Bool("dry-run", false, "Show what would be updated without making changes")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = expandHome(args[0])
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Check go.mod exists
	gomodPath := filepath.Join(projectDir, "go.mod")
	if _, err := os.Stat(gomodPath); os.IsNotExist(err) {
		return fmt.Errorf("no go.mod found in %s", projectDir)
	}

	// Get stale components
	stackfilePath := filepath.Join(projectDir, "Stackfile.md")
	pinned, err := parseStackfile(stackfilePath)
	if err != nil {
		return fmt.Errorf("reading Stackfile.md: %w", err)
	}

	// Discover components for version lookup
	roots := discoveryRoots()

	components, err := DiscoverComponents(roots...)
	if err != nil {
		return fmt.Errorf("discovering components: %w", err)
	}

	compByName := make(map[string]*Component)
	for _, c := range components {
		compByName[strings.ToLower(c.Name)] = c
	}

	// Read go.mod to check for replace directives
	gomodContent, err := os.ReadFile(gomodPath)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}
	gomodStr := string(gomodContent)

	var results []UpdateResult
	var updated bool

	for _, p := range pinned {
		comp, ok := compByName[strings.ToLower(p.Name)]
		if !ok {
			continue
		}

		currentVersion := normalizeVersion(p.Version)
		latestVersion := normalizeVersion(comp.Version)
		if currentVersion == latestVersion {
			continue // not stale
		}

		result := UpdateResult{
			Component:  p.Name,
			Module:     p.Module,
			OldVersion: p.Version,
			NewVersion: comp.Version,
		}

		// Check if this module has a replace directive
		hasReplace := strings.Contains(gomodStr, "replace "+p.Module) ||
			strings.Contains(gomodStr, "replace (\n") && strings.Contains(gomodStr, p.Module+" =>")

		if hasReplace {
			// For local replaces, just bump the require version
			result.Method = "replace-local"
			if !dryRun {
				goGetCmd := exec.Command("go", "get", fmt.Sprintf("%s@%s", p.Module, comp.Version))
				goGetCmd.Dir = projectDir
				if out, err := goGetCmd.CombinedOutput(); err != nil {
					result.Error = fmt.Sprintf("go get failed: %s", strings.TrimSpace(string(out)))
				} else {
					updated = true
				}
			}
		} else {
			// Standard go get
			result.Method = "go-get"
			if !dryRun {
				goGetCmd := exec.Command("go", "get", fmt.Sprintf("%s@%s", p.Module, comp.Version))
				goGetCmd.Dir = projectDir
				if out, err := goGetCmd.CombinedOutput(); err != nil {
					result.Error = fmt.Sprintf("go get failed: %s", strings.TrimSpace(string(out)))
				} else {
					updated = true
				}
			}
		}

		results = append(results, result)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "all stack deps are up to date")
		return nil
	}

	// Run go mod tidy if we made changes
	if updated && !dryRun {
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = projectDir
		if out, err := tidyCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: go mod tidy failed: %s\n", strings.TrimSpace(string(out)))
		}
	}

	if dryRun {
		fmt.Fprintln(os.Stderr, "DRY RUN — no changes made")
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
