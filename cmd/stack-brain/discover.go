package main

import (
	"os"
	"path/filepath"

	"github.com/panyam/stack-brain/internal/env"
	"github.com/spf13/viper"
)

// discoveryRoots returns the directories to scan for CAPABILITIES.md files.
// If an environment is active (via STACK_ENV or cwd detection), it returns
// the environment's registered repo paths — scoping discovery to just those repos.
// Otherwise, falls back to the legacy behavior: scan the parent of brain-dir
// (i.e., the entire ~/newstack/ tree).
//
// This is the single point where env-awareness meets component discovery.
// All commands (lookup, dag, refresh, stale, update) call this instead of
// computing stackRoot directly.
func discoveryRoots() []string {
	// Try environment detection first
	envName, err := env.Detect()
	if err == nil && envName != "" {
		e, err := env.Load(envName)
		if err == nil && len(e.Repos) > 0 {
			return e.Repos
		}
	}

	// Fallback: legacy brain-dir parent
	brainDir := expandHome(viper.GetString("brain_dir"))
	return []string{filepath.Dir(brainDir)}
}

// discoveryRootsWithBrainDir is like discoveryRoots but also returns the
// brain directory for commands that need it (e.g., refresh writes catalog there).
// When an environment is active, the brain dir comes from the env config
// (either env.BrainDir for migrated envs, or the env's own config dir).
func discoveryRootsWithBrainDir() (roots []string, brainDir string) {
	envName, err := env.Detect()
	if err == nil && envName != "" {
		e, err := env.Load(envName)
		if err == nil && len(e.Repos) > 0 {
			bd := e.BrainDir
			if bd == "" {
				bd = env.EnvDir(envName)
			}
			return e.Repos, bd
		}
	}

	brainDir = expandHome(viper.GetString("brain_dir"))
	return []string{filepath.Dir(brainDir)}, brainDir
}

// activeBrainDir returns the brain directory for the active context.
// For env-aware usage, this is either the env's BrainDir (migrated setup)
// or the env's config dir. For legacy usage, it's viper's brain_dir.
func activeBrainDir() string {
	envName, err := env.Detect()
	if err == nil && envName != "" {
		e, err := env.Load(envName)
		if err == nil {
			if e.BrainDir != "" {
				return e.BrainDir
			}
			return env.EnvDir(envName)
		}
	}

	brainDir := viper.GetString("brain_dir")
	if brainDir == "" {
		home, _ := os.UserHomeDir()
		brainDir = filepath.Join(home, "newstack", "brain")
	}
	return expandHome(brainDir)
}
