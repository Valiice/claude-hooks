#Requires -Version 5.1
<#
.SYNOPSIS
    [DEPRECATED] Installs Claude Code hooks for Obsidian logging and Windows notifications.
.DESCRIPTION
    This installer is deprecated. Please install claude-hooks as a Claude Code plugin instead.
    See README.md for plugin installation instructions, then run /setup-obsidian-hooks.

    Legacy behavior (still works):
    - Prompts for the Obsidian vault path
    - Sets CLAUDE_VAULT as a user-level environment variable
    - Copies pre-built Go binaries (claude-notify.exe, claude-obsidian.exe) to ~/.claude/hooks/
    - Copies skills to ~/.claude/skills/
    - Merges hooks config into ~/.claude/settings.json (preserving existing settings)
#>

param(
    [string]$VaultPath
)

$ErrorActionPreference = "Stop"

Write-Host "`n=== Claude Code Hooks Installer ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "[DEPRECATED] This installer is deprecated." -ForegroundColor Yellow
Write-Host "             Install claude-hooks as a Claude Code plugin instead." -ForegroundColor Yellow
Write-Host "             See README.md for plugin installation instructions." -ForegroundColor Yellow
Write-Host "             After installing the plugin, run /setup-obsidian-hooks" -ForegroundColor Yellow
Write-Host ""

# 1. Prompt for Obsidian vault path
if (-not $VaultPath) {
    if ($env:CLAUDE_VAULT) {
        $change = Read-Host "Vault path is '$env:CLAUDE_VAULT'. Press Enter to keep, or type a new path"
        if ($change) {
            $VaultPath = $change
        } else {
            $VaultPath = $env:CLAUDE_VAULT
        }
    } else {
        $VaultPath = Read-Host "Obsidian vault path for Claude logs (e.g. C:\Obsidian\MyVault\Claude)"
        if (-not $VaultPath) {
            Write-Host "[ERROR] Vault path is required. Pass it as -VaultPath or enter it at the prompt." -ForegroundColor Red
            exit 1
        }
    }
}

# Normalize path (remove trailing slash)
$VaultPath = $VaultPath.TrimEnd("\", "/")

# Create vault dir if it doesn't exist
if (-not (Test-Path $VaultPath)) {
    Write-Host "Creating vault directory: $VaultPath" -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $VaultPath -Force | Out-Null
}

# 2. Set CLAUDE_VAULT environment variable (user-level, persistent)
[Environment]::SetEnvironmentVariable("CLAUDE_VAULT", $VaultPath, "User")
$env:CLAUDE_VAULT = $VaultPath
Write-Host "[OK] Set CLAUDE_VAULT = $VaultPath" -ForegroundColor Green

# 3. Ensure ~/.claude/hooks/ exists
$claudeDir = Join-Path $env:USERPROFILE ".claude"
$hooksDir = Join-Path $claudeDir "hooks"
if (-not (Test-Path $hooksDir)) {
    New-Item -ItemType Directory -Path $hooksDir -Force | Out-Null
    Write-Host "[OK] Created $hooksDir" -ForegroundColor Green
}

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$binDir = Join-Path $repoRoot "go-hooks\bin"

# 4. Copy pre-built Go binaries
$binaries = @("claude-notify.exe", "claude-obsidian.exe")
foreach ($bin in $binaries) {
    $src = Join-Path $binDir $bin
    if (-not (Test-Path $src)) {
        Write-Host "[ERROR] Pre-built binary not found: $src" -ForegroundColor Red
        Write-Host "        Run 'go build' in go-hooks/ first, or check that bin/ contains the .exe files." -ForegroundColor Red
        exit 1
    }
    Copy-Item -Path $src -Destination (Join-Path $hooksDir $bin) -Force
    Write-Host "[OK] Installed $bin" -ForegroundColor Green
}

# 5. Copy skills from repo's skills/ folder
$repoSkillsDir = Join-Path $repoRoot "skills"
$skillsDir = Join-Path $claudeDir "skills"
if (Test-Path $repoSkillsDir) {
    $skillFolders = Get-ChildItem -Path $repoSkillsDir -Directory
    foreach ($folder in $skillFolders) {
        $destFolder = Join-Path $skillsDir $folder.Name
        if (-not (Test-Path $destFolder)) {
            New-Item -ItemType Directory -Path $destFolder -Force | Out-Null
        }
        $skillFiles = Get-ChildItem -Path $folder.FullName -File
        foreach ($file in $skillFiles) {
            Copy-Item -Path $file.FullName -Destination (Join-Path $destFolder $file.Name) -Force
        }
        Write-Host "[OK] Installed skill: $($folder.Name)" -ForegroundColor Green
    }
}

# 6. Build hooks config with absolute paths for this machine
# Use forward slashes so paths work in both PowerShell and bash (MSYS/Git Bash)
$notifyExe = (Join-Path $hooksDir "claude-notify.exe") -replace '\\', '/'
$obsidianExe = (Join-Path $hooksDir "claude-obsidian.exe") -replace '\\', '/'

$hooksConfig = @{
    "Stop" = @(
        @{
            matcher = "*"
            hooks = @(
                @{ type = "command"; command = "$notifyExe --message `"Waiting for you!`"" }
                @{ type = "command"; command = "$obsidianExe log-response" }
            )
        }
    )
    "UserPromptSubmit" = @(
        @{
            hooks = @(
                @{ type = "command"; command = "$obsidianExe log-prompt" }
            )
        }
    )
    "Notification" = @(
        @{
            matcher = "*"
            hooks = @(
                @{ type = "command"; command = "$notifyExe --message `"Needs your attention!`"" }
            )
        }
    )
}

# 7. Merge into existing settings.json or create new one
$settingsPath = Join-Path $claudeDir "settings.json"
$utf8 = New-Object System.Text.UTF8Encoding($false)

if (Test-Path $settingsPath) {
    $settings = Get-Content $settingsPath -Raw | ConvertFrom-Json
    Write-Host "[OK] Read existing settings.json" -ForegroundColor Green
} else {
    $settings = [PSCustomObject]@{}
    Write-Host "[OK] Creating new settings.json" -ForegroundColor Green
}

# Replace hooks on the PSCustomObject directly (preserves key order from original file)
if ($settings.PSObject.Properties['hooks']) {
    $settings.hooks = $hooksConfig
} else {
    $settings | Add-Member -NotePropertyName 'hooks' -NotePropertyValue $hooksConfig
}

# Serialize and reformat (PS 5.1's ConvertTo-Json uses inconsistent indentation)
$json = $settings | ConvertTo-Json -Depth 10
$level = 0
$formatted = @()
foreach ($line in ($json -split "`n")) {
    $trimmed = $line.Trim()
    if (-not $trimmed) { continue }
    $trimmed = $trimmed -replace ':\s{2,}', ': '
    if ($trimmed -match '^[\}\]]') { $level = [Math]::Max(0, $level - 1) }
    $formatted += ('  ' * $level) + $trimmed
    if ($trimmed -match '[\{\[]\s*$') { $level++ }
}
$json = $formatted -join "`n"

[System.IO.File]::WriteAllText($settingsPath, $json, $utf8)
Write-Host "[OK] Updated settings.json with hooks config" -ForegroundColor Green

# 8. Clean up old claude-hooks.exe if present
$oldExe = Join-Path $hooksDir "claude-hooks.exe"
if (Test-Path $oldExe) {
    Remove-Item $oldExe -Force
    Write-Host "[OK] Removed old claude-hooks.exe" -ForegroundColor Green
}

# 9. Install Obsidian CSS snippet
$cssSource = Join-Path $repoRoot "claude-sessions.css"
if (Test-Path $cssSource) {
    # Walk up from vault path to find .obsidian/snippets/
    $vaultRoot = $VaultPath
    while ($vaultRoot -and -not (Test-Path (Join-Path $vaultRoot ".obsidian"))) {
        $vaultRoot = Split-Path $vaultRoot -Parent
    }
    if ($vaultRoot) {
        $snippetsDir = Join-Path $vaultRoot ".obsidian\snippets"
        if (-not (Test-Path $snippetsDir)) {
            New-Item -ItemType Directory -Path $snippetsDir -Force | Out-Null
        }
        Copy-Item -Path $cssSource -Destination (Join-Path $snippetsDir "claude-sessions.css") -Force
        Write-Host "[OK] Installed claude-sessions.css to Obsidian snippets" -ForegroundColor Green
        Write-Host "     Enable it in Obsidian: Settings > Appearance > CSS snippets" -ForegroundColor Gray
    } else {
        Write-Host "[SKIP] Could not find .obsidian folder - copy claude-sessions.css manually" -ForegroundColor Yellow
    }
}

# 10. Configure hooks via config.json
$configPath = Join-Path $hooksDir "config.json"
$hooksCfg = @{ skip_when_focused = $true; git_auto_push = $false }

if (Test-Path $configPath) {
    try {
        $existing = Get-Content $configPath -Raw | ConvertFrom-Json
        if ($null -ne $existing.skip_when_focused) { $hooksCfg.skip_when_focused = [bool]$existing.skip_when_focused }
        if ($null -ne $existing.git_auto_push) { $hooksCfg.git_auto_push = [bool]$existing.git_auto_push }
        Write-Host "[OK] Read existing config.json" -ForegroundColor Green
    } catch {
        Write-Host "[WARN] Could not parse config.json, using defaults" -ForegroundColor Yellow
    }
}

# Prompt for skip_when_focused
if ($hooksCfg.skip_when_focused) {
    $change = Read-Host "Skip notifications when terminal is focused (currently: yes). Press Enter to keep, or type 'n' to disable"
    if ($change -eq "n" -or $change -eq "N") {
        $hooksCfg.skip_when_focused = $false
    }
} else {
    $change = Read-Host "Skip notifications when terminal is focused? (y/n, currently: no)"
    if ($change -eq "y" -or $change -eq "Y") {
        $hooksCfg.skip_when_focused = $true
    }
}

# Prompt for git_auto_push
if ($hooksCfg.git_auto_push) {
    $change = Read-Host "Git auto-push is enabled. Press Enter to keep, or type 'n' to disable"
    if ($change -eq "n" -or $change -eq "N") {
        $hooksCfg.git_auto_push = $false
        Write-Host "[OK] Disabled git auto-push" -ForegroundColor Green
    }
} else {
    $gitPush = Read-Host "Enable git auto-push for vault? (y/n)"
    if ($gitPush -eq "y" -or $gitPush -eq "Y") {
        $hooksCfg.git_auto_push = $true
        Write-Host "[OK] Enabled git auto-push" -ForegroundColor Green
        Write-Host "     Vault changes will be committed and pushed after each response." -ForegroundColor Gray
        Write-Host "     Make sure your vault has a git remote configured." -ForegroundColor Gray
    }
}

# Write config.json
$cfgJson = $hooksCfg | ConvertTo-Json
[System.IO.File]::WriteAllText($configPath, $cfgJson, $utf8)
Write-Host "[OK] Updated config.json" -ForegroundColor Green

# Migrate: remove legacy CLAUDE_VAULT_GIT_PUSH env var if present
if ($env:CLAUDE_VAULT_GIT_PUSH) {
    [Environment]::SetEnvironmentVariable("CLAUDE_VAULT_GIT_PUSH", $null, "User")
    Remove-Item Env:\CLAUDE_VAULT_GIT_PUSH -ErrorAction SilentlyContinue
    Write-Host "[OK] Removed legacy CLAUDE_VAULT_GIT_PUSH env var (migrated to config.json)" -ForegroundColor Green
}

Write-Host "`n=== Installation complete ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Hooks installed to: $hooksDir" -ForegroundColor White
Write-Host "Vault path:         $VaultPath" -ForegroundColor White
Write-Host ""
Write-Host "Start a new Claude Code session to activate the hooks." -ForegroundColor White
Write-Host ""
