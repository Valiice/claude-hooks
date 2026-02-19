# Claude Code Hooks

A [Claude Code](https://docs.anthropic.com/en/docs/claude-code) plugin for Windows that provides:

- **Obsidian logging** — every prompt and response is logged to Obsidian as a nicely-formatted session note with frontmatter, callouts, and a daily index
- **Windows notifications** — balloon notifications when Claude finishes a task or needs your attention

## Prerequisites

- Windows 10/11
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code)
- [Obsidian](https://obsidian.md/) (for the logging hooks)

## Install (Plugin — Recommended)

Install the plugin from the marketplace or add it as a custom plugin, then run the setup command:

```
/setup-obsidian-hooks
```

The setup command will:

1. Ask for your Obsidian vault path
2. Ask about notification and git-push preferences
3. Create `.claude/claude-hooks.local.md` settings file
4. Install `claude-sessions.css` to your Obsidian vault's snippets folder
5. Test notifications
6. Warn if legacy hooks exist in `settings.json`

No Go installation required — the pre-built binaries are included in the plugin.

## Install (Legacy Installer)

> **Note:** The legacy installer is deprecated. Prefer the plugin installation above.

```powershell
git clone https://github.com/EdwardKerckhof/claude-hooks.git
cd claude-hooks
.\install.ps1
```

The installer will:

1. Ask for your Obsidian vault path
2. Set the `CLAUDE_VAULT` environment variable (user-level, persistent)
3. Copy pre-built Go binaries to `~/.claude/hooks/`
4. Copy skills to `~/.claude/skills/`
5. Install `claude-sessions.css` to your Obsidian vault's snippets folder
6. Merge hooks config into `~/.claude/settings.json`

## Configuration

Settings are stored in `.claude/claude-hooks.local.md` (YAML frontmatter). Run `/setup-obsidian-hooks` to configure, or create the file manually:

```markdown
---
vault_path: "C:\\Obsidian\\MyVault\\Claude"
skip_when_focused: true
git_auto_push: false
---
```

| Setting | Description | Default |
|---------|-------------|---------|
| `vault_path` | Path to the Obsidian folder where session logs are written | *(set during setup)* |
| `skip_when_focused` | Skip notifications when terminal is focused | `true` |
| `git_auto_push` | Auto-commit and push vault changes after each response | `false` |

**Settings file locations** (highest priority first):
1. `$CLAUDE_PROJECT_DIR/.claude/claude-hooks.local.md` (project-level)
2. `~/.claude/claude-hooks.local.md` (user-global)

The `CLAUDE_VAULT` environment variable is still supported for backward compatibility and takes priority over settings files.

## Architecture

The plugin uses two standalone Go binaries:

| Binary | Purpose | Commands | External Deps |
|--------|---------|----------|---------------|
| `claude-notify.exe` | Desktop notifications | `--title`, `--message` flags | `beeep` |
| `claude-obsidian.exe` | Session logging | `log-prompt`, `log-response` subcommands | None (stdlib only) |

Source code is in `go-hooks/cmd/notify/` and `go-hooks/cmd/obsidian/`. Internal packages (`internal/hookdata/`, `internal/obsidian/`, `internal/session/`, `internal/settings/`) handle parsing, formatting, and configuration.

### Rebuilding (for contributors)

If you modify the Go source, rebuild the binaries:

```powershell
cd go-hooks
go build -ldflags="-s -w" -o ../bin/claude-notify.exe ./cmd/notify
go build -ldflags="-s -w" -o ../bin/claude-obsidian.exe ./cmd/obsidian
```

### Running tests

```powershell
cd go-hooks
go test ./... -v
```

### Legacy PowerShell hooks

The `hooks-legacy/` folder contains the original PowerShell implementations. These are kept as reference but are no longer used. The Go binaries are significantly faster (~5ms startup vs ~500ms for PowerShell).

## Skills

Skills are available as slash commands in any Claude Code session when the plugin is installed.

| Skill | Command | Description |
|-------|---------|-------------|
| synopsis | `/synopsis` | Generates a retrospective of your Claude Code sessions from the Obsidian logs. Supports arguments: `/synopsis`, `/synopsis 2026-02-12`, `/synopsis week` |

## Commands

| Command | Description |
|---------|-------------|
| `/setup-obsidian-hooks` | First-time setup — configure vault path, preferences, and install CSS snippet |

## Obsidian CSS snippet

`assets/claude-sessions.css` styles the custom callouts (`[!user]`, `[!claude]`, `[!plan]`) used in the session notes.

The setup command installs this automatically. To enable it in Obsidian:

**Settings > Appearance > CSS snippets** > enable **claude-sessions**

If you installed manually, copy `assets/claude-sessions.css` to your vault's `.obsidian/snippets/` folder.
