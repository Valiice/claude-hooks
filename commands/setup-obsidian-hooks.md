---
name: setup-obsidian-hooks
description: First-time setup for claude-hooks plugin — configure Obsidian vault path, notification preferences, and install CSS snippets.
allowed-tools: ["Read", "Write", "Edit", "Bash", "Glob", "AskUserQuestion"]
---

# Setup Obsidian Hooks

Guide the user through first-time configuration of the claude-hooks plugin.

## Step 1: Check for existing configuration

Check if `.claude/claude-hooks.local.md` already exists in the current project directory. If it does, read it and show the current settings. Ask if the user wants to reconfigure.

Also check if the `CLAUDE_VAULT` environment variable is set (run `echo $CLAUDE_VAULT` in Bash). If set, mention it as the currently active vault path.

## Step 2: Ask for Obsidian vault path

Ask the user for their Obsidian vault path — the folder where Claude session logs should be written (e.g., `C:\Obsidian\MyVault\Claude` or `D:\Notes\Claude`).

If `CLAUDE_VAULT` is already set, offer to keep it or change it.

## Step 3: Ask about preferences

Ask the user:
1. **Skip notifications when terminal is focused?** (default: yes) — suppresses toast notifications when you're actively looking at the terminal
2. **Enable git auto-push for vault?** (default: no) — automatically commits and pushes vault changes after each Claude response. Requires git remote configured in vault.

## Step 4: Create settings file

Create `.claude/claude-hooks.local.md` with the user's preferences:

```markdown
---
vault_path: "<user's vault path>"
skip_when_focused: true
git_auto_push: false
---

# Claude Hooks Configuration

Plugin settings for [claude-hooks](https://github.com/EdwardKerckhof/claude-hooks).
This file is read by the plugin's Go binaries at runtime.

To reconfigure, run `/setup-obsidian-hooks` again or edit this file directly.
```

Ask the user if they also want a **global** settings file at `~/.claude/claude-hooks.local.md` (applies to all projects where no project-level file exists). If yes, create it with the same content at that path.

## Step 5: Create vault directory

If the vault path doesn't exist yet, create it:

```bash
mkdir -p "<vault_path>"
```

## Step 6: Install CSS snippet

Check if the vault has an `.obsidian` folder (walk up from vault path to find it). If found, copy the CSS snippet:

```bash
# Find .obsidian folder (may be at vault root or parent)
# Copy CSS to .obsidian/snippets/
cp "${CLAUDE_PLUGIN_ROOT}/assets/claude-sessions.css" "<vault_root>/.obsidian/snippets/claude-sessions.css"
```

Tell the user to enable it in Obsidian: **Settings > Appearance > CSS snippets > enable "claude-sessions"**.

If `.obsidian` folder is not found, tell the user to manually copy the CSS file later.

## Step 7: Test notification

Run a test notification:

```bash
"${CLAUDE_PLUGIN_ROOT}/bin/claude-notify.exe" --message "Claude hooks configured!"
```

If it works, tell the user setup is complete.

## Step 8: Check for legacy hooks

Check if `~/.claude/settings.json` contains hooks pointing to `claude-obsidian.exe` or `claude-notify.exe` in the old `~/.claude/hooks/` location. If found, warn the user:

> **Legacy hooks detected** in `~/.claude/settings.json`. Since you're now using the plugin, you should remove the old hook entries from settings.json to avoid duplicate logging. The old binaries in `~/.claude/hooks/` can also be removed.

## Step 9: Summary

Print a summary of what was configured:
- Vault path
- Settings file location(s)
- CSS snippet status
- Notification test result
- Any warnings about legacy config

Remind the user to **restart Claude Code** for hooks to take effect if this is a fresh install.
