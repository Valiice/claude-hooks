---
name: project-stats
description: Generate per-project statistics from Obsidian session logs. Shows all-time stats, recent activity, commit timeline, and top files for a specific project.
---

# Project Stats

## Overview

Generate comprehensive statistics for a specific project from Claude Code session logs. Produces an all-time overview with recent activity trends.

## Arguments

- Project name (required) — matches the project folder name in the vault
- If not provided, ask the user which project to analyze

## Workflow

1. **Resolve the vault path:**
   - Use `CLAUDE_VAULT` environment variable
   - If not set, ask the user for the path

2. **Find all session files** for the project:
   - Glob `{vault}/{project}/*.md` (exclude Stats.md, Archive/)
   - Read frontmatter from each

3. **Aggregate statistics** across all sessions.

4. **Write stats page** to `{vault}/{project}/Stats.md`:

```markdown
---
project: {project}
type: project-stats
generated: {datetime}
tags:
  - claude-stats
  - {project-tag}
---

# {Project} — Session Statistics

## All-Time Overview

| Metric | Value |
|--------|-------|
| Total Sessions | {count} |
| First Session | {date} |
| Last Session | {date} |
| Total Time | {hours}h {mins}m |
| Total Cost | ~${total} |
| Total Tokens | {in} in / {out} out |
| Total Tool Calls | {count} |
| Total Commits | {count} |

## Recent Activity (Last 30 Days)

| Metric | Value |
|--------|-------|
| Sessions | {count} |
| Time | {time} |
| Cost | ~${cost} |
| Avg Session | {mins}min |

## Top Tools

| Tool | Count |
|------|-------|
| {name} | {count} |

## Top Files

| File | Times Accessed |
|------|---------------|
| {path} | {count} |

(Top 20 files)

## Commit Timeline

| Date | Commits | Key Changes |
|------|---------|-------------|
| {date} | {count} | {first commit message} |

## Session History

| Date | Time | Duration | Cost | Tools | Category |
|------|------|----------|------|-------|----------|
| {date} | [[{path}|{time}]] | {dur} | ~${cost} | {tools} | {cat} |
```

## Rules

- Graceful degradation for missing frontmatter fields
- Sort session history newest first
- Format large numbers with K suffix
- If project folder doesn't exist, inform the user
