package obsidian

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/valentinclaes/claude-hooks/internal/settings"
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

// VaultDir returns the Obsidian vault directory.
// Priority: 1) CLAUDE_VAULT env var, 2) settings file (.claude/claude-hooks.local.md).
func VaultDir() string {
	if v := os.Getenv("CLAUDE_VAULT"); v != "" {
		return v
	}
	if s := settings.ReadAll(); s.VaultPath != "" {
		return s.VaultPath
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

// BuildFrontmatter generates YAML frontmatter for a new session file.
func BuildFrontmatter(date, sessionID, project, startTime, resumedFrom string) string {
	projectTag := strings.ToLower(regexp.MustCompile(`\s+`).ReplaceAllString(project, "-"))

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("date: " + date + "\n")
	sb.WriteString("session_id: " + sessionID + "\n")
	sb.WriteString("project: " + project + "\n")
	sb.WriteString("start_time: " + startTime + "\n")
	if resumedFrom != "" {
		sb.WriteString("resumed_from: \"[[" + resumedFrom + "]]\"\n")
	}
	sb.WriteString("tags:\n")
	sb.WriteString("  - claude-session\n")
	sb.WriteString("  - " + projectTag + "\n")
	sb.WriteString("---\n")
	sb.WriteString("\n# Claude Session - " + project + "\n")

	if resumedFrom != "" {
		parentName := filepath.Base(resumedFrom)
		sb.WriteString("Resumed from [[" + resumedFrom + "|" + parentName + "]]\n")
	}

	sb.WriteString("\n---\n")
	return sb.String()
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
