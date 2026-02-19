function Format-CalloutContent([string]$text) {
    $lines = $text -split "`n"
    $output = @()
    foreach ($line in $lines) {
        $output += "> $line"
    }
    return ($output -join "`n")
}

try {
    $stdinStream = [Console]::OpenStandardInput()
    $reader = New-Object System.IO.StreamReader($stdinStream, [System.Text.Encoding]::UTF8)
    $rawInput = $reader.ReadToEnd()
    $reader.Close()
    $json = $rawInput | ConvertFrom-Json
    $sessionId = $json.session_id
    $cwd = $json.cwd
    $prompt = $json.prompt

    if (-not $prompt) { exit 0 }

    # Strip system-injected content - keep only what the user typed
    $prompt = $prompt -replace '(?s)<system-reminder>.*?</system-reminder>', ''
    $prompt = $prompt -replace '(?s)<task-notification>.*?</task-notification>', ''
    $prompt = $prompt -replace '(?s)<claude-mem-context>.*?</claude-mem-context>', ''
    $prompt = $prompt -replace '(?s)<context-window-budget>.*?</context-window-budget>', ''
    $prompt = $prompt -replace '(?s)<skill-reminders>.*?</skill-reminders>', ''
    $prompt = $prompt -replace '(?s)<local-command-caveat>.*?</local-command-caveat>', ''
    $prompt = $prompt -replace '(?s)<command-name>.*?</command-name>', ''
    $prompt = $prompt -replace '(?s)<command-message>.*?</command-message>', ''
    $prompt = $prompt -replace '(?s)<command-args>.*?</command-args>', ''
    $prompt = $prompt -replace '(?s)<local-command-stdout>.*?</local-command-stdout>', ''
    $prompt = $prompt.Trim()

    if (-not $prompt) { exit 0 }

    # Truncate very long prompts (e.g. pasted code blocks)
    if ($prompt.Length -gt 5000) {
        $prompt = $prompt.Substring(0, 5000) + "`n`n... (truncated, $($prompt.Length) chars total)"
    }

    $vaultDir = if ($env:CLAUDE_VAULT) { $env:CLAUDE_VAULT }
               elseif ([Environment]::GetEnvironmentVariable("CLAUDE_VAULT", "User")) { [Environment]::GetEnvironmentVariable("CLAUDE_VAULT", "User") }
               else { $null }
    if (-not $vaultDir) { exit 0 }  # CLAUDE_VAULT not configured, skip logging
    $claudeProjects = Join-Path $env:USERPROFILE ".claude\projects"
    if (-not (Test-Path $vaultDir)) {
        New-Item -ItemType Directory -Path $vaultDir -Force | Out-Null
    }

    $date = Get-Date -Format "yyyy-MM-dd"
    $time = Get-Date -Format "HH:mm:ss"
    $project = (Split-Path $cwd -Leaf) -replace '[\\/:*?"<>|]', '_'
    $utf8 = New-Object System.Text.UTF8Encoding($false)

    # Use a temp file to track session-to-filename mapping
    $sessionMap = Join-Path $env:TEMP "claude_session_$sessionId.txt"

    # Clean up stale session mapping files (older than 24h)
    Get-ChildItem -Path $env:TEMP -Filter "claude_session_*.txt" -ErrorAction SilentlyContinue |
        Where-Object { $_.LastWriteTime -lt (Get-Date).AddHours(-24) } |
        Remove-Item -Force -ErrorAction SilentlyContinue

    if (Test-Path $sessionMap) {
        $data = [System.IO.File]::ReadAllText($sessionMap, $utf8).Trim() -split "`n"
        $filePath = $data[0]
        $promptNum = [int]$data[1] + 1
        # Update prompt counter
        [System.IO.File]::WriteAllText($sessionMap, "$filePath`n$promptNum", $utf8)
    } else {
        # New session - create project subfolder and readable filename
        $promptNum = 1
        $projectDir = Join-Path $vaultDir $project
        if (-not (Test-Path $projectDir)) {
            New-Item -ItemType Directory -Path $projectDir -Force | Out-Null
        }
        $timeShort = Get-Date -Format "HHmm"
        $fileName = "${date}_${timeShort}.md"
        $filePath = Join-Path $projectDir $fileName

        # Handle rare collision (same project, same minute)
        $counter = 2
        while (Test-Path $filePath) {
            $fileName = "${date}_${timeShort}_${counter}.md"
            $filePath = Join-Path $projectDir $fileName
            $counter++
        }

        # Save mapping + prompt counter for subsequent prompts
        [System.IO.File]::WriteAllText($sessionMap, "$filePath`n1", $utf8)

        # Check for parent session (resumed session linking)
        $resumedFrom = ""
        try {
            $transcriptFiles = Get-ChildItem -Path $claudeProjects -Recurse -Filter "$sessionId.jsonl" -ErrorAction SilentlyContinue
            if ($transcriptFiles) {
                $transcriptFile = $transcriptFiles[0].FullName
                # Read first 20 lines to find parentUuid
                $tLines = [System.IO.File]::ReadAllLines($transcriptFile, $utf8) | Select-Object -First 20
                foreach ($tLine in $tLines) {
                    $tObj = $tLine | ConvertFrom-Json -ErrorAction SilentlyContinue
                    if ($tObj -and $tObj.parentUuid) {
                        $parentId = $tObj.parentUuid
                        # Search Obsidian files for the parent session_id in frontmatter
                        $sessionFiles = Get-ChildItem -Path $vaultDir -Recurse -Filter "*.md" -ErrorAction SilentlyContinue
                        foreach ($sf in $sessionFiles) {
                            $sfContent = [System.IO.File]::ReadAllText($sf.FullName, $utf8)
                            if ($sfContent -match "session_id:\s*$parentId") {
                                $relPath = $sf.FullName.Replace($vaultDir + "\", "").Replace("\", "/").Replace(".md", "")
                                $resumedFrom = $relPath
                                break
                            }
                        }
                        break
                    }
                }
            }
        } catch {
            # Don't block on linking errors
        }

        $startTime = Get-Date -Format "HH:mm"
        $projectTag = $project.ToLower() -replace '\s+', '-'

        $frontmatterLines = @(
            "---"
            "date: $date"
            "session_id: $sessionId"
            "project: $project"
            "start_time: $startTime"
        )
        if ($resumedFrom) {
            $frontmatterLines += "resumed_from: `"[[$resumedFrom]]`""
        }
        $frontmatterLines += @(
            "tags:"
            "  - claude-session"
            "  - $projectTag"
            "---"
        )

        $frontmatter = ($frontmatterLines -join "`n") + "`n`n# Claude Session - $project`n"

        if ($resumedFrom) {
            $parentName = Split-Path $resumedFrom -Leaf
            $frontmatter += "Resumed from [[$resumedFrom|$parentName]]`n"
        }

        $frontmatter += "`n---`n"

        [System.IO.File]::WriteAllText($filePath, $frontmatter, $utf8)
    }

    # Format prompt as Obsidian callout (expanded by default)
    $bt = [char]96
    $promptLines = Format-CalloutContent $prompt
    $entry = @"

> [!user]+ #$promptNum - You ($time)
> **cwd**: $bt$bt$cwd$bt$bt
>
$promptLines

---

"@
    [System.IO.File]::AppendAllText($filePath, $entry, $utf8)

} catch {
    # Never block Claude
}
exit 0
