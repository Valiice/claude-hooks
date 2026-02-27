---
name: weekly
description: Generate a weekly dashboard of Claude Code sessions from Obsidian logs. Aggregates stats, costs, tool usage, and activity patterns into a comprehensive report.
---

# Weekly Dashboard

## Overview

Generate a comprehensive weekly dashboard summarizing all Claude Code sessions. Aggregates statistics from session frontmatter including tool usage, token counts, costs, and activity patterns.

## Arguments

- No arguments = last 7 days (including today)
- `last-week` = previous Monday through Sunday
- A specific date like `2026-02-10` = 7 days starting from that date

## Workflow

1. **Resolve the vault path:**
   - Use `CLAUDE_VAULT` environment variable
   - If not set, ask the user for the path

2. **Resolve date range** from arguments.

3. **Read daily index files** (`{date}.md`) from the vault root for each date in the range. Parse session links to find all session files.

4. **Read each session file's frontmatter** to extract:
   - `project`, `duration`, `start_time`, `session_id`
   - `tools` (map of tool name to count)
   - `tokens_in`, `tokens_out`
   - `estimated_cost`
   - `files_touched` (list)
   - `branch`, `commits` (if available)
   - `bookmarked` (boolean)
   - `category` (if available)

5. **Aggregate statistics** across all sessions.

6. **Write the dashboard** to `{vault}/Weekly-{startDate}-to-{endDate}.md`:

```markdown
---
date_range: "{startDate} to {endDate}"
type: weekly-dashboard
tags:
  - claude-weekly
---

# Weekly Dashboard: {startDate} to {endDate}

## Overview

| Metric | Value |
|--------|-------|
| Sessions | {count} |
| Total Time | {hours}h {mins}m |
| Total Cost | ~${total} |
| Tokens In | {tokensIn} |
| Tokens Out | {tokensOut} |
| Tool Calls | {totalTools} |
| Commits | {totalCommits} |

## Projects

| Project | Sessions | Time | Cost | Tools |
|---------|----------|------|------|-------|
| {name} | {count} | {time} | ~${cost} | {tools} |

## Tool Usage

| Tool | Count | % |
|------|-------|---|
| {name} | {count} | {pct}% |

## Most Touched Files

| File | Times Accessed |
|------|---------------|
| {path} | {count} |

(Top 20 files across all sessions)

## Streaks & Trends

- **Active days**: {N}/{total} days
- **Consecutive days**: {streak}
- **Busiest day**: {day} ({sessions} sessions, ~${cost})
- **Busiest hour**: {hour}:00 ({sessions} sessions)
- **Avg session length**: {mins}min

## Bookmarked Sessions

- [[{path}|{time}]] — {project} ({duration})

(Only shown if bookmarked sessions exist)

## Daily Breakdown

| Date | Sessions | Time | Cost | Top Project |
|------|----------|------|------|-------------|
| {date} | {count} | {time} | ~${cost} | {project} |
```

## Rules

- If frontmatter fields (tools, tokens_in, estimated_cost) are missing, use 0/empty — graceful degradation
- Sort tool usage by count descending
- Sort files by access count descending
- Format token counts with K suffix (e.g., "45K") for readability
- Format costs to 2 decimal places
- If no sessions found for the date range, inform the user instead of writing an empty report
