package obsidian

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Pre-compiled regexes for stripping system-injected XML tags.
var systemTagPatterns []*regexp.Regexp

func init() {
	tags := []string{
		"system-reminder",
		"task-notification",
		"claude-mem-context",
		"context-window-budget",
		"skill-reminders",
		"local-command-caveat",
		"command-name",
		"command-message",
		"command-args",
		"local-command-stdout",
	}
	for _, tag := range tags {
		// (?s) = dotall so . matches newlines
		pat := regexp.MustCompile(`(?s)<` + tag + `>.*?</` + tag + `>`)
		systemTagPatterns = append(systemTagPatterns, pat)
	}
}

var sanitizeRe = regexp.MustCompile(`[\\/:*?"<>|]`)

// VaultDir returns the Obsidian vault directory from CLAUDE_VAULT env var.
func VaultDir() string {
	if v := os.Getenv("CLAUDE_VAULT"); v != "" {
		return v
	}
	return ""
}

// SanitizeProject strips illegal filesystem characters and leading dots
// from a project name. Leading dots are stripped because Obsidian hides
// dotfolders by default.
func SanitizeProject(name string) string {
	name = strings.TrimLeft(name, ".")
	if name == "" {
		name = "unnamed"
	}
	return sanitizeRe.ReplaceAllString(name, "_")
}

// StripSystemTags removes all system-injected XML tags from prompt text.
func StripSystemTags(prompt string) string {
	for _, pat := range systemTagPatterns {
		prompt = pat.ReplaceAllString(prompt, "")
	}
	return strings.TrimSpace(prompt)
}

// Truncate truncates text and appends a note with total char count.
func Truncate(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + fmt.Sprintf("\n\n... (truncated, %d chars total)", len(text))
}

// TruncateSimple truncates text with a simple "... (truncated)" suffix.
func TruncateSimple(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "\n\n... (truncated)"
}

// FormatCalloutContent prefixes each line with "> ".
func FormatCalloutContent(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}

// FrontmatterData holds all data for session frontmatter generation.
type FrontmatterData struct {
	Date, SessionID, Project, StartTime, ResumedFrom string
	// Phase 1: stats
	Model         string // "opus", "sonnet", "haiku"
	Tools         map[string]int
	TokensIn      int
	TokensOut     int
	CacheRead     int
	CacheCreation int
	EstCost       string // "$0.23"
	FilesTouched  []string
	// Phase 2: git
	Branch  string
	Commits []string
}

// BuildFrontmatter generates YAML frontmatter for a new session file.
func BuildFrontmatter(data FrontmatterData) string {
	projectTag := strings.ToLower(regexp.MustCompile(`\s+`).ReplaceAllString(data.Project, "-"))

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("date: " + data.Date + "\n")
	sb.WriteString("session_id: " + data.SessionID + "\n")
	sb.WriteString("project: " + data.Project + "\n")
	sb.WriteString("start_time: " + data.StartTime + "\n")
	if data.ResumedFrom != "" {
		sb.WriteString("resumed_from: \"[[" + data.ResumedFrom + "]]\"\n")
	}
	writeStatsFrontmatter(&sb, data)
	sb.WriteString("tags:\n")
	sb.WriteString("  - claude-session\n")
	sb.WriteString("  - " + projectTag + "\n")
	sb.WriteString("---\n")
	sb.WriteString("\n# Claude Session - " + data.Project + "\n")

	if data.ResumedFrom != "" {
		parentName := filepath.Base(data.ResumedFrom)
		sb.WriteString("Resumed from [[" + data.ResumedFrom + "|" + parentName + "]]\n")
	}

	sb.WriteString("\n---\n")
	return sb.String()
}

// writeStatsFrontmatter writes branch/model/tool/token/cache/cost/files fields (omitted when zero/empty).
func writeStatsFrontmatter(sb *strings.Builder, data FrontmatterData) {
	if data.Branch != "" {
		sb.WriteString("branch: " + data.Branch + "\n")
	}
	if data.Model != "" {
		sb.WriteString("model: " + data.Model + "\n")
	}
	if len(data.Tools) > 0 {
		sb.WriteString("tools:\n")
		// Sort tool names for deterministic output
		names := sortedKeys(data.Tools)
		for _, name := range names {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", name, data.Tools[name]))
		}
	}
	if data.TokensIn > 0 {
		sb.WriteString(fmt.Sprintf("tokens_in: %d\n", data.TokensIn))
	}
	if data.TokensOut > 0 {
		sb.WriteString(fmt.Sprintf("tokens_out: %d\n", data.TokensOut))
	}
	if data.CacheRead > 0 {
		sb.WriteString(fmt.Sprintf("cache_read: %d\n", data.CacheRead))
	}
	if data.CacheCreation > 0 {
		sb.WriteString(fmt.Sprintf("cache_creation: %d\n", data.CacheCreation))
	}
	if data.EstCost != "" {
		sb.WriteString("estimated_cost: \"" + data.EstCost + "\"\n")
	}
	if len(data.FilesTouched) > 0 {
		sb.WriteString("files_touched:\n")
		for _, f := range data.FilesTouched {
			sb.WriteString("  - " + f + "\n")
		}
	}
	if len(data.Commits) > 0 {
		sb.WriteString("commits:\n")
		for _, c := range data.Commits {
			sb.WriteString("  - " + c + "\n")
		}
	}
}

// sortedKeys returns sorted keys from a map.
func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FormatStatsLine generates a compact stats summary for the session heading.
func FormatStatsLine(toolCounts map[string]int, tokensIn, tokensOut int, estCost string) string {
	totalTools := 0
	for _, c := range toolCounts {
		totalTools += c
	}
	if totalTools == 0 && tokensIn == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("> **")
	sb.WriteString(fmt.Sprintf("%d tool calls", totalTools))
	sb.WriteString("**")
	if tokensIn > 0 || tokensOut > 0 {
		sb.WriteString(" | **")
		sb.WriteString(formatTokenCount(tokensIn))
		sb.WriteString(" in / ")
		sb.WriteString(formatTokenCount(tokensOut))
		sb.WriteString(" out tokens**")
	}
	if estCost != "" {
		sb.WriteString(" | ~" + estCost)
	}
	sb.WriteString("\n")

	// Tool breakdown line
	if totalTools > 0 {
		sb.WriteString("> ")
		type toolEntry struct {
			Name  string
			Count int
		}
		var entries []toolEntry
		for name, count := range toolCounts {
			entries = append(entries, toolEntry{name, count})
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Count != entries[j].Count {
				return entries[i].Count > entries[j].Count
			}
			return entries[i].Name < entries[j].Name
		})
		for i, e := range entries {
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(fmt.Sprintf("%s(%d)", e.Name, e.Count))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatTokenCount formats token counts with K suffix for readability.
func formatTokenCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%dK", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

// FormatCommitsEntry formats git commits as an Obsidian callout.
func FormatCommitsEntry(timeStr string, commits []string) string {
	if len(commits) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n> [!git]- Commits (%s)\n", timeStr))
	for _, c := range commits {
		sb.WriteString("> - `" + c + "`\n")
	}
	sb.WriteString("\n---\n")
	return sb.String()
}

// UpdateFrontmatterStats updates an existing session file's frontmatter with stats.
// It inserts stats fields before the tags: line in the frontmatter.
func UpdateFrontmatterStats(content string, tools map[string]int, tokensIn, tokensOut, cacheRead, cacheCreation int, estCost string, filesTouched, commits []string, branch, model string) string {
	// Find the frontmatter boundaries
	if !strings.HasPrefix(content, "---\n") {
		return content
	}
	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx < 0 {
		return content
	}
	endIdx += 4 // adjust for the offset

	frontmatter := content[4:endIdx]
	rest := content[endIdx:]

	// Build new stats lines
	var stats strings.Builder
	data := FrontmatterData{
		Model:         model,
		Tools:         tools,
		TokensIn:      tokensIn,
		TokensOut:     tokensOut,
		CacheRead:     cacheRead,
		CacheCreation: cacheCreation,
		EstCost:       estCost,
		FilesTouched:  filesTouched,
		Commits:       commits,
		Branch:        branch,
	}

	// Remove existing stats lines if present
	var cleanedLines []string
	skipFields := map[string]bool{
		"tools:": true, "tokens_in:": true, "tokens_out:": true,
		"cache_read:": true, "cache_creation:": true,
		"estimated_cost:": true, "files_touched:": true, "commits:": true,
		"branch:": true, "model:": true,
	}
	lines := strings.Split(frontmatter, "\n")
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Check if this line starts a field we want to replace
		isSkipField := false
		for prefix := range skipFields {
			if strings.HasPrefix(trimmed, prefix) {
				isSkipField = true
				break
			}
		}
		if isSkipField {
			inBlock = true
			continue
		}
		// Skip indented continuation lines (YAML block items under a skipped field)
		if inBlock && strings.HasPrefix(line, "  ") {
			continue
		}
		inBlock = false
		cleanedLines = append(cleanedLines, line)
	}

	// Find tags: line to insert before
	insertIdx := len(cleanedLines)
	for i, line := range cleanedLines {
		if strings.HasPrefix(strings.TrimSpace(line), "tags:") {
			insertIdx = i
			break
		}
	}

	// Build stats fragment
	writeStatsFrontmatter(&stats, data)
	statsStr := stats.String()
	if statsStr != "" {
		statsStr = strings.TrimRight(statsStr, "\n")
		statsLines := strings.Split(statsStr, "\n")
		// Insert before tags
		newLines := make([]string, 0, len(cleanedLines)+len(statsLines))
		newLines = append(newLines, cleanedLines[:insertIdx]...)
		newLines = append(newLines, statsLines...)
		newLines = append(newLines, cleanedLines[insertIdx:]...)
		cleanedLines = newLines
	}

	return "---\n" + strings.Join(cleanedLines, "\n") + rest
}

// FormatPromptEntry formats a user prompt as an Obsidian callout (expanded).
func FormatPromptEntry(promptNum int, timeStr, cwd, promptText string) string {
	calloutContent := FormatCalloutContent(promptText)
	return fmt.Sprintf("\n> [!user]+ #%d - You (%s)\n> **cwd**: ``%s``\n>\n%s\n\n---\n", promptNum, timeStr, cwd, calloutContent)
}

// FormatPlanEntry formats a plan as a collapsed Obsidian callout.
func FormatPlanEntry(timeStr, planText string) string {
	calloutContent := FormatCalloutContent(planText)
	return fmt.Sprintf("\n> [!plan]- Claude's Plan (%s)\n%s\n\n---\n", timeStr, calloutContent)
}

// FormatResponseEntry formats a response as a collapsed Obsidian callout.
func FormatResponseEntry(timeStr, responseText string) string {
	calloutContent := FormatCalloutContent(responseText)
	return fmt.Sprintf("\n> [!claude]- Claude (%s)\n%s\n\n---\n", timeStr, calloutContent)
}

// FindParentSession searches for a parent session's vault path by reading
// the transcript file for parentUuid, then finding the matching Obsidian note.
func FindParentSession(sessionID, claudeProjectsDir, vaultDir string) string {
	// Find transcript file
	var transcriptFile string
	filepath.Walk(claudeProjectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == sessionID+".jsonl" {
			transcriptFile = path
			return filepath.SkipAll
		}
		return nil
	})
	if transcriptFile == "" {
		return ""
	}

	// Read first ~20 lines to find parentUuid
	data, err := os.ReadFile(transcriptFile)
	if err != nil {
		return ""
	}
	lines := strings.SplitN(string(data), "\n", 21)
	if len(lines) > 20 {
		lines = lines[:20]
	}

	parentID := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj struct {
			ParentUuid string `json:"parentUuid"`
		}
		if err := json.Unmarshal([]byte(line), &obj); err == nil && obj.ParentUuid != "" {
			parentID = obj.ParentUuid
			break
		}
	}
	if parentID == "" {
		return ""
	}

	// Search Obsidian vault for a file with session_id: parentID
	sessionRe := regexp.MustCompile(`session_id:\s*` + regexp.QuoteMeta(parentID))
	var result string
	filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if sessionRe.Match(content) {
			rel, err := filepath.Rel(vaultDir, path)
			if err != nil {
				return nil
			}
			result = strings.ReplaceAll(rel, "\\", "/")
			result = strings.TrimSuffix(result, ".md")
			return filepath.SkipAll
		}
		return nil
	})
	return result
}
