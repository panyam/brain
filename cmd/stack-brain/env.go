package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/panyam/stack-brain/internal/env"
	"github.com/spf13/cobra"
)

// newEnvCmd builds the top-level "env" subcommand group for managing
// environments — named collections of repos that are reasoned about together.
func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments (named collections of repos)",
		Long: `Environments let you group repos and reason about them together.
Each environment has its own catalog, conventions, and gap tracking.

Detection is automatic: set STACK_ENV or just cd into a member repo.`,
	}

	cmd.AddCommand(
		newEnvCreateCmd(),
		newEnvListCmd(),
		newEnvAddCmd(),
		newEnvRemoveCmd(),
		newEnvInfoCmd(),
	)

	return cmd
}

// newEnvCreateCmd builds "env create <name>" which initializes a new environment
// with its directory structure, empty conventions.md, and gaps.md.
// Supports --import-catalog to bootstrap from an existing STACK_CATALOG.md
// (extracts component locations and registers them as member repos).
// Supports --brain-dir to record the legacy brain directory for migration.
func newEnvCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new environment",
		Args:  cobra.ExactArgs(1),
		RunE:  runEnvCreate,
	}

	cmd.Flags().String("import-catalog", "", "Import repos from an existing STACK_CATALOG.md")
	cmd.Flags().String("brain-dir", "", "Set legacy brain-dir for this environment (for migration)")

	return cmd
}

func runEnvCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	e, err := env.Create(name)
	if err != nil {
		return err
	}

	brainDir, _ := cmd.Flags().GetString("brain-dir")
	if brainDir != "" {
		e.BrainDir = env.ExpandHome(brainDir)
		if err := e.Save(); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "✓ Created environment %q at %s\n", name, env.EnvDir(name))

	importPath, _ := cmd.Flags().GetString("import-catalog")
	if importPath != "" {
		importPath = env.ExpandHome(importPath)
		return importFromCatalog(e, importPath)
	}

	return nil
}

// importFromCatalog parses a STACK_CATALOG.md markdown table and registers
// each component's Location column as a member repo. This is the migration
// path from the old stack-brain layout to the environment model.
func importFromCatalog(e *env.Environment, catalogPath string) error {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("reading catalog: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	inTable := false
	imported := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "|---") || strings.HasPrefix(line, "| ---") {
			inTable = true
			continue
		}
		if strings.Contains(line, "| Component |") {
			continue
		}

		if inTable && strings.HasPrefix(line, "|") {
			cols := strings.Split(line, "|")
			var fields []string
			for _, c := range cols {
				c = strings.TrimSpace(c)
				if c != "" {
					fields = append(fields, c)
				}
			}
			// Table columns: Component | Module | Location | Version | Status | Capabilities
			// Location may be relative (e.g., "newstack/devloop") — resolve against home dir
			if len(fields) >= 3 {
				loc := env.ExpandHome(fields[2])
				// Try as-is first, then relative to home dir
				if _, err := os.Stat(loc); err != nil {
					home, _ := os.UserHomeDir()
					loc = filepath.Join(home, fields[2])
				}
				if _, err := os.Stat(loc); err == nil {
					if err := e.AddRepo(loc); err == nil {
						imported++
						fmt.Fprintf(os.Stderr, "  + %s (%s)\n", fields[0], loc)
					}
				}
			}
		} else if inTable && !strings.HasPrefix(line, "|") {
			inTable = false
		}
	}

	fmt.Fprintf(os.Stderr, "✓ Imported %d repos from %s\n", imported, catalogPath)
	return nil
}

// newEnvListCmd builds "env list" which outputs JSON with all environments,
// annotating which one is currently active (via STACK_ENV or cwd detection).
func newEnvListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		RunE:  runEnvList,
	}
}

func runEnvList(cmd *cobra.Command, args []string) error {
	names, err := env.List()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "no environments found")
		return nil
	}

	active, _ := env.Detect()

	type envEntry struct {
		Name   string `json:"name"`
		Active bool   `json:"active"`
		Repos  int    `json:"repos"`
	}

	var entries []envEntry
	for _, name := range names {
		e, err := env.Load(name)
		if err != nil {
			continue
		}
		entries = append(entries, envEntry{
			Name:   name,
			Active: name == active,
			Repos:  len(e.Repos),
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

// newEnvAddCmd builds "env add [paths...]" which registers repos into the
// active environment. Supports:
//   - Multiple paths and shell globs (e.g., ~/work/GVIP/*)
//   - --external flag for repos you track but don't control (writes pointer
//     files to the env config, nothing into the external repo itself)
//   - --dep-type and --relationship for semantic dependency edges
//   - --env flag to target a specific environment instead of auto-detecting
func newEnvAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [paths...]",
		Short: "Add repos to the active environment",
		Long: `Add one or more repo paths to the active environment.
Use --external for repos you don't control (creates pointer files only).

Paths can use shell globs: stack-brain env add ~/work/GVIP/*

Environment is detected from STACK_ENV, --env flag, or cwd.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runEnvAdd,
	}

	cmd.Flags().Bool("external", false, "Add as external repo (pointer only, nothing written to repo)")
	cmd.Flags().String("env", "", "Target environment (overrides auto-detection)")
	cmd.Flags().String("dep-type", "hard", "Dependency type: hard or semantic")
	cmd.Flags().String("relationship", "", "Relationship description (e.g., fork-of, shares-api-contract)")

	return cmd
}

func runEnvAdd(cmd *cobra.Command, args []string) error {
	e, err := resolveEnv(cmd)
	if err != nil {
		return err
	}

	external, _ := cmd.Flags().GetBool("external")
	depType, _ := cmd.Flags().GetString("dep-type")
	relationship, _ := cmd.Flags().GetString("relationship")

	for _, arg := range args {
		arg = env.ExpandHome(arg)

		// Expand shell globs (e.g., ~/work/GVIP/* → list of directories)
		matches, err := filepath.Glob(arg)
		if err != nil || len(matches) == 0 {
			matches = []string{arg}
		}

		for _, path := range matches {
			absPath, err := filepath.Abs(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: cannot resolve %s: %v\n", path, err)
				continue
			}

			info, err := os.Stat(absPath)
			if err != nil || !info.IsDir() {
				fmt.Fprintf(os.Stderr, "warning: %s is not a directory, skipping\n", absPath)
				continue
			}

			if external {
				name := filepath.Base(absPath)
				lastChecked := gitHeadShortAt(absPath)
				ext := env.ExternalRepo{
					Name:         name,
					Path:         absPath,
					LastChecked:  lastChecked,
					DepType:      depType,
					Relationship: relationship,
				}
				if err := e.AddExternal(ext); err != nil {
					fmt.Fprintf(os.Stderr, "warning: %s: %v\n", name, err)
					continue
				}
				fmt.Fprintf(os.Stderr, "  + %s (external, %s)\n", name, absPath)
			} else {
				if err := e.AddRepo(absPath); err != nil {
					fmt.Fprintf(os.Stderr, "warning: %s: %v\n", absPath, err)
					continue
				}
				fmt.Fprintf(os.Stderr, "  + %s\n", absPath)
			}
		}
	}

	return nil
}

// newEnvRemoveCmd builds "env remove [path]" which unregisters a repo from
// the active environment. Does not delete any files from the repo itself.
func newEnvRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [path]",
		Short: "Remove a repo from the active environment",
		Args:  cobra.ExactArgs(1),
		RunE:  runEnvRemove,
	}

	cmd.Flags().String("env", "", "Target environment (overrides auto-detection)")

	return cmd
}

func runEnvRemove(cmd *cobra.Command, args []string) error {
	e, err := resolveEnv(cmd)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(env.ExpandHome(args[0]))
	if err != nil {
		return err
	}

	if err := e.RemoveRepo(absPath); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "  - %s removed from %s\n", absPath, e.Name)
	return nil
}

// newEnvInfoCmd builds "env info" which outputs JSON with full details about
// the active environment: name, config directory, member repos, external
// repos, and legacy brain-dir (if set during migration).
func newEnvInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show active environment details",
		RunE:  runEnvInfo,
	}

	cmd.Flags().String("env", "", "Target environment (overrides auto-detection)")

	return cmd
}

func runEnvInfo(cmd *cobra.Command, args []string) error {
	e, err := resolveEnv(cmd)
	if err != nil {
		return err
	}

	externals, _ := e.ListExternals()

	type infoOutput struct {
		Name      string             `json:"name"`
		Dir       string             `json:"dir"`
		Repos     []string           `json:"repos"`
		Externals []env.ExternalRepo `json:"externals,omitempty"`
		BrainDir  string             `json:"brain_dir,omitempty"`
	}

	output := infoOutput{
		Name:      e.Name,
		Dir:       env.EnvDir(e.Name),
		Repos:     e.Repos,
		Externals: externals,
		BrainDir:  e.BrainDir,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// resolveEnv determines which environment to operate on, using a three-level
// priority chain:
//  1. --env flag (explicit, highest priority)
//  2. STACK_ENV env var
//  3. cwd auto-detection (checks if working directory is inside a member repo)
//
// Returns an error if no environment can be determined.
func resolveEnv(cmd *cobra.Command) (*env.Environment, error) {
	envFlag, _ := cmd.Flags().GetString("env")
	if envFlag != "" {
		return env.Load(envFlag)
	}

	name, err := env.Detect()
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("no active environment — set STACK_ENV, use --env, or cd into a member repo")
	}

	return env.Load(name)
}

// gitHeadShortAt returns the abbreviated git commit hash for a given directory,
// or empty string if the directory is not a git repo.
func gitHeadShortAt(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
