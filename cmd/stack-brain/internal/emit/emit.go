// Package emit compiles repo-intrinsic knowledge (CONSTRAINTS.md, CAPABILITIES.md)
// and environment conventions into agent-native instruction files.
//
// Each agent has a different format for instruction files:
//   - Claude Code: CLAUDE.md (markdown)
//   - Cursor: .cursor/rules/<name>.mdc (markdown with YAML frontmatter)
//   - Windsurf: .windsurfrules (markdown)
//   - Copilot: .github/copilot-instructions.md (markdown)
//
// The emit system uses marker-based injection to preserve hand-written content
// in existing files. Content between <!-- stack-brain:start --> and
// <!-- stack-brain:end --> markers is managed by emit; everything else is untouched.
//
// Source hierarchy (what gets compiled):
//  1. Repo's CONSTRAINTS.md — architectural rules
//  2. Repo's CAPABILITIES.md — what this repo provides (summary only)
//  3. Environment's conventions.md — cross-cutting rules for the group
//  4. Environment's external repo pointers — relevant external dependencies
package emit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	MarkerStart = "<!-- stack-brain:start -->"
	MarkerEnd   = "<!-- stack-brain:end -->"
)

// Target represents an agent instruction format.
type Target string

const (
	TargetClaude   Target = "claude"
	TargetCursor   Target = "cursor"
	TargetWindsurf Target = "windsurf"
	TargetCopilot  Target = "copilot"
)

// AllTargets returns all supported emit targets.
func AllTargets() []Target {
	return []Target{TargetClaude, TargetCursor, TargetWindsurf, TargetCopilot}
}

// Content holds the compiled knowledge to be emitted.
type Content struct {
	RepoName     string
	Constraints  string // Parsed from CONSTRAINTS.md
	Conventions  string // From environment conventions.md
	Capabilities string // Summary from CAPABILITIES.md
}

// GatherContent reads constraints, conventions, and capabilities for a repo
// within an environment context. repoDir is the repo root. envDir is the
// environment config dir (empty string if no env active).
func GatherContent(repoDir string, envDir string) (*Content, error) {
	c := &Content{
		RepoName: filepath.Base(repoDir),
	}

	// Read repo's CONSTRAINTS.md
	constraintsPath := filepath.Join(repoDir, "CONSTRAINTS.md")
	if data, err := os.ReadFile(constraintsPath); err == nil {
		c.Constraints = extractConstraintRules(string(data))
	}

	// Read repo's CAPABILITIES.md (just a brief summary)
	capPath := filepath.Join(repoDir, "CAPABILITIES.md")
	if data, err := os.ReadFile(capPath); err == nil {
		c.Capabilities = extractCapabilitySummary(string(data))
	}

	// Read environment conventions
	if envDir != "" {
		convPath := filepath.Join(envDir, "conventions.md")
		if data, err := os.ReadFile(convPath); err == nil {
			conv := strings.TrimSpace(string(data))
			// Only include if there's actual content beyond the header
			lines := strings.Split(conv, "\n")
			var meaningful []string
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ">") {
					continue
				}
				meaningful = append(meaningful, line)
			}
			if len(meaningful) > 0 {
				c.Conventions = conv
			}
		}
	}

	return c, nil
}

// extractConstraintRules pulls the constraint entries from CONSTRAINTS.md,
// reformatting them as concise rules for agent instruction files.
func extractConstraintRules(content string) string {
	var rules []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentRule string
	var inRule bool

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "### ") {
			// Flush previous rule
			if inRule && currentRule != "" {
				rules = append(rules, currentRule)
			}
			name := strings.TrimPrefix(trimmed, "### ")
			currentRule = "### " + name
			inRule = true
			continue
		}

		if inRule {
			if strings.HasPrefix(trimmed, "**Rule**:") {
				rule := strings.TrimPrefix(trimmed, "**Rule**:")
				currentRule += "\n" + strings.TrimSpace(rule)
			} else if strings.HasPrefix(trimmed, "**Why**:") {
				why := strings.TrimPrefix(trimmed, "**Why**:")
				currentRule += "\n*Why: " + strings.TrimSpace(why) + "*"
			}
			// Skip Verify and Scope — agents don't need those
		}
	}
	// Flush last rule
	if inRule && currentRule != "" {
		rules = append(rules, currentRule)
	}

	if len(rules) == 0 {
		return ""
	}

	return strings.Join(rules, "\n\n")
}

// extractCapabilitySummary pulls a one-line summary from CAPABILITIES.md.
func extractCapabilitySummary(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var name string
	var tags []string
	section := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "# ") && name == "" {
			name = strings.TrimPrefix(trimmed, "# ")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			section = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			continue
		}
		if section == "provides" && strings.HasPrefix(trimmed, "- ") {
			tag := strings.TrimPrefix(trimmed, "- ")
			if idx := strings.Index(tag, ":"); idx > 0 {
				tag = tag[:idx]
			}
			// Strip bold markers for sub-components
			tag = strings.ReplaceAll(tag, "**", "")
			if idx := strings.Index(tag, " ("); idx > 0 {
				tag = tag[:idx]
			}
			tags = append(tags, strings.TrimSpace(tag))
		}
	}

	if name == "" {
		return ""
	}
	if len(tags) > 0 {
		return fmt.Sprintf("%s provides: %s", name, strings.Join(tags, ", "))
	}
	return name
}

// EmitForTarget generates the instruction file content for a specific agent target.
func EmitForTarget(target Target, content *Content) string {
	body := compileBody(content)
	if body == "" {
		return ""
	}

	switch target {
	case TargetCursor:
		return compileCursor(content, body)
	default:
		// Claude, Windsurf, Copilot all use plain markdown
		return body
	}
}

// compileBody generates the markdown content shared across most targets.
func compileBody(content *Content) string {
	var sections []string

	if content.Capabilities != "" {
		sections = append(sections, "## Overview\n"+content.Capabilities)
	}

	if content.Constraints != "" {
		sections = append(sections, "## Constraints\n\nArchitectural rules — do not violate these.\n\n"+content.Constraints)
	}

	if content.Conventions != "" {
		sections = append(sections, "## Conventions\n\n"+content.Conventions)
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n\n")
}

// compileCursor wraps content in Cursor's MDC format with frontmatter.
func compileCursor(content *Content, body string) string {
	return fmt.Sprintf(`---
description: "Architectural rules and conventions for %s (managed by stack-brain)"
globs:
  - "**/*"
alwaysApply: true
---

%s`, content.RepoName, body)
}

// OutputPath returns the file path where the emitted content should be written,
// relative to the repo root.
func OutputPath(target Target, repoDir string) string {
	switch target {
	case TargetClaude:
		return filepath.Join(repoDir, "CLAUDE.md")
	case TargetCursor:
		return filepath.Join(repoDir, ".cursor", "rules", "stack-brain.mdc")
	case TargetWindsurf:
		return filepath.Join(repoDir, ".windsurfrules")
	case TargetCopilot:
		return filepath.Join(repoDir, ".github", "copilot-instructions.md")
	default:
		return ""
	}
}

// WriteWithMarkers writes emitted content into an existing file using marker-based
// injection. Content between <!-- stack-brain:start --> and <!-- stack-brain:end -->
// is replaced; everything else is preserved. If markers don't exist, they're
// appended to the end of the file. If the file doesn't exist, it's created with
// just the marked content.
func WriteWithMarkers(filePath string, emittedContent string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	marked := fmt.Sprintf("%s\n%s\n%s", MarkerStart, emittedContent, MarkerEnd)

	existing, err := os.ReadFile(filePath)
	if err != nil {
		// File doesn't exist — create with just the marked content
		return os.WriteFile(filePath, []byte(marked+"\n"), 0644)
	}

	content := string(existing)
	startIdx := strings.Index(content, MarkerStart)
	endIdx := strings.Index(content, MarkerEnd)

	if startIdx >= 0 && endIdx >= 0 && endIdx > startIdx {
		// Replace between markers
		newContent := content[:startIdx] + marked + content[endIdx+len(MarkerEnd):]
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	// No markers found — append
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n" + marked + "\n"
	return os.WriteFile(filePath, []byte(content), 0644)
}

// WriteDirect writes emitted content directly to a file (no marker injection).
// Used for files that are fully managed by stack-brain (like .cursor/rules/stack-brain.mdc).
func WriteDirect(filePath string, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return os.WriteFile(filePath, []byte(content+"\n"), 0644)
}
