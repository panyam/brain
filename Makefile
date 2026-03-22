CLAUDE_MD := $(HOME)/.claude/CLAUDE.md
COMMANDS_DIR := $(HOME)/.claude/commands
BRAIN_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
SKILLS := stack-integrate.md stack-update.md stack-catalog-refresh.md stack-audit.md checkpoint.md

POINTER_LINE := - Before adding third-party dependencies or building infrastructure from scratch, consult ~/newstack/brain/STACK_CATALOG.md

.PHONY: setup check clean

## setup: Ensure global CLAUDE.md has stack pointer, skills and CLI are installed
setup: setup-claude-md setup-skills setup-cli
	@echo "✓ Stack brain setup complete"

.PHONY: setup-claude-md
setup-claude-md:
	@if ! grep -q "STACK_CATALOG" "$(CLAUDE_MD)" 2>/dev/null; then \
		echo "" >> "$(CLAUDE_MD)"; \
		echo "## Stack System" >> "$(CLAUDE_MD)"; \
		echo "$(POINTER_LINE)" >> "$(CLAUDE_MD)"; \
		echo "- For full stack rules, conventions, and update procedures, see ~/newstack/brain/CLAUDE.md" >> "$(CLAUDE_MD)"; \
		echo "✓ Added stack pointer to $(CLAUDE_MD)"; \
	else \
		echo "· Stack pointer already in $(CLAUDE_MD)"; \
	fi

.PHONY: setup-skills
setup-skills:
	@mkdir -p "$(COMMANDS_DIR)"
	@for skill in $(SKILLS); do \
		cp "$(BRAIN_DIR)/skills/$$skill" "$(COMMANDS_DIR)/$$skill" 2>/dev/null || \
		cp "$(COMMANDS_DIR)/$$skill" "$(COMMANDS_DIR)/$$skill" 2>/dev/null || true; \
		echo "· Installed $$skill"; \
	done

.PHONY: setup-cli
setup-cli:
	@cd "$(BRAIN_DIR)/cmd/stack-brain" && go build -o stack-brain . && \
		mkdir -p "$(HOME)/.local/bin" && \
		cp stack-brain "$(HOME)/.local/bin/stack-brain" && \
		echo "· Installed stack-brain CLI to ~/.local/bin/stack-brain"

## refresh: Rebuild STACK_CATALOG.md via CLI
refresh:
	@stack-brain refresh

## check: Verify the brain is properly set up
check:
	@echo "Checking stack brain setup..."
	@test -f "$(CLAUDE_MD)" && grep -q "STACK_CATALOG" "$(CLAUDE_MD)" && \
		echo "✓ Global CLAUDE.md has stack pointer" || \
		echo "✗ Global CLAUDE.md missing stack pointer — run 'make setup'"
	@for skill in $(SKILLS); do \
		test -f "$(COMMANDS_DIR)/$$skill" && \
			echo "✓ Skill $$skill installed" || \
			echo "✗ Skill $$skill missing — run 'make setup'"; \
	done
	@test -f "$(BRAIN_DIR)/STACK_CATALOG.md" && \
		echo "✓ STACK_CATALOG.md exists" || \
		echo "✗ STACK_CATALOG.md missing — run 'make refresh'"
	@which stack-brain >/dev/null 2>&1 && \
		echo "✓ stack-brain CLI installed" || \
		echo "✗ stack-brain CLI missing — run 'make setup'"
	@echo "Done."

## clean: Remove stack pointer from global CLAUDE.md (does NOT remove skills)
clean:
	@echo "To remove, manually edit $(CLAUDE_MD) and remove the '## Stack System' section"
