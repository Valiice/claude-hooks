---
name: synopsis
description: Generate a synopsis of Claude Code sessions from Obsidian logs. Use when the user wants a summary or retrospective of their work sessions - invoked via /synopsis or when asked for a synopsis/summary of sessions.
---

# Synopsis Generator

## Overview

Read Claude Code session logs from Obsidian and generate a structured synopsis of what was accomplished. The synopsis captures the narrative of the work — what was built, what decisions were made, what patterns emerged.

## Arguments

- No arguments = today
- A date like `2026-02-12` = that specific day
- A range like `2026-02-12 to 2026-02-13` = those days inclusive
- `yesterday` = yesterday's date
- `week` = last 7 days

## Workflow

1. **Resolve the vault path:**
   - First, check `.claude/claude-hooks.local.md` in the current project directory — if it exists, read the `vault_path` field from its YAML frontmatter
   - If not found there, check `~/.claude/claude-hooks.local.md` (user-global settings)
   - If still not found, read the `CLAUDE_VAULT` environment variable: run `echo $CLAUDE_VAULT` in Bash (not `%CLAUDE_VAULT%` — that's CMD syntax and will not work)
   - If none of the above provide a path, ask the user

2. **Resolve the date range** from arguments (default: today).

3. **Read the daily index file(s)** (`{date}.md`) from the vault root for each date in the range. These list all sessions grouped by project with links like `[[project/2026-02-13_1006|10:06]]`.

4. **Read every session file** referenced in the daily index(es). Session files are at `{vault}/{project}/{date}_{time}.md` and contain:
   - Frontmatter: date, session_id, project, start_time, duration, tags
   - Alternating `[!user]+` (prompts) and `[!claude]-` (responses) callouts

5. **Analyze and synthesize** across all sessions:
   - Group by project
   - For each project: summarize what was worked on, key decisions, outcomes
   - Identify cross-project patterns or recurring themes
   - Note any important technical decisions or architectural choices
   - Highlight corrections or course-changes ("Claude suggested X, user pushed back to Y")

6. **Write the synopsis** to `{vault}/Synopsis-{date}.md` (or `Synopsis-{startDate}-to-{endDate}.md` for ranges). Use this structure:

```markdown
# Synopsis: Claude Code Sessions - {date(s)}

> Retrospective of all Claude Code sessions -- what was built, decided, and learned.

---

tags: #claude-session #tag1 #tag2 ...

---

## Quick Navigation

- [[#Project Name]]
  - [[#Session Title 1]]
  - [[#Session Title 2]]
- [[#Another Project]]
  - [[#Session Title 3]]
- [[#Recurring Patterns]]

---

## Overview

| Track | Focus |
|-------|-------|
| **{project}** ({N} sessions) | One-line summary |

---

## {Project Name}

**{Brief title} ({time})** -- What happened in this session. Key decisions, outcomes, pushback.

{Repeat for each notable session or group of related sessions}

---

## Recurring Patterns

{Numbered list of patterns observed across sessions}
```

### Tags
- Collect **all unique tags** from every session file's frontmatter `tags:` array
- Deduplicate and list them as Obsidian hashtags (e.g. `#claude-session #coding #claude-hooks`)
- Place them between two `---` separators, right after the intro quote

### Quick Navigation
- List every project as a top-level `[[#heading]]` link
- Under each project, list every session title as a nested `[[#heading]]` link
- Include `[[#Recurring Patterns]]` at the end
- These must use Obsidian `[[#heading]]` syntax so they work as clickable links in Obsidian

## Rules

- **Read every session file** — don't skip any. The synopsis should cover all work.
- **Be concise but specific** — mention actual file names, class names, commands when relevant. Not vague summaries.
- **Capture the human decisions** — when the user corrected Claude, changed direction, or expressed a preference, that's important context.
- **Don't fabricate** — only write about what's actually in the session logs.
- **Use em-dashes** (`--`) as separator in narrative text, matching the existing synopsis style.
- **Keep it scannable** — tables for comparisons, bold for session titles with timestamps, short paragraphs.
- If a session is trivial (1 prompt, quick answer), it can be a single line rather than a full paragraph.
- The synopsis is a **working document for the user**, not a report for someone else. Be direct.
