---
name: monthly
description: Generate a comprehensive monthly retrospective of Claude Code sessions. Covers project narratives, statistics, cost analysis, and patterns across the entire month.
---

# Monthly Retrospective

## Overview

Generate a comprehensive monthly retrospective summarizing all Claude Code activity. More detailed than weekly dashboards — includes project narratives, milestone tracking, and pattern analysis.

## Arguments

- No arguments = current month
- A month like `2026-02` or `February 2026` = that specific month
- `last-month` = previous month

## Workflow

1. **Resolve the vault path:**
   - Use `CLAUDE_VAULT` environment variable
   - If not set, ask the user for the path

2. **Resolve month** and determine date range (first to last day).

3. **Read all daily indexes** for the month. Parse session links.

4. **Read every session file** referenced. Extract frontmatter and content.

5. **Generate retrospective** at `{vault}/Monthly-{YYYY-MM}.md`:

```markdown
---
month: "{YYYY-MM}"
type: monthly-retrospective
tags:
  - claude-monthly
---

# Monthly Retrospective: {Month Year}

> Comprehensive review of all Claude Code sessions for {Month Year}.

---

## Summary

| Metric | Value |
|--------|-------|
| Total Sessions | {count} |
| Active Days | {days}/{total} |
| Total Time | {hours}h {mins}m |
| Total Cost | ~${total} |
| Tokens | {in} in / {out} out |
| Tool Calls | {count} |
| Commits | {count} |
| Projects | {count} |

## Project Narratives

### {Project Name}

**Sessions**: {count} | **Time**: {time} | **Cost**: ~${cost}

{Narrative paragraph describing what was built, key milestones, important decisions, and how the project evolved over the month. Reference specific sessions and dates.}

**Key milestones:**
- {date}: {milestone description}

(Repeat for each project)

## Weekly Breakdown

| Week | Sessions | Time | Cost | Focus |
|------|----------|------|------|-------|
| {date range} | {count} | {time} | ~${cost} | {primary project/theme} |

## Tool Usage Evolution

| Tool | Week 1 | Week 2 | Week 3 | Week 4 | Total |
|------|--------|--------|--------|--------|-------|
| {name} | {count} | {count} | {count} | {count} | {total} |

## Cost Analysis

| Project | Cost | % of Total |
|---------|------|------------|
| {name} | ~${cost} | {pct}% |

**Daily average**: ~${avg}/day (active days only)
**Weekly average**: ~${avg}/week

## Notable Sessions

{List bookmarked sessions and sessions with unusually high tool counts, costs, or long durations}

- [[{path}|{date} {time}]] — {project}: {brief description from prompts}

## Patterns & Insights

{Numbered list of observations:}
1. {Pattern observed across the month — e.g., "Most coding happened between 9-11am and 8-11pm"}
2. {Trend — e.g., "Refactoring increased in week 3 after feature-heavy weeks 1-2"}
3. {Notable shift — e.g., "Shifted from exploration to implementation mid-month"}
```

## Rules

- **Read every session file** — comprehensive coverage required
- Write project narratives as actual prose, not just stats — tell the story
- Identify genuine patterns, don't fabricate insights
- If a month has very few sessions, produce a shorter report accordingly
- Graceful degradation for missing frontmatter fields (tools, tokens, cost)
- Use em-dashes (`--`) in narrative text
- Keep tables aligned and readable
