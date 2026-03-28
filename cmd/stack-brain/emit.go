package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/panyam/stack-brain/internal/emit"
	"github.com/panyam/stack-brain/internal/env"
	"github.com/spf13/cobra"
)

// newEmitCmd builds "stack-brain emit <env> [repos...]" which compiles repo
// constraints, capabilities, and environment conventions into agent-native
// instruction files.
//
// The env argument is required — you must be explicit about which environment's
// conventions are being stamped into the repos. This is intentional: emit is a
// write operation that pushes knowledge into repos, so there should be no magic.
//
// If no repos are given, emits into all repos registered in the environment.
func newEmitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emit <env> [repos...]",
		Short: "Generate agent instruction files from constraints and conventions",
		Long: `Compiles CONSTRAINTS.md, CAPABILITIES.md, and environment conventions
into agent-native instruction files.

The env argument is required — it determines which environment's conventions
are included alongside each repo's own constraints.

If no repos are given, emits into all repos registered in the environment.

Supported targets:
  claude    → CLAUDE.md (marker-based injection)
  cursor    → .cursor/rules/stack-brain.mdc (dedicated file)
  windsurf  → .windsurfrules (marker-based injection)
  copilot   → .github/copilot-instructions.md (marker-based injection)
  all       → all of the above

Examples:
  stack-brain emit newstack                                    # all repos in env
  stack-brain emit newstack ~/projects/lilbattle ~/projects/slyds  # specific repos
  stack-brain emit gvip --target cursor                        # cursor only
  stack-brain emit newstack --dry-run                          # preview`,
		Args: cobra.MinimumNArgs(1),
		RunE: runEmit,
	}

	cmd.Flags().String("target", "all", "Emit target: claude, cursor, windsurf, copilot, or all")
	cmd.Flags().Bool("dry-run", false, "Print what would be written without writing files")

	return cmd
}

func runEmit(cmd *cobra.Command, args []string) error {
	envName := args[0]

	// Load the environment
	e, err := env.Load(envName)
	if err != nil {
		return fmt.Errorf("loading environment %q: %w", envName, err)
	}
	envDir := env.EnvDir(envName)

	// Determine which repos to emit into
	var repoDirs []string
	if len(args) > 1 {
		// Explicit repos from args
		for _, arg := range args[1:] {
			absPath, err := filepath.Abs(env.ExpandHome(arg))
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: cannot resolve %s: %v\n", arg, err)
				continue
			}
			repoDirs = append(repoDirs, absPath)
		}
	} else {
		// All repos in the environment
		repoDirs = e.Repos
	}

	if len(repoDirs) == 0 {
		return fmt.Errorf("no repos to emit into — add repos to env %q or specify them as arguments", envName)
	}

	targetFlag, _ := cmd.Flags().GetString("target")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Determine targets
	var targets []emit.Target
	if targetFlag == "all" {
		targets = emit.AllTargets()
	} else {
		t := emit.Target(strings.ToLower(targetFlag))
		targets = []emit.Target{t}
	}

	totalEmitted := 0
	for _, repoDir := range repoDirs {
		// Verify directory exists
		if info, err := os.Stat(repoDir); err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "warning: %s is not a directory, skipping\n", repoDir)
			continue
		}

		// Gather content from repo + env
		content, err := emit.GatherContent(repoDir, envDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", repoDir, err)
			continue
		}

		repoName := content.RepoName
		repoEmitted := 0
		for _, target := range targets {
			output := emit.EmitForTarget(target, content)
			if output == "" {
				continue
			}

			outPath := emit.OutputPath(target, repoDir)
			if outPath == "" {
				continue
			}

			if dryRun {
				fmt.Fprintf(os.Stderr, "--- %s: %s → %s ---\n", repoName, target, outPath)
				fmt.Println(output)
				fmt.Println()
				repoEmitted++
				continue
			}

			// Cursor gets a dedicated file (fully managed by stack-brain).
			// Others use marker injection to preserve existing content.
			switch target {
			case emit.TargetCursor:
				if err := emit.WriteDirect(outPath, output); err != nil {
					fmt.Fprintf(os.Stderr, "warning: %s: %v\n", outPath, err)
					continue
				}
			default:
				if err := emit.WriteWithMarkers(outPath, output); err != nil {
					fmt.Fprintf(os.Stderr, "warning: %s: %v\n", outPath, err)
					continue
				}
			}

			fmt.Fprintf(os.Stderr, "  ✓ %s: %s → %s\n", repoName, target, outPath)
			repoEmitted++
		}
		totalEmitted += repoEmitted
	}

	if totalEmitted == 0 {
		fmt.Fprintln(os.Stderr, "no content to emit (no constraints, conventions, or capabilities found)")
	} else if !dryRun {
		fmt.Fprintf(os.Stderr, "✓ Emitted %d file(s) across %d repo(s)\n", totalEmitted, len(repoDirs))
	}

	return nil
}
