CLAUDE_MD := $(HOME)/.claude/CLAUDE.md
COMMANDS_DIR := $(HOME)/.claude/commands
SCRIPTS_DIR := $(HOME)/.claude/scripts
SETTINGS_JSON := $(HOME)/.claude/settings.json
BRAIN_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
SKILLS := stack-integrate.md stack-update.md stack-catalog-refresh.md stack-audit.md stack-constraints-check.md stack-constraints-overview.md stack-constraints-add.md stack-constraints-promote.md stack-migrate-frontend.md checkpoint.md

POINTER_LINE := - Before adding third-party dependencies or building infrastructure from scratch, consult ~/newstack/brain/STACK_CATALOG.md

.PHONY: setup check clean

## setup: Ensure global CLAUDE.md has stack pointer, skills, scripts, settings and CLI are installed
setup: setup-claude-md setup-skills setup-scripts setup-settings setup-cli
	@echo "✓ Stack brain setup complete"

.PHONY: setup-claude-md
setup-claude-md:
	@if [ ! -f "$(CLAUDE_MD)" ]; then \
		mkdir -p "$$(dirname $(CLAUDE_MD))"; \
		cp "$(BRAIN_DIR)/dotclaude/CLAUDE.md" "$(CLAUDE_MD)"; \
		echo "✓ Installed full CLAUDE.md from brain"; \
	elif ! grep -q "STACK_CATALOG" "$(CLAUDE_MD)" 2>/dev/null; then \
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

.PHONY: setup-scripts
setup-scripts:
	@mkdir -p "$(SCRIPTS_DIR)"
	@for script in $(BRAIN_DIR)/dotclaude/scripts/*; do \
		cp "$$script" "$(SCRIPTS_DIR)/" 2>/dev/null || true; \
		chmod +x "$(SCRIPTS_DIR)/$$(basename $$script)" 2>/dev/null || true; \
		echo "· Installed script $$(basename $$script)"; \
	done

.PHONY: setup-settings
setup-settings:
	@if [ ! -f "$(SETTINGS_JSON)" ]; then \
		cp "$(BRAIN_DIR)/dotclaude/settings.json" "$(SETTINGS_JSON)"; \
		echo "✓ Installed settings.json"; \
	else \
		echo "· settings.json already exists — not overwriting (sync manually if needed)"; \
	fi

.PHONY: setup-claude-md-full
setup-claude-md-full:
	@if [ ! -f "$(CLAUDE_MD)" ]; then \
		cp "$(BRAIN_DIR)/dotclaude/CLAUDE.md" "$(CLAUDE_MD)"; \
		echo "✓ Installed full CLAUDE.md"; \
	else \
		echo "· CLAUDE.md already exists — using pointer setup only"; \
	fi

.PHONY: setup-cli
setup-cli:
	@cd "$(BRAIN_DIR)/cmd/stack-brain" && go build -o stack-brain . && \
		mkdir -p "$(HOME)/.local/bin" && \
		cp stack-brain "$(HOME)/.local/bin/stack-brain" && \
		echo "· Installed stack-brain CLI to ~/.local/bin/stack-brain"

# Install into $GOBIN
install:
	cd cmd/stack-brain/ && go build -ldflags="$(LDFLAGS)" -o ${GOBIN}/stack-brain .

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
	@for script in $(BRAIN_DIR)/dotclaude/scripts/*; do \
		test -f "$(SCRIPTS_DIR)/$$(basename $$script)" && \
			echo "✓ Script $$(basename $$script) installed" || \
			echo "✗ Script $$(basename $$script) missing — run 'make setup'"; \
	done
	@test -f "$(SETTINGS_JSON)" && \
		echo "✓ settings.json exists" || \
		echo "✗ settings.json missing — run 'make setup'"
	@which stack-brain >/dev/null 2>&1 && \
		echo "✓ stack-brain CLI installed" || \
		echo "✗ stack-brain CLI missing — run 'make setup'"
	@echo "Done."

## clean: Remove stack pointer from global CLAUDE.md (does NOT remove skills)
clean:
	@echo "To remove, manually edit $(CLAUDE_MD) and remove the '## Stack System' section"
