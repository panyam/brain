package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newMigrationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrations <component> <from-version> <to-version>",
		Short: "Concatenate migration files between versions",
		Long: `Finds and concatenates all migration files for a component
between the given versions, in order. Also extracts the Migrations
section from CAPABILITIES.md if present.

Example:
  stack-brain migrations servicekit 0.2.0 0.4.0`,
		Args: cobra.ExactArgs(3),
		RunE: runMigrations,
	}
}

func runMigrations(cmd *cobra.Command, args []string) error {
	componentName := args[0]
	fromVersion := args[1]
	toVersion := args[2]

	brainDir := viper.GetString("brain_dir")
	stackRoot := filepath.Dir(expandHome(brainDir))

	components, err := DiscoverComponents(stackRoot)
	if err != nil {
		return fmt.Errorf("discovering components: %w", err)
	}

	// Find the component
	var comp *Component
	for _, c := range components {
		if strings.EqualFold(c.Name, componentName) {
			comp = c
			break
		}
	}

	if comp == nil {
		return fmt.Errorf("component %q not found", componentName)
	}

	compDir := filepath.Dir(comp.FilePath)
	migrationsDir := filepath.Join(compDir, "migrations")

	fmt.Fprintf(os.Stdout, "# Migrations for %s: %s → %s\n\n", comp.Name, fromVersion, toVersion)

	// Check for migrations directory
	if info, err := os.Stat(migrationsDir); err == nil && info.IsDir() {
		// Find migration files and sort them
		entries, err := os.ReadDir(migrationsDir)
		if err != nil {
			return fmt.Errorf("reading migrations dir: %w", err)
		}

		var migrationFiles []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			// Include file if it's in the version range
			// Files named like: 0.1_to_0.2.md, 0.2_to_0.3.md, MIGRATION_0_3.md, etc.
			name := e.Name()
			if isInVersionRange(name, fromVersion, toVersion) {
				migrationFiles = append(migrationFiles, filepath.Join(migrationsDir, name))
			}
		}

		sort.Strings(migrationFiles)

		if len(migrationFiles) > 0 {
			for _, f := range migrationFiles {
				content, err := os.ReadFile(f)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not read %s: %v\n", f, err)
					continue
				}
				fmt.Fprintf(os.Stdout, "## From: %s\n\n", filepath.Base(f))
				fmt.Fprintln(os.Stdout, string(content))
				fmt.Fprintln(os.Stdout, "---\n")
			}
		} else {
			fmt.Fprintln(os.Stdout, "*No migration files found in the version range.*\n")
		}
	} else {
		fmt.Fprintln(os.Stdout, "*No migrations/ directory found.*\n")
	}

	// Also extract Migrations section from CAPABILITIES.md
	capContent, err := os.ReadFile(comp.FilePath)
	if err == nil {
		section := extractMigrationsSection(string(capContent))
		if section != "" {
			fmt.Fprintln(os.Stdout, "## From CAPABILITIES.md\n")
			fmt.Fprintln(os.Stdout, section)
		}
	}

	return nil
}

// isInVersionRange does a simple lexicographic check on the filename.
// Migration files are expected to sort lexicographically in version order.
func isInVersionRange(filename, from, to string) bool {
	// Normalize: replace dots and underscores for comparison
	norm := func(v string) string {
		v = strings.ReplaceAll(v, ".", "_")
		v = strings.ReplaceAll(v, "-", "_")
		return strings.ToLower(v)
	}

	nf := norm(filename)
	nFrom := norm(from)
	nTo := norm(to)

	// Check if the filename contains version indicators in range
	// e.g., "0_1_to_0_2.md" should match if from <= 0.1 and 0.2 <= to
	return nf >= nFrom || strings.Contains(nf, nFrom) || strings.Contains(nf, nTo) || nf <= nTo
}

// extractMigrationsSection pulls out the ## Migrations section from CAPABILITIES.md content.
func extractMigrationsSection(content string) string {
	lines := strings.Split(content, "\n")
	var section []string
	inSection := false

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "## Migrations") {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		if inSection {
			section = append(section, line)
		}
	}

	return strings.TrimSpace(strings.Join(section, "\n"))
}
