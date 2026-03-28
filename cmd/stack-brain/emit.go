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

// newEmitCmd builds "stack-brain emit" which compiles repo constraints,
// capabilities, and environment conventions into agent-native instruction files.
//
// For CLAUDE.md, uses marker-based injection to preserve hand-written content.
// For .cursor/rules/stack-brain.mdc, writes a dedicated file (no injection needed).
// For .windsurfrules and copilot-instructions.md, uses marker-based injection.
func newEmitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emit [repo-dir]",
		Short: "Generate agent instruction files from constraints and conventions",
		Long: `Compiles CONSTRAINTS.md, CAPABILITIES.md, and environment conventions
into agent-native instruction files.

Supported targets:
  claude    → CLAUDE.md (marker-based injection)
  cursor    → .cursor/rules/stack-brain.mdc (dedicated file)
  windsurf  → .windsurfrules (marker-based injection)
  copilot   → .github/copilot-instructions.md (marker-based injection)
  all       → all of the above

If no repo-dir is given, uses the current directory.

Examples:
  stack-brain emit                    # emit all targets for cwd
  stack-brain emit --target claude    # CLAUDE.md only
  stack-brain emit ~/work/GVIP/svc   # emit for specific repo`,
		Args: cobra.MaximumNArgs(1),
		RunE: runEmit,
	}

	cmd.Flags().String("target", "all", "Emit target: claude, cursor, windsurf, copilot, or all")
	cmd.Flags().Bool("dry-run", false, "Print what would be written without writing files")

	return cmd
}

func runEmit(cmd *cobra.Command, args []string) error {
	repoDir := "."
	if len(args) > 0 {
		repoDir = env.ExpandHome(args[0])
	}

	repoDir, err := filepath.Abs(repoDir)
	if err != nil {
		return fmt.Errorf("resolving repo dir: %w", err)
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

	// Determine env dir (if active)
	envDir := ""
	envName, err := env.Detect()
	if err == nil && envName != "" {
		envDir = env.EnvDir(envName)
	}

	// Gather content from repo + env
	content, err := emit.GatherContent(repoDir, envDir)
	if err != nil {
		return fmt.Errorf("gathering content: %w", err)
	}

	emitted := 0
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
			fmt.Fprintf(os.Stderr, "--- %s → %s ---\n", target, outPath)
			fmt.Println(output)
			fmt.Println()
			emitted++
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

		fmt.Fprintf(os.Stderr, "  ✓ %s → %s\n", target, outPath)
		emitted++
	}

	if emitted == 0 {
		fmt.Fprintln(os.Stderr, "no content to emit (no constraints, conventions, or capabilities found)")
	} else if !dryRun {
		fmt.Fprintf(os.Stderr, "✓ Emitted %d target(s)\n", emitted)
	}

	return nil
}
