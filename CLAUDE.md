# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A Claude Code **plugin** for Windows that logs every session to Obsidian and shows desktop notifications. Two standalone Go binaries, pre-built and committed to the repo. Can be installed as a plugin (recommended) or via the legacy `install.ps1` installer.

## Repo contents

- `.claude-plugin/plugin.json` — plugin manifest
- `hooks/hooks.json` — plugin hook configuration (auto-discovered)
- `bin/` — pre-built binaries (committed, no Go required to install)
- `commands/` — slash commands (`/setup-obsidian-hooks`)
- `skills/` — Claude Code skills (synopsis generator)
- `assets/` — Obsidian CSS snippet for callout styling
- `go-hooks/` — Go source code for both binaries
  - `cmd/notify/` — `claude-notify.exe` entry point (desktop notifications via `beeep`)
  - `cmd/obsidian/` — `claude-obsidian.exe` entry point (session logging, stdlib only)
  - `internal/hookdata/` — stdin JSON parsing (shared types)
  - `internal/obsidian/` — Obsidian formatting, frontmatter, daily index, tag stripping
  - `internal/session/` — session-to-file mapping via temp files
  - `internal/settings/` — settings file parser (.claude/claude-hooks.local.md)
  - `internal/config/` — config loader (config.json + settings overlay)
  - `bin/` — legacy binary location (use root `bin/` instead)
- `hooks-legacy/` — legacy PowerShell scripts (kept as reference)
- `install.ps1` — legacy installer (deprecated, use plugin install instead)

## Architecture

Two independent Go binaries:

| Binary | Source | Purpose | Deps |
|--------|--------|---------|------|
| `claude-notify.exe` | `cmd/notify/` | Desktop notifications | `beeep` |
| `claude-obsidian.exe` | `cmd/obsidian/` | Session logging | stdlib + `internal/settings` |

Both binaries have `defer recover()` in `main()` — they must never block Claude Code.

### Hook data flow (plugin mode)

```
Claude Code event
  -> ${CLAUDE_PLUGIN_ROOT}/bin/claude-obsidian.exe log-prompt
    -> stdin: JSON { session_id, cwd, prompt }
    -> reads/writes: %TEMP%\claude_session_<id>.txt (session mapping)
    -> writes: <vault>/<project>/<date>_<time>.md (Obsidian note)

  -> ${CLAUDE_PLUGIN_ROOT}/bin/claude-notify.exe --message "..."
    -> shows Windows toast notification
```

### Configuration

Settings are read from `.claude/claude-hooks.local.md` (YAML frontmatter):

```yaml
---
vault_path: "C:\\Obsidian\\MyVault\\Claude"
skip_when_focused: true
git_auto_push: false
---
```

Search order (highest priority first):
1. `CLAUDE_VAULT` env var (backward compat)
2. `$CLAUDE_PROJECT_DIR/.claude/claude-hooks.local.md` (project-level)
3. `~/.claude/claude-hooks.local.md` (user-global)
4. `~/.claude/hooks/config.json` (legacy config)

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

`log-prompt` creates `%TEMP%\claude_session_<session_id>.txt` mapping the session to its Obsidian file path and prompt counter. `log-response` reads this to find where to append. Stale files (>24h) are cleaned up automatically.

### Key design constraints

- Hooks **must never block Claude Code** — both binaries use `defer recover()` and exit silently on errors
- Hooks receive JSON on **stdin** (parsed via `internal/hookdata`)
- `SanitizeProject` strips leading dots (Obsidian hides dotfolders) and illegal path characters
- `StripSystemTags` removes `<system-reminder>`, `<task-notification>`, etc. before logging
- `readTranscript` walks backwards through JSONL (up to 50 lines) to find the last assistant response

## Build

```powershell
cd go-hooks
go build -ldflags="-s -w" -o ../bin/claude-notify.exe ./cmd/notify
go build -ldflags="-s -w" -o ../bin/claude-obsidian.exe ./cmd/obsidian
```

After rebuilding, the binaries in `bin/` are ready for the plugin. No copy step needed.

## Tests

```powershell
cd go-hooks
go test ./... -v
```

Tests cover: settings parser, obsidian formatting, frontmatter, truncation, tag stripping, and daily index generation.

## Install (Plugin)

Install as a Claude Code plugin from the marketplace or a custom registry. Then run `/setup-obsidian-hooks` for first-time configuration.

## Install (Legacy)

```powershell
.\install.ps1
```

Copies pre-built binaries from `go-hooks/bin/` to `~/.claude/hooks/`, installs skills, CSS snippet, sets `CLAUDE_VAULT` env var, and merges hooks config into `~/.claude/settings.json`.

## Verify

1. Plugin: `.claude-plugin/plugin.json` exists, `hooks/hooks.json` uses `${CLAUDE_PLUGIN_ROOT}` paths
2. Binaries exist in `bin/` at repo root
3. `bin/claude-notify.exe --message "Test"` — toast appears
4. Send a prompt in Claude Code — check vault for session file
