// Package env manages stack-brain environments — named collections of repos
// with shared conventions, catalogs, and optional external repo tracking.
//
// An environment is a "lens" over a set of repos: it defines which repos you
// reason about together (lookup, DAG, stale checks) without changing anything
// intrinsic to those repos. Constraints and capabilities stay in the repo;
// the environment just scopes which repos participate in cross-cutting operations.
//
// Detection is automatic and deterministic (zero LLM tokens):
//  1. STACK_ENV env var — explicit override
//  2. cwd membership — walk up from working directory, match against registered repos
//  3. --env flag on individual commands
//
// Storage layout:
//
//	~/.config/stack-brain/
//	  envs/
//	    <name>/
//	      env.yaml          — environment config (repo list, brain-dir, metadata)
//	      conventions.md    — cross-cutting rules for this group of repos
//	      gaps.md           — capability gaps identified in this environment
//	      catalog.md        — auto-generated index of member repos' capabilities
//	      external/         — pointer files for repos you track but don't control
//	        <repo-name>/
//	          repo.yaml         — location, last-checked commit, relationship type
//	          capabilities.md   — your notes on what this external repo provides
//	          constraints.md    — rules you've learned about this external repo
package env

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Environment represents a named collection of repos that are reasoned about
// together. It tracks which repos are members, where external repo pointers
// live, and optionally a legacy brain-dir for migrated stack-brain setups.
type Environment struct {
	Name        string   `yaml:"name"`
	Repos       []string `yaml:"repos"`                 // Absolute paths to member repos
	ExternalDir string   `yaml:"external_dir,omitempty"` // Derived: path to external/ dir within env config
	BrainDir    string   `yaml:"brain_dir,omitempty"`    // Legacy: set when migrating from old stack-brain layout
}

// ExternalRepo represents a repo tracked by the environment but not controlled
// by the user. Instead of writing CAPABILITIES.md into the external repo, a
// thin pointer file (~80 tokens) is stored in the environment's external/ dir.
// The pointer tells the LLM *where to look*, not *what's there* — actual source
// is read on demand to avoid stale summaries.
type ExternalRepo struct {
	Name         string `yaml:"name"`
	Path         string `yaml:"path"`                    // Filesystem path or remote URL
	LastChecked  string `yaml:"last_checked,omitempty"`   // Git commit hash at time of last check
	DepType      string `yaml:"dep_type,omitempty"`       // "hard" (module import) or "semantic" (fork-of, etc.)
	Relationship string `yaml:"relationship,omitempty"`   // Human-readable: "fork-of", "shares-api-contract"
}

// ConfigDir returns the base config directory for all stack-brain environments.
// Respects XDG_CONFIG_HOME if set, otherwise defaults to ~/.config/stack-brain.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "stack-brain")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "stack-brain")
}

// EnvDir returns the on-disk directory for a specific environment's config and data.
func EnvDir(name string) string {
	return filepath.Join(ConfigDir(), "envs", name)
}

// Create initializes a new environment: creates its directory structure,
// writes an empty env.yaml, and scaffolds conventions.md and gaps.md.
// Returns an error if the environment already exists.
func Create(name string) (*Environment, error) {
	dir := EnvDir(name)
	if _, err := os.Stat(dir); err == nil {
		return nil, fmt.Errorf("environment %q already exists at %s", name, dir)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating env dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "external"), 0755); err != nil {
		return nil, fmt.Errorf("creating external dir: %w", err)
	}

	env := &Environment{
		Name:        name,
		ExternalDir: filepath.Join(dir, "external"),
	}

	if err := env.Save(); err != nil {
		return nil, err
	}

	// Scaffold empty cross-cutting docs
	scaffolds := map[string]string{
		"conventions.md": fmt.Sprintf("# Conventions\n\n> Environment: %s\n", name),
		"gaps.md":        fmt.Sprintf("# Gaps\n\n> Environment: %s\n", name),
	}
	for filename, content := range scaffolds {
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("creating %s: %w", filename, err)
		}
	}

	return env, nil
}

// Load reads an environment's config from its env.yaml on disk.
// The ExternalDir field is derived from the env directory (not stored in yaml).
func Load(name string) (*Environment, error) {
	dir := EnvDir(name)
	data, err := os.ReadFile(filepath.Join(dir, "env.yaml"))
	if err != nil {
		return nil, fmt.Errorf("loading environment %q: %w", name, err)
	}

	var env Environment
	if err := yaml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parsing env.yaml: %w", err)
	}
	env.ExternalDir = filepath.Join(dir, "external")
	return &env, nil
}

// Save persists the environment's config to env.yaml on disk.
func (e *Environment) Save() error {
	dir := EnvDir(e.Name)
	data, err := yaml.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshaling env: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "env.yaml"), data, 0644)
}

// AddRepo registers an absolute path as a member repo of this environment.
// The path is cleaned/normalized before storage. Returns an error if the
// repo is already a member.
func (e *Environment) AddRepo(absPath string) error {
	absPath = filepath.Clean(absPath)

	if slices.Contains(e.Repos, absPath) {
		return fmt.Errorf("repo %s is already in environment %s", absPath, e.Name)
	}

	e.Repos = append(e.Repos, absPath)
	return e.Save()
}

// RemoveRepo unregisters a repo path from this environment.
// Returns an error if the path is not currently a member.
func (e *Environment) RemoveRepo(absPath string) error {
	absPath = filepath.Clean(absPath)
	found := false
	var filtered []string
	for _, r := range e.Repos {
		if r == absPath {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if !found {
		return fmt.Errorf("repo %s is not in environment %s", absPath, e.Name)
	}
	e.Repos = filtered
	return e.Save()
}

// HasRepo reports whether a given absolute path (or any subdirectory of it)
// falls within one of this environment's registered repo paths. This enables
// cwd-based auto-detection: if you're inside ~/work/GVIP/GVIP_shared/pkg/,
// and ~/work/GVIP/GVIP_shared is a member, HasRepo returns true.
func (e *Environment) HasRepo(absPath string) bool {
	absPath = filepath.Clean(absPath)
	for _, r := range e.Repos {
		if absPath == r || strings.HasPrefix(absPath, r+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// AddExternal registers an external repo — one you track but don't control.
// Creates a directory under external/ with:
//   - repo.yaml: location, last-checked commit, dependency type, relationship
//   - capabilities.md: skeleton pointer file for the LLM (what to look at, not a summary)
//
// Nothing is written to the external repo itself.
func (e *Environment) AddExternal(ext ExternalRepo) error {
	dir := filepath.Join(e.ExternalDir, ext.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating external dir: %w", err)
	}

	// Write repo.yaml with metadata
	data, err := yaml.Marshal(ext)
	if err != nil {
		return fmt.Errorf("marshaling external repo: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "repo.yaml"), data, 0644); err != nil {
		return err
	}

	// Create skeleton capabilities.md if it doesn't already exist.
	// This is a pointer file (~80 tokens) — it tells the LLM *where to look*
	// in the external repo, not a full summary of what's there.
	capPath := filepath.Join(dir, "capabilities.md")
	if _, err := os.Stat(capPath); os.IsNotExist(err) {
		content := fmt.Sprintf("# %s (external)\n\n**Repo**: %s\n**Relevant to us**: (describe what matters)\n**Key paths**:\n  - (add key directories/files)\n**Last checked**: %s\n",
			ext.Name, ext.Path, ext.LastChecked)
		if err := os.WriteFile(capPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// ListExternals reads all external repo configs from this environment's
// external/ directory. Each subdirectory with a valid repo.yaml is returned.
// Directories without repo.yaml or with parse errors are silently skipped.
func (e *Environment) ListExternals() ([]ExternalRepo, error) {
	entries, err := os.ReadDir(e.ExternalDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var externals []ExternalRepo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(e.ExternalDir, entry.Name(), "repo.yaml"))
		if err != nil {
			continue
		}
		var ext ExternalRepo
		if err := yaml.Unmarshal(data, &ext); err != nil {
			continue
		}
		externals = append(externals, ext)
	}
	return externals, nil
}

// List returns the names of all environments that have a valid env.yaml.
// Directories under envs/ without env.yaml are ignored.
func List() ([]string, error) {
	envsDir := filepath.Join(ConfigDir(), "envs")
	entries, err := os.ReadDir(envsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := os.Stat(filepath.Join(envsDir, entry.Name(), "env.yaml")); err == nil {
				names = append(names, entry.Name())
			}
		}
	}
	return names, nil
}

// Detect determines the active environment without any LLM involvement.
// Resolution order:
//  1. STACK_ENV env var — explicit override, verified to exist
//  2. cwd membership — checks if the working directory falls within any
//     environment's registered repos
//
// Returns "" with no error if no environment matches.
// Returns an error if STACK_ENV points to a nonexistent env, or if the cwd
// is a member of multiple environments (ambiguous — user must disambiguate).
func Detect() (string, error) {
	// 1. Explicit env var takes priority
	if envName := os.Getenv("STACK_ENV"); envName != "" {
		if _, err := os.Stat(filepath.Join(EnvDir(envName), "env.yaml")); err != nil {
			return "", fmt.Errorf("STACK_ENV=%s but environment does not exist", envName)
		}
		return envName, nil
	}

	// 2. Auto-detect from cwd: check which environments contain this path
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil
	}
	cwd = filepath.Clean(cwd)

	names, err := List()
	if err != nil {
		return "", nil
	}

	var matches []string
	for _, name := range names {
		e, err := Load(name)
		if err != nil {
			continue
		}
		if e.HasRepo(cwd) {
			matches = append(matches, name)
		}
	}

	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("cwd is in multiple environments: %s — set STACK_ENV or use --env", strings.Join(matches, ", "))
	}
}

// ExpandHome expands a leading ~/ to the user's home directory.
// Paths without ~/ are returned unchanged.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
