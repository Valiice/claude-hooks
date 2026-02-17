---
name: categorize
description: Auto-categorize Claude Code sessions by analyzing prompts and responses. Adds category tags to session frontmatter for filtering and analytics.
---

# Categorize Sessions

## Overview

Analyze Claude Code session content and automatically assign category tags. Categories help with filtering, analytics, and retrospectives.

## Categories

- `bugfix` — Fixing bugs, errors, test failures
- `feature` — Building new functionality
- `refactor` — Restructuring code without changing behavior
- `exploration` — Research, reading code, understanding systems
- `docs` — Documentation, README, comments
- `testing` — Writing or fixing tests
- `devops` — CI/CD, deployment, infrastructure
- `config` — Configuration, settings, environment setup

## Arguments

- No arguments = categorize today's sessions
- A date like `2026-02-12` = that specific day
- `all` = all uncategorized sessions across all projects

## Workflow

1. **Resolve the vault path:**
   - Use `CLAUDE_VAULT` environment variable
   - If not set, ask the user for the path

2. **Find target sessions:**
   - Based on arguments, find session files
   - Filter to only sessions missing a `category` field in frontmatter

3. **Analyze each session:**
   - Read the session's prompts (`[!user]+` callouts) and responses (`[!claude]-` callouts)
   - Classify based on content:
     - Bug-related keywords (fix, bug, error, failing, broken) → `bugfix`
     - New functionality (add, create, implement, new feature) → `feature`
     - Restructuring (refactor, rename, move, reorganize, clean up) → `refactor`
     - Research (explain, how does, what is, show me, understand) → `exploration`
     - Documentation (docs, readme, comment, document) → `docs`
     - Testing (test, spec, coverage, assertion) → `testing`
     - Infrastructure (deploy, CI, docker, pipeline, build) → `devops`
     - Configuration (config, settings, env, setup, install) → `config`
   - Use the overall theme, not just keyword matching — consider the full context

4. **Present results for review:**
   - Show a table of sessions and proposed categories
   - Ask the user to confirm or adjust before applying

5. **Apply categories:**
   - For each confirmed session, add `category: {value}` to frontmatter after `project:` line
   - Use the Edit tool

## Rules

- **Always present for review** before modifying files
- If a session could fit multiple categories, pick the primary one
- If a session is ambiguous, suggest the best fit and note the uncertainty
- Never overwrite an existing `category` field — skip those sessions
- Be specific about why you chose each category (brief reasoning)
