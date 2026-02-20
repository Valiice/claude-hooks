---
name: bookmark
description: Bookmark the current Claude Code session in Obsidian. Adds bookmarked: true to the session's frontmatter for easy filtering and highlighting.
---

# Bookmark Session

## Overview

Mark the current Claude Code session as bookmarked in its Obsidian log file. Bookmarked sessions are highlighted in weekly/monthly reports and can be easily filtered.

## Arguments

- No arguments = bookmark current session
- `remove` = remove bookmark from current session

## Workflow

1. **Resolve the vault path:**
   - Use `CLAUDE_VAULT` environment variable
   - If not set, ask the user for the path

2. **Find the current session file:**
   - Get today's date in `YYYY-MM-DD` format
   - Get the current working directory's project name (basename of cwd)
   - Glob for `{vault}/{project}/{today}_*.md` files
   - Pick the most recently modified file (this is the current session)

3. **Read the session file** and check its frontmatter.

4. **Add or remove bookmark:**
   - If bookmarking: Add `bookmarked: true` after the `project:` line in frontmatter (or update existing)
   - If removing: Remove the `bookmarked: true` line from frontmatter
   - Use the Edit tool to make the change

5. **Confirm** to the user what was done, showing the file path.

## Rules

- Only modify the frontmatter, never the session content
- If no session file is found for today, inform the user
- If already bookmarked (when adding) or not bookmarked (when removing), inform the user — no error
