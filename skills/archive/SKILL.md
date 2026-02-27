---
name: archive
description: Archive old Claude Code sessions to declutter project folders. Moves sessions older than a threshold to an Archive subfolder.
---

# Archive Sessions

## Overview

Move old Claude Code session files to an Archive subfolder within each project directory. Helps keep active session lists manageable while preserving history.

## Arguments

- No arguments = archive sessions older than 30 days
- `--days N` = custom age threshold (e.g., `--days 14`)
- `--dry-run` = show what would be archived without making changes
- `--project {name}` = only archive for a specific project

## Workflow

1. **Resolve the vault path:**
   - Use `CLAUDE_VAULT` environment variable
   - If not set, ask the user for the path

2. **Parse arguments** for threshold, dry-run flag, and project filter.

3. **Find sessions to archive:**
   - Scan project directories in the vault
   - For each session file (`{date}_{time}.md`), parse the date from the filename
   - If older than threshold, mark for archival
   - Exclude: Stats.md, any file already in Archive/

4. **Show preview:**
   - Display a table of files that would be archived, grouped by project
   - Show count and date range

5. **If not dry-run, confirm with user**, then:
   - Create `{vault}/{project}/Archive/` directory if needed
   - Move each session file to the Archive folder
   - Update daily index files to remove archived session links
   - Report what was moved

6. **Warn about wikilinks:**
   - After archiving, warn that any `[[project/date_time]]` wikilinks in other notes will need updating to `[[project/Archive/date_time]]`
   - List affected daily index files that were modified

## Rules

- **Always show preview** before archiving (unless --dry-run)
- Never archive today's sessions
- Never archive bookmarked sessions (check frontmatter for `bookmarked: true`)
- Preserve file content exactly — only change location
- Update daily indexes to remove archived entries
- If Archive/ folder already has a file with the same name, skip it and warn
