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
    $hookData = $rawInput | ConvertFrom-Json
    $sessionId = $hookData.session_id
    $transcriptPath = $hookData.transcript_path

    if (-not $transcriptPath -or -not (Test-Path $transcriptPath)) { exit 0 }

    $utf8 = New-Object System.Text.UTF8Encoding($false)

    # Find the session file via temp mapping
    $sessionMap = Join-Path $env:TEMP "claude_session_$sessionId.txt"
    if (-not (Test-Path $sessionMap)) { exit 0 }

    $data = [System.IO.File]::ReadAllText($sessionMap, $utf8).Trim() -split "`n"
    $filePath = $data[0]
    if (-not (Test-Path $filePath)) { exit 0 }

    # Read transcript and collect text + plan content from recent messages
    $lines = [System.IO.File]::ReadAllLines($transcriptPath, $utf8)
    $responseText = ""
    $planText = ""

    # Walk backwards to find the last assistant text response
    for ($i = $lines.Count - 1; $i -ge 0; $i--) {
        $obj = $lines[$i] | ConvertFrom-Json -ErrorAction SilentlyContinue
        if (-not $obj) { continue }

        # Check for plan content on user messages (plans are stored as planContent)
        if (-not $planText -and $obj.planContent) {
            $planText = $obj.planContent
        }

        if ($obj.type -eq 'assistant' -and $obj.message.role -eq 'assistant') {
            $content = $obj.message.content
            if ($content -is [array]) {
                $texts = @()
                foreach ($item in $content) {
                    if ($item.type -eq 'text' -and $item.text) {
                        $texts += $item.text
                    }
                }
                if ($texts.Count -gt 0) {
                    $responseText = $texts -join "`n`n"
                    break
                }
            }
        }

        # Stop searching after 50 lines back
        if (($lines.Count - 1 - $i) -gt 50) { break }
    }

    $time = Get-Date -Format "HH:mm:ss"
    $output = ""

    # Log plan if found (collapsed by default)
    if ($planText) {
        if ($planText.Length -gt 5000) {
            $planText = $planText.Substring(0, 5000) + "`n`n... (truncated)"
        }
        $planLines = Format-CalloutContent $planText
        $output += @"

> [!plan]- Claude's Plan ($time)
$planLines

---

"@
    }

    # Log response (collapsed by default)
    if ($responseText) {
        if ($responseText.Length -gt 3000) {
            $responseText = $responseText.Substring(0, 3000) + "`n`n... (truncated, $($responseText.Length) chars total)"
        }
        $responseLines = Format-CalloutContent $responseText
        $output += @"

> [!claude]- Claude ($time)
$responseLines

---

"@
    }

    if ($output) {
        [System.IO.File]::AppendAllText($filePath, $output, $utf8)
    }

    # Update session duration in frontmatter
    try {
        $fileContent = [System.IO.File]::ReadAllText($filePath, $utf8)
        if ($fileContent -match '(?m)^start_time:\s*(\d{2}:\d{2})') {
            $startStr = $Matches[1]
            $today = Get-Date -Format "yyyy-MM-dd"
            $startDt = [datetime]::ParseExact("$today $startStr", "yyyy-MM-dd HH:mm", $null)
            $now = Get-Date
            $dur = $now - $startDt
            $totalMin = [math]::Floor($dur.TotalMinutes)
            if ($totalMin -lt 1) { $totalMin = 1 }
            $durStr = "${totalMin}min"

            if ($fileContent -match '(?m)^duration:.*$') {
                $fileContent = $fileContent -replace '(?m)^duration:.*$', "duration: $durStr"
            } else {
                $fileContent = $fileContent -replace '(?m)^(start_time:\s*.*)$', "`$1`nduration: $durStr"
            }
            [System.IO.File]::WriteAllText($filePath, $fileContent, $utf8)
        }
    } catch {
        # Don't block on duration errors
    }

    # Update daily index note
    try {
        $vaultDir = if ($env:CLAUDE_VAULT) { $env:CLAUDE_VAULT }
                   elseif ([Environment]::GetEnvironmentVariable("CLAUDE_VAULT", "User")) { [Environment]::GetEnvironmentVariable("CLAUDE_VAULT", "User") }
                   else { $null }
        if (-not $vaultDir) { return }  # CLAUDE_VAULT not configured, skip daily index
        $date = Get-Date -Format "yyyy-MM-dd"
        $dailyPath = Join-Path $vaultDir "$date.md"

        # Find all session files for today across all projects
        $projectDirs = Get-ChildItem -Path $vaultDir -Directory -ErrorAction SilentlyContinue
        $sessions = @()

        foreach ($pd in $projectDirs) {
            $todayFiles = Get-ChildItem -Path $pd.FullName -Filter "${date}_*.md" -ErrorAction SilentlyContinue
            foreach ($tf in $todayFiles) {
                $tfContent = [System.IO.File]::ReadAllText($tf.FullName, $utf8)

                # Extract time from filename (e.g., 2026-02-12_0915.md -> 09:15)
                $tfTime = ""
                if ($tf.Name -match "${date}_(\d{2})(\d{2})") {
                    $tfTime = "$($Matches[1]):$($Matches[2])"
                }

                # Extract duration from frontmatter
                $tfDuration = ""
                if ($tfContent -match '(?m)^duration:\s*(.+)$') {
                    $tfDuration = $Matches[1].Trim()
                }

                # Extract prompt count from temp mapping or by counting callouts
                $tfPrompts = 0
                $tfSessionId = ""
                if ($tfContent -match '(?m)^session_id:\s*(.+)$') {
                    $tfSessionId = $Matches[1].Trim()
                }
                if ($tfSessionId) {
                    $tfMap = Join-Path $env:TEMP "claude_session_$tfSessionId.txt"
                    if (Test-Path $tfMap) {
                        $tfData = [System.IO.File]::ReadAllText($tfMap, $utf8).Trim() -split "`n"
                        if ($tfData.Count -ge 2) {
                            $tfPrompts = [int]$tfData[1]
                        }
                    }
                }
                # Fallback: count user callouts
                if ($tfPrompts -eq 0) {
                    $tfPrompts = ([regex]::Matches($tfContent, '\[!user\]')).Count
                }

                # Build relative path for wikilink
                $relPath = $tf.FullName.Replace($vaultDir + "\", "").Replace("\", "/").Replace(".md", "")

                $sessions += [PSCustomObject]@{
                    Project  = $pd.Name
                    RelPath  = $relPath
                    Time     = $tfTime
                    Duration = $tfDuration
                    Prompts  = $tfPrompts
                }
            }
        }

        if ($sessions.Count -gt 0) {
            $grouped = $sessions | Sort-Object Time | Group-Object Project

            $indexContent = @"
---
date: $date
tags:
  - claude-daily
---

# Claude Sessions - $date

"@
            foreach ($group in ($grouped | Sort-Object Name)) {
                $indexContent += "`n## $($group.Name)`n"
                foreach ($s in $group.Group) {
                    $parts = @()
                    if ($s.Duration) { $parts += $s.Duration }
                    if ($s.Prompts -gt 0) { $parts += "$($s.Prompts) prompts" }
                    $meta = ""
                    if ($parts.Count -gt 0) { $meta = " ($($parts -join ', '))" }
                    $indexContent += "- [[$($s.RelPath)|$($s.Time)]]$meta`n"
                }
            }

            [System.IO.File]::WriteAllText($dailyPath, $indexContent, $utf8)
        }
    } catch {
        # Don't block on daily index errors
    }

} catch {
    # Never block Claude
}
exit 0
