package obsidian

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	durationRe    = regexp.MustCompile(`(?m)^duration:\s*(.+)$`)
	sessionIDRe   = regexp.MustCompile(`(?m)^session_id:\s*(.+)$`)
	userCallout   = regexp.MustCompile(`\[!user\]`)
	estCostRe     = regexp.MustCompile(`(?m)^estimated_cost:\s*"?([^"\n]+)"?$`)
	toolsBlockRe  = regexp.MustCompile(`(?m)^tools:\n((?:\s+\w+:\s*\d+\n)*)`)
	singleToolRe  = regexp.MustCompile(`\s+\w+:\s*(\d+)`)
)

type sessionEntry struct {
	Project  string
	RelPath  string
	Time     string
	Duration string
	Prompts  int
	Tools    int
	EstCost  string
}

// RebuildDailyIndex scans project subdirs for today's sessions and rebuilds the daily index.
func RebuildDailyIndex(vaultDir, date string) error {
	entries, err := os.ReadDir(vaultDir)
	if err != nil {
		return err
	}

	var sessions []sessionEntry

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectDir := filepath.Join(vaultDir, entry.Name())
		pattern := filepath.Join(projectDir, date+"_*.md")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			contentStr := string(content)
			fileName := filepath.Base(match)

			// Extract time from filename (e.g., 2026-02-12_0915.md -> 09:15)
			timeStr := ""
			prefix := date + "_"
			after := strings.TrimPrefix(fileName, prefix)
			if len(after) >= 4 {
				digits := after[:4]
				if isDigits(digits) {
					timeStr = digits[:2] + ":" + digits[2:4]
				}
			}

			// Extract duration from frontmatter
			duration := ""
			if m := durationRe.FindStringSubmatch(contentStr); len(m) > 1 {
				duration = strings.TrimSpace(m[1])
			}

			// Extract prompt count: try session temp file, fallback to counting callouts
			prompts := 0
			if m := sessionIDRe.FindStringSubmatch(contentStr); len(m) > 1 {
				sid := strings.TrimSpace(m[1])
				mapFile := filepath.Join(os.TempDir(), "claude_session_"+sid+".txt")
				if data, err := os.ReadFile(mapFile); err == nil {
					lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
					if len(lines) >= 2 {
						fmt.Sscanf(strings.TrimSpace(lines[1]), "%d", &prompts)
					}
				}
			}
			if prompts == 0 {
				prompts = len(userCallout.FindAllString(contentStr, -1))
			}

			// Extract tool count from frontmatter
			toolCount := 0
			if m := toolsBlockRe.FindStringSubmatch(contentStr); len(m) > 1 {
				for _, sm := range singleToolRe.FindAllStringSubmatch(m[1], -1) {
					if len(sm) > 1 {
						n, _ := strconv.Atoi(sm[1])
						toolCount += n
					}
				}
			}

			// Extract estimated cost from frontmatter
			estCost := ""
			if m := estCostRe.FindStringSubmatch(contentStr); len(m) > 1 {
				estCost = strings.TrimSpace(m[1])
			}

			rel, err := filepath.Rel(vaultDir, match)
			if err != nil {
				continue
			}
			relPath := strings.ReplaceAll(rel, "\\", "/")
			relPath = strings.TrimSuffix(relPath, ".md")

			sessions = append(sessions, sessionEntry{
				Project:  entry.Name(),
				RelPath:  relPath,
				Time:     timeStr,
				Duration: duration,
				Prompts:  prompts,
				Tools:    toolCount,
				EstCost:  estCost,
			})
		}
	}

	if len(sessions) == 0 {
		return nil
	}

	// Sort by time, then group by project
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Time < sessions[j].Time
	})

	grouped := make(map[string][]sessionEntry)
	var projectOrder []string
	for _, s := range sessions {
		if _, ok := grouped[s.Project]; !ok {
			projectOrder = append(projectOrder, s.Project)
		}
		grouped[s.Project] = append(grouped[s.Project], s)
	}
	sort.Slice(projectOrder, func(i, j int) bool {
		return strings.ToLower(projectOrder[i]) < strings.ToLower(projectOrder[j])
	})

	var sb strings.Builder
	sb.WriteString("---\ndate: " + date + "\ntags:\n  - claude-daily\n---\n\n# Claude Sessions - " + date + "\n")

	for _, proj := range projectOrder {
		sb.WriteString("\n## " + proj + "\n")
		for _, s := range grouped[proj] {
			var parts []string
			if s.Duration != "" {
				parts = append(parts, s.Duration)
			}
			if s.Prompts > 0 {
				parts = append(parts, fmt.Sprintf("%d prompts", s.Prompts))
			}
			if s.Tools > 0 {
				parts = append(parts, fmt.Sprintf("%d tools", s.Tools))
			}
			if s.EstCost != "" {
				parts = append(parts, "~"+s.EstCost)
			}
			meta := ""
			if len(parts) > 0 {
				meta = " (" + strings.Join(parts, ", ") + ")"
			}
			sb.WriteString("- [[" + s.RelPath + "|" + s.Time + "]]" + meta + "\n")
		}
	}

	dailyPath := filepath.Join(vaultDir, date+".md")
	return os.WriteFile(dailyPath, []byte(sb.String()), 0644)
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
