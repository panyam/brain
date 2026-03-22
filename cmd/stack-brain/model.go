package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Component represents a parsed CAPABILITIES.md
type Component struct {
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Module       string         `json:"module"`
	Location     string         `json:"location"`
	Status       string         `json:"status"`
	Capabilities []Capability   `json:"capabilities"`
	StackDeps    []string       `json:"stack_deps"`
	FilePath     string         `json:"file_path"`
}

// Capability is a single capability tag with description, optionally a sub-component
type Capability struct {
	Tag         string `json:"tag"`
	Description string `json:"description"`
	// Sub-component fields (non-empty if this is an embedded sub-library)
	SubLocation string `json:"sub_location,omitempty"`
	SubModule   string `json:"sub_module,omitempty"`
}

// Match scores a component against a set of query phrases.
// Returns total score (number of phrase hits across all searchable text).
func (c *Component) Match(phrases []string) int {
	score := 0
	// Build searchable text from all fields
	searchable := strings.ToLower(c.Name + " " + c.Module + " " + c.Status)
	for _, cap := range c.Capabilities {
		searchable += " " + strings.ToLower(cap.Tag+" "+cap.Description+" "+cap.SubModule+" "+cap.SubLocation)
	}
	for _, dep := range c.StackDeps {
		searchable += " " + strings.ToLower(dep)
	}

	for _, phrase := range phrases {
		p := strings.ToLower(phrase)
		// Count occurrences across the whole searchable text
		if strings.Contains(searchable, p) {
			score++
			// Bonus for exact tag match
			for _, cap := range c.Capabilities {
				if strings.ToLower(cap.Tag) == p {
					score++
				}
			}
		}
	}
	return score
}

// ParseCapabilitiesFile reads a CAPABILITIES.md and returns a Component.
func ParseCapabilitiesFile(path string) (*Component, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	c := &Component{FilePath: path}
	scanner := bufio.NewScanner(f)

	var section string
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Track which ## section we're in
		if strings.HasPrefix(trimmed, "## ") {
			section = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			continue
		}

		// H1 is the component name
		if strings.HasPrefix(trimmed, "# ") && c.Name == "" {
			c.Name = strings.TrimPrefix(trimmed, "# ")
			continue
		}

		switch section {
		case "module":
			if trimmed != "" && c.Module == "" {
				c.Module = trimmed
			}
		case "location":
			if trimmed != "" && c.Location == "" {
				c.Location = expandHome(trimmed)
			}
		case "status":
			if trimmed != "" && c.Status == "" {
				c.Status = trimmed
			}
		case "provides":
			if strings.HasPrefix(trimmed, "- **") {
				// Sub-component: - **name** (module): description
				cap := parseSubCapability(trimmed)
				if cap != nil {
					c.Capabilities = append(c.Capabilities, *cap)
				}
			} else if strings.HasPrefix(trimmed, "- location:") {
				// Sub-component location continuation
				if len(c.Capabilities) > 0 {
					loc := strings.TrimSpace(strings.TrimPrefix(trimmed, "- location:"))
					c.Capabilities[len(c.Capabilities)-1].SubLocation = expandHome(loc)
				}
			} else if strings.HasPrefix(trimmed, "- module:") {
				// Sub-component module continuation
				if len(c.Capabilities) > 0 {
					mod := strings.TrimSpace(strings.TrimPrefix(trimmed, "- module:"))
					c.Capabilities[len(c.Capabilities)-1].SubModule = mod
				}
			} else if strings.HasPrefix(trimmed, "- ") {
				// Regular capability: - tag: description
				cap := parseCapability(trimmed)
				if cap != nil {
					c.Capabilities = append(c.Capabilities, *cap)
				}
			}
		case "stack dependencies":
			if strings.HasPrefix(trimmed, "- ") {
				dep := strings.TrimPrefix(trimmed, "- ")
				if strings.ToLower(dep) != "none" {
					// Extract component name (before the parenthetical module path)
					if idx := strings.Index(dep, " ("); idx > 0 {
						dep = dep[:idx]
					}
					c.StackDeps = append(c.StackDeps, dep)
				}
			}
		}
	}

	// Derive version from source instead of CAPABILITIES.md
	c.Version = resolveVersion(filepath.Dir(path))

	return c, scanner.Err()
}

// resolveVersion derives the component version from its source:
// 1. Git tag (latest tag reachable from HEAD)
// 2. package.json version field
// 3. Fallback to git short hash
func resolveVersion(dir string) string {
	// Try git tag first
	if v := gitLatestTag(dir); v != "" {
		return v
	}
	// Try package.json
	if v := packageJSONVersion(dir); v != "" {
		return v
	}
	// Fallback to git short hash
	if v := gitHeadShort(dir); v != "" {
		return v
	}
	return "unknown"
}

func gitLatestTag(dir string) string {
	cmd := exec.Command("git", "-C", dir, "describe", "--tags", "--abbrev=0")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitHeadShort(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func packageJSONVersion(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	return pkg.Version
}

// parseCapability parses "- tag: description"
func parseCapability(line string) *Capability {
	line = strings.TrimPrefix(line, "- ")
	parts := strings.SplitN(line, ": ", 2)
	if len(parts) < 2 {
		return &Capability{Tag: strings.TrimSpace(line)}
	}
	return &Capability{
		Tag:         strings.TrimSpace(parts[0]),
		Description: strings.TrimSpace(parts[1]),
	}
}

// parseSubCapability parses "- **name** (module): description"
func parseSubCapability(line string) *Capability {
	line = strings.TrimPrefix(line, "- ")
	// Extract bold name: **name**
	start := strings.Index(line, "**")
	if start < 0 {
		return nil
	}
	end := strings.Index(line[start+2:], "**")
	if end < 0 {
		return nil
	}
	name := line[start+2 : start+2+end]
	rest := line[start+2+end+2:]

	cap := &Capability{Tag: name}

	// Extract (module) if present
	if strings.HasPrefix(strings.TrimSpace(rest), "(") {
		modEnd := strings.Index(rest, ")")
		if modEnd > 0 {
			modStart := strings.Index(rest, "(")
			cap.SubModule = rest[modStart+1 : modEnd]
			rest = rest[modEnd+1:]
		}
	}

	// Extract description after ": "
	if idx := strings.Index(rest, ": "); idx >= 0 {
		cap.Description = strings.TrimSpace(rest[idx+2:])
	} else {
		cap.Description = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
	}

	return cap
}

// DiscoverComponents finds all CAPABILITIES.md files under the given roots.
func DiscoverComponents(roots ...string) ([]*Component, error) {
	var components []*Component
	seen := make(map[string]bool)

	for _, root := range roots {
		root = expandHome(root)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip errors
			}
			// Skip node_modules, .git, vendor, brain directory itself
			if info.IsDir() {
				base := filepath.Base(path)
				if base == "node_modules" || base == ".git" || base == "vendor" {
					return filepath.SkipDir
				}
			}
			if info.Name() == "CAPABILITIES.md" && !seen[path] {
				// Skip the template
				if strings.Contains(path, "brain/templates/") {
					return nil
				}
				seen[path] = true
				comp, err := ParseCapabilitiesFile(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not parse %s: %v\n", path, err)
					return nil
				}
				components = append(components, comp)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking %s: %w", root, err)
		}
	}
	return components, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
