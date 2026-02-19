package settings

import (
	"os"
	"path/filepath"
	"strings"
)

const settingsFileName = "claude-hooks.local.md"

// Settings holds plugin configuration from .claude/claude-hooks.local.md.
// Pointer fields are nil when not set, allowing callers to distinguish
// "not configured" from "explicitly set to default".
type Settings struct {
	VaultPath       string
	SkipWhenFocused *bool
	GitAutoPush     *bool
}

// ReadAll reads settings with project-level overriding user-global.
// Project-level: $CLAUDE_PROJECT_DIR/.claude/claude-hooks.local.md
// User-global:   ~/.claude/claude-hooks.local.md
// Returns zero Settings on any error.
func ReadAll() Settings {
	var s Settings

	// User-global (lowest priority, applied first)
	if home, err := os.UserHomeDir(); err == nil {
		global := readFrom(filepath.Join(home, ".claude", settingsFileName))
		merge(&s, &global)
	}

	// Project-level (highest priority, overrides global)
	if projDir := os.Getenv("CLAUDE_PROJECT_DIR"); projDir != "" {
		project := readFrom(filepath.Join(projDir, ".claude", settingsFileName))
		merge(&s, &project)
	}

	return s
}

// merge overlays src onto dst. Non-nil/non-empty src fields override dst.
func merge(dst, src *Settings) {
	if src.VaultPath != "" {
		dst.VaultPath = src.VaultPath
	}
	if src.SkipWhenFocused != nil {
		dst.SkipWhenFocused = src.SkipWhenFocused
	}
	if src.GitAutoPush != nil {
		dst.GitAutoPush = src.GitAutoPush
	}
}

// readFrom parses a single .local.md file's YAML frontmatter.
func readFrom(path string) Settings {
	var s Settings
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	fm := extractFrontmatter(string(data))
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := parseYAMLLine(line)
		if !ok {
			continue
		}
		switch key {
		case "vault_path":
			s.VaultPath = val
		case "skip_when_focused":
			b := parseBool(val)
			s.SkipWhenFocused = &b
		case "git_auto_push":
			b := parseBool(val)
			s.GitAutoPush = &b
		}
	}
	return s
}

// extractFrontmatter returns the text between the first pair of "---" lines.
func extractFrontmatter(content string) string {
	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if start == -1 {
				start = i
			} else {
				return strings.Join(lines[start+1:i], "\n")
			}
		}
	}
	return ""
}

// parseYAMLLine splits "key: value" into (key, value, true).
// Returns ("", "", false) for comments, blank lines, or malformed lines.
func parseYAMLLine(line string) (string, string, bool) {
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	val := strings.TrimSpace(line[idx+1:])
	// Strip surrounding quotes
	val = strings.Trim(val, "\"'")
	return key, val, key != ""
}

// parseBool interprets a YAML boolean value.
func parseBool(val string) bool {
	switch strings.ToLower(val) {
	case "true", "yes", "on", "1":
		return true
	default:
		return false
	}
}
