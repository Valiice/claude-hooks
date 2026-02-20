# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

Windows hooks for Claude Code that log every session to Obsidian and show desktop notifications. Two standalone Go binaries, pre-built and committed to the repo. An installer (`install.ps1`) copies them to `~/.claude/hooks/` and wires up `settings.json`.

## Repo contents

- `go-hooks/` — Go source code for both binaries
  - `cmd/notify/` — `claude-notify.exe` entry point (desktop notifications via `beeep`)
  - `cmd/obsidian/` — `claude-obsidian.exe` entry point (session logging with stats & git)
  - `internal/hookdata/` — stdin JSON parsing (shared types)
  - `internal/obsidian/` — Obsidian formatting, frontmatter, daily index, auto-reports, tag stripping
  - `internal/session/` — session-to-file mapping via temp files (5-line format: path, promptNum, branch, startHash, cwd)
  - `internal/transcript/` — JSONL transcript parsing for tool counts, token usage, cost estimation
  - `internal/gitctx/` — git context capture (branch, hash, commits since)
  - `internal/config/` — config loader for `~/.claude/hooks/config.json`
  - `internal/focus/` — Windows focus detection (suppresses notifications when terminal is focused)
  - `internal/gitsync/` — vault git auto-commit/push after each response
  - `bin/` — pre-built binaries (committed, no Go required to install)
- `hooks/` — legacy PowerShell scripts (kept as reference, not used by installer)
- `skills/` — Claude Code skills
  - `synopsis/` — session summary generator
  - `bookmark/` — mark sessions as bookmarked
  - `weekly/` — weekly dashboard generator
  - `monthly/` — monthly retrospective generator
  - `project-stats/` — per-project statistics
  - `categorize/` — auto-categorize sessions by type
  - `archive/` — archive old sessions
- `claude-sessions.css` — Obsidian CSS snippet for callout styling (user, claude, plan, git)
- `install.ps1` — one-step installer

## Architecture

Two independent Go binaries with zero shared code:

| Binary | Source | Purpose | Deps |
|--------|--------|---------|------|
| `claude-notify.exe` | `cmd/notify/` | Desktop notifications (skip when focused) | `beeep`, `x/sys/windows`, `config`, `focus` |
| `claude-obsidian.exe` | `cmd/obsidian/` | Session logging + stats + git + vault sync | `config`, `gitsync`, `gitctx`, `obsidian`, `session`, `transcript` |

Both binaries have `defer recover()` in `main()` — they must never block Claude Code.

### Hook data flow

```
Claude Code event
  -> C:\Users\<user>\.claude\hooks\claude-obsidian.exe log-prompt
    -> stdin: JSON { session_id, cwd, prompt }
    -> captures git context (branch + HEAD hash)
    -> reads/writes: %TEMP%\claude_session_<id>.txt (session mapping)
    -> writes: %CLAUDE_VAULT%\<project>\<date>_<time>.md (Obsidian note)

  -> C:\Users\<user>\.claude\hooks\claude-obsidian.exe log-response
    -> stdin: JSON { session_id, transcript_path }
    -> parses transcript JSONL for tool counts, tokens, cost
    -> detects git commits since session start
    -> updates frontmatter with stats + appends response/stats/commits
    -> rebuilds daily index with tool count and cost
    -> rebuilds weekly/monthly stats reports if stale (at most once/day)

  -> C:\Users\<user>\.claude\hooks\claude-obsidian.exe log-response (cont.)
    -> git sync: if git_auto_push enabled, commits and pushes vault changes
       -> acquires .git/claude-sync.lock (stale after 5min)
       -> git add -A && git commit && git push (60s timeout, best-effort)
       -> if push rejected (remote ahead): git pull --rebase, then retry push
       -> if pull fails (conflict): git rebase --abort (restores clean state)

  -> C:\Users\<user>\.claude\hooks\claude-notify.exe --message "..."
    -> checks if terminal is focused (walks process tree via Win32 API)
    -> if skip_when_focused=true and terminal is foreground: skip
    -> otherwise: shows Windows toast notification
```

### Stdin JSON shapes

**UserPromptSubmit** (received by `claude-obsidian.exe log-prompt`):
```json
{ "session_id": "...", "cwd": "C:\\...", "prompt": "user's message with system tags" }
```

**Stop** (received by `claude-obsidian.exe log-response`):
```json
{ "session_id": "...", "transcript_path": "C:\\...\\<id>.jsonl" }
```

### Session state

`log-prompt` creates `%TEMP%\claude_session_<session_id>.txt` with 5 lines: file path, prompt counter, branch, start hash, and cwd. `log-response` reads this to find where to append and to detect git commits. Stale files (>24h) are cleaned up automatically. The reader handles old 2-line format gracefully.

### Frontmatter fields

Session files include YAML frontmatter with these fields (stats fields omitted when zero/empty):

```yaml
date: 2026-02-17
session_id: abc-123
project: Coding
start_time: 14:30
duration: 25min
branch: feature/new-thing
model: opus
tools:
  Edit: 12
  Read: 15
tokens_in: 45230
tokens_out: 12840
cache_read: 125000
cache_creation: 14829
estimated_cost: "$0.23"
files_touched:
  - cmd/main.go
commits:
  - a1b2c3d Add feature
tags:
  - claude-session
  - coding
```

### Auto-generated reports

After each `log-response`, weekly and monthly stats reports are rebuilt if stale (file doesn't exist or was last modified before today). This means at most **one rebuild per day** — negligible overhead.

- **Weekly**: `Weekly-{start}-to-{end}.md` in vault root. Covers Monday of current week through today. Contains overview, projects, tool usage, most touched files, streaks, cost analysis, and daily breakdown tables.
- **Monthly**: `Monthly-{YYYY-MM}.md` in vault root. Covers 1st of month through today. Same as weekly plus weekly breakdown and tool usage evolution tables.

Both have `auto_generated: true` in frontmatter and a note pointing to `/weekly` and `/monthly` skills for narrative versions. `ScanSessions` walks all project subdirs and parses frontmatter from session files within the date range.

### Config file

`~/.claude/hooks/config.json` controls runtime behavior:

```json
{
  "skip_when_focused": true,
  "git_auto_push": false
}
```

| Field | Default | Effect |
|-------|---------|--------|
| `skip_when_focused` | `true` | Suppress toast notifications when terminal window is focused |
| `git_auto_push` | `false` | Auto-commit and push vault changes after each `log-response` |

Config is loaded by `internal/config.Load()` — returns defaults on missing/invalid file.

### Key design constraints

- Hooks **must never block Claude Code** — both binaries use `defer recover()` and exit silently on errors
- Hooks receive JSON on **stdin** (parsed via `internal/hookdata`)
- `SanitizeProject` strips leading dots (Obsidian hides dotfolders) and illegal path characters
- `StripSystemTags` removes `<system-reminder>`, `<task-notification>`, etc. before logging
- `readTranscript` walks backwards through JSONL (up to 50 lines) to find the last assistant response
- `ParseTranscript` does a full forward scan for tool/token/cost stats (skips sidechain messages)
- `gitctx.Capture` has a 2s timeout — returns empty on any error
- Model-aware pricing (Opus $5/$25, Sonnet $3/$15, Haiku $1/$5) with cache token accounting (reads at 0.1x, writes at 1.25x input rate). Model detected from transcript `message.model` field; defaults to Sonnet if unknown
- `BuildFrontmatter` accepts a `FrontmatterData` struct; `UpdateFrontmatterStats` patches existing frontmatter in-place
- `RebuildWeeklyStatsIfStale` / `RebuildMonthlyStatsIfStale` — auto-generate stats reports at most once per day; checks file mtime to skip if already rebuilt today
- `gitsync.SyncIfEnabled` uses a file lock (`.git/claude-sync.lock`, stale after 5min) and 60s timeout. If `git push` is rejected (remote ahead), it runs `git pull --rebase` then retries; on pull conflict, runs `git rebase --abort` to restore clean state
- `focus.TerminalIsFocused` walks the process tree via Win32 `CreateToolhelp32Snapshot` to check if the foreground window belongs to an ancestor process

## Build

```powershell
cd go-hooks
go build -ldflags="-s -w" -o bin/claude-notify.exe ./cmd/notify
go build -ldflags="-s -w" -o bin/claude-obsidian.exe ./cmd/obsidian
```

After rebuilding, copy to `~/.claude/hooks/` or re-run `install.ps1`.

## Tests

```powershell
cd go-hooks
go test ./... -v
```

72 tests across 6 packages:
- `internal/obsidian/` — 33 tests: formatting, frontmatter (with stats/branch/files/model/cache), truncation, tag stripping, daily index (with tools/cost), stats line, commits entry, frontmatter update, session scanning, report building (weekly/monthly), staleness checks, duration parsing, week start calculation
- `internal/transcript/` — 10 tests: tool counts, token sums, file dedup, cost calc (Sonnet default), model detection, Opus pricing, cache cost reduction, empty file, malformed lines, sidechain skip
- `internal/gitctx/` — 4 tests: not-a-repo, valid repo, commits-since, empty hash
- `internal/gitsync/` — tests: sync enabled/disabled, lock acquisition, stale lock cleanup, no-op on clean tree, git root detection, push-rejected-then-recovered, pull-conflict-aborts-cleanly
- `internal/config/` — tests: defaults, load from file, missing file, invalid JSON
- `internal/focus/` — tests: ancestor walking, process map, cycle detection

## Install

```powershell
.\install.ps1
```

Copies pre-built binaries from `go-hooks/bin/` to `~/.claude/hooks/`, installs all skills, CSS snippet, sets `CLAUDE_VAULT` env var, merges hooks config into `~/.claude/settings.json`, and creates/updates `~/.claude/hooks/config.json` (prompts for `skip_when_focused` and `git_auto_push`).

## Verify

1. Both binaries exist in `~/.claude/hooks/`
2. `settings.json` hooks point to `claude-notify.exe` and `claude-obsidian.exe`
3. `claude-notify.exe --message "Test"` — toast appears
4. Send a prompt in Claude Code — check `%CLAUDE_VAULT%\<project>\` for session file
5. After a response, session file should have `model:`, `tools:`, `tokens_in:`, `cache_read:`, `cache_creation:`, `estimated_cost:` in frontmatter
6. Daily index should show tool count and cost per session
7. In a git repo, session file should show `branch:` and `[!git]` callout if commits were made
8. Auto-generated `Weekly-{start}-to-{end}.md` and `Monthly-{YYYY-MM}.md` appear in vault root after first response of the day
