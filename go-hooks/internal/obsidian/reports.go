package obsidian

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Regexes for frontmatter fields not covered by dailyindex.go.
var (
	tokensInRe        = regexp.MustCompile(`(?m)^tokens_in:\s*(\d+)`)
	tokensOutRe       = regexp.MustCompile(`(?m)^tokens_out:\s*(\d+)`)
	cacheReadRe       = regexp.MustCompile(`(?m)^cache_read:\s*(\d+)`)
	cacheCreationRe   = regexp.MustCompile(`(?m)^cache_creation:\s*(\d+)`)
	modelFmRe         = regexp.MustCompile(`(?m)^model:\s*(.+)$`)
	branchFmRe        = regexp.MustCompile(`(?m)^branch:\s*(.+)$`)
	startTimeFmRe     = regexp.MustCompile(`(?m)^start_time:\s*(\d{2}:\d{2})`)
	filesTouchedRe    = regexp.MustCompile(`(?m)^files_touched:\n((?:\s+-.+\n)*)`)
	commitsBlockRe    = regexp.MustCompile(`(?m)^commits:\n((?:\s+-.+\n)*)`)
	toolNameCountRe   = regexp.MustCompile(`\s+(\w+):\s*(\d+)`)
	durationHoursRe   = regexp.MustCompile(`(\d+)h`)
	durationMinutesRe = regexp.MustCompile(`(\d+)min`)
)

// SessionMeta holds parsed frontmatter data for a single session file.
type SessionMeta struct {
	Project       string
	Date          string // "2026-02-17"
	StartTime     string // "14:30"
	Duration      string // "25min"
	DurationMin   int
	Model         string // "opus", "sonnet", "haiku"
	Tools         map[string]int
	TokensIn      int
	TokensOut     int
	CacheRead     int
	CacheCreation int
	EstCost       string  // "$0.23"
	CostFloat     float64 // 0.23
	FilesTouched  []string
	Commits       int
	Branch        string
	Prompts       int
}

// ScanSessions walks project subdirs in vaultDir, finds session files within
// [startDate, endDate], and parses their frontmatter.
func ScanSessions(vaultDir string, startDate, endDate time.Time) []SessionMeta {
	entries, err := os.ReadDir(vaultDir)
	if err != nil {
		return nil
	}

	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")
	var sessions []SessionMeta

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		project := entry.Name()
		projectDir := filepath.Join(vaultDir, project)
		matches, err := filepath.Glob(filepath.Join(projectDir, "????-??-??_*.md"))
		if err != nil {
			continue
		}
		for _, match := range matches {
			fileName := filepath.Base(match)
			if len(fileName) < 10 {
				continue
			}
			fileDate := fileName[:10]
			if fileDate < startStr || fileDate > endStr {
				continue
			}
			content, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			meta := parseSessionMeta(string(content), project, fileDate)
			sessions = append(sessions, meta)
		}
	}

	return sessions
}

func parseSessionMeta(content, project, date string) SessionMeta {
	content = strings.ReplaceAll(content, "\r", "")
	meta := SessionMeta{
		Project: project,
		Date:    date,
		Tools:   make(map[string]int),
	}

	if m := startTimeFmRe.FindStringSubmatch(content); len(m) > 1 {
		meta.StartTime = strings.TrimSpace(m[1])
	}
	if m := durationRe.FindStringSubmatch(content); len(m) > 1 {
		meta.Duration = strings.TrimSpace(m[1])
		meta.DurationMin = parseDurationMin(meta.Duration)
	}
	if m := toolsBlockRe.FindStringSubmatch(content); len(m) > 1 {
		for _, sm := range toolNameCountRe.FindAllStringSubmatch(m[1], -1) {
			if len(sm) > 2 {
				n, _ := strconv.Atoi(sm[2])
				meta.Tools[sm[1]] = n
			}
		}
	}
	if m := modelFmRe.FindStringSubmatch(content); len(m) > 1 {
		meta.Model = strings.TrimSpace(m[1])
	}
	if m := tokensInRe.FindStringSubmatch(content); len(m) > 1 {
		meta.TokensIn, _ = strconv.Atoi(m[1])
	}
	if m := tokensOutRe.FindStringSubmatch(content); len(m) > 1 {
		meta.TokensOut, _ = strconv.Atoi(m[1])
	}
	if m := cacheReadRe.FindStringSubmatch(content); len(m) > 1 {
		meta.CacheRead, _ = strconv.Atoi(m[1])
	}
	if m := cacheCreationRe.FindStringSubmatch(content); len(m) > 1 {
		meta.CacheCreation, _ = strconv.Atoi(m[1])
	}
	if m := estCostRe.FindStringSubmatch(content); len(m) > 1 {
		meta.EstCost = strings.TrimSpace(m[1])
		meta.CostFloat = parseCost(meta.EstCost)
	}
	if m := branchFmRe.FindStringSubmatch(content); len(m) > 1 {
		meta.Branch = strings.TrimSpace(m[1])
	}
	if m := filesTouchedRe.FindStringSubmatch(content); len(m) > 1 {
		for _, line := range strings.Split(m[1], "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- ") {
				meta.FilesTouched = append(meta.FilesTouched, strings.TrimPrefix(line, "- "))
			}
		}
	}
	if m := commitsBlockRe.FindStringSubmatch(content); len(m) > 1 {
		for _, line := range strings.Split(m[1], "\n") {
			if strings.TrimSpace(line) != "" {
				meta.Commits++
			}
		}
	}
	meta.Prompts = len(userCallout.FindAllString(content, -1))
	return meta
}

// parseDurationMin parses "25min", "1h 30min", "2h" into total minutes.
func parseDurationMin(s string) int {
	total := 0
	if m := durationHoursRe.FindStringSubmatch(s); len(m) > 1 {
		h, _ := strconv.Atoi(m[1])
		total += h * 60
	}
	if m := durationMinutesRe.FindStringSubmatch(s); len(m) > 1 {
		mins, _ := strconv.Atoi(m[1])
		total += mins
	}
	return total
}

func parseCost(s string) float64 {
	s = strings.TrimPrefix(s, "$")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// weekStart returns Monday 00:00 of the week containing t.
func weekStart(t time.Time) time.Time {
	wd := t.Weekday()
	if wd == time.Sunday {
		wd = 7
	}
	offset := int(wd) - int(time.Monday)
	y, m, d := t.Date()
	return time.Date(y, m, d-offset, 0, 0, 0, 0, t.Location())
}

// formatDuration converts total minutes to human-readable format.
func formatDuration(totalMin int) string {
	if totalMin <= 0 {
		return "0m"
	}
	h := totalMin / 60
	m := totalMin % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("~%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("~%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

// isStaleToday returns true if path doesn't exist or was last modified before today.
func isStaleToday(path string, now time.Time) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	return info.ModTime().Format("2006-01-02") != now.Format("2006-01-02")
}

// RebuildWeeklyStatsIfStale rebuilds the weekly stats report if stale.
// The filename includes the date range (e.g. Weekly-2026-02-16-to-2026-02-19.md).
// When the end date advances, the old file is removed and replaced.
func RebuildWeeklyStatsIfStale(vaultDir string, now time.Time) error {
	start := weekStart(now)
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")
	fileName := fmt.Sprintf("Weekly-%s-to-%s.md", startStr, endStr)
	filePath := filepath.Join(vaultDir, fileName)

	if !isStaleToday(filePath, now) {
		return nil
	}

	// Remove old weekly files for the same week start but different end date
	prefix := fmt.Sprintf("Weekly-%s-to-", startStr)
	entries, _ := os.ReadDir(vaultDir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) && e.Name() != fileName {
			os.Remove(filepath.Join(vaultDir, e.Name()))
		}
	}

	sessions := ScanSessions(vaultDir, start, end)
	if len(sessions) == 0 {
		return nil
	}

	report := buildWeeklyReport(sessions, start, end)
	return os.WriteFile(filePath, []byte(report), 0644)
}

// RebuildMonthlyStatsIfStale rebuilds the monthly stats report if stale.
func RebuildMonthlyStatsIfStale(vaultDir string, now time.Time) error {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	fileName := fmt.Sprintf("Monthly-%s.md", start.Format("2006-01"))
	filePath := filepath.Join(vaultDir, fileName)

	if !isStaleToday(filePath, now) {
		return nil
	}

	sessions := ScanSessions(vaultDir, start, end)
	if len(sessions) == 0 {
		return nil
	}

	report := buildMonthlyReport(sessions, start)
	return os.WriteFile(filePath, []byte(report), 0644)
}

// --- Report builders ---

func buildWeeklyReport(sessions []SessionMeta, startDate, endDate time.Time) string {
	var sb strings.Builder
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	totalSessions := len(sessions)
	activeDays := uniqueDates(sessions)
	totalDays := int(endDate.Sub(startDate).Hours()/24) + 1
	totalTimeMin := sumDuration(sessions)
	totalCost := sumCost(sessions)
	totalTokensIn := sumTokensIn(sessions)
	totalTokensOut := sumTokensOut(sessions)
	totalTools := sumTools(sessions)
	totalCommits := sumCommits(sessions)
	projects := aggregateProjects(sessions)

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("date_range: \"%s to %s\"\n", startStr, endStr))
	sb.WriteString("type: weekly-stats\n")
	sb.WriteString("auto_generated: true\n")
	sb.WriteString("tags:\n  - claude-weekly\n")
	sb.WriteString("---\n\n")

	sb.WriteString(fmt.Sprintf("# Weekly Stats: %s to %s\n\n", startStr, endStr))
	sb.WriteString("> *Auto-generated. Run `/weekly` for narrative version.*\n\n")

	// Overview
	sb.WriteString("## Overview\n\n")
	sb.WriteString("| Metric | Value |\n|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Sessions | %d |\n", totalSessions))
	sb.WriteString(fmt.Sprintf("| Active Days | %d/%d |\n", activeDays, totalDays))
	sb.WriteString(fmt.Sprintf("| Total Time | %s |\n", formatDuration(totalTimeMin)))
	sb.WriteString(fmt.Sprintf("| Total Cost | ~$%.2f |\n", totalCost))
	sb.WriteString(fmt.Sprintf("| Tokens In | %s |\n", formatTokenCount(totalTokensIn)))
	sb.WriteString(fmt.Sprintf("| Tokens Out | %s |\n", formatTokenCount(totalTokensOut)))
	sb.WriteString(fmt.Sprintf("| Tool Calls | %d |\n", totalTools))
	sb.WriteString(fmt.Sprintf("| Commits | %d |\n", totalCommits))
	sb.WriteString("\n")

	// Projects
	sb.WriteString("## Projects\n\n")
	sb.WriteString("| Project | Sessions | Time | Cost | Tools | Commits |\n")
	sb.WriteString("|---------|----------|------|------|-------|---------|\n")
	for _, p := range projects {
		sb.WriteString(fmt.Sprintf("| %s | %d | %s | ~$%.2f | %d | %d |\n",
			p.Name, p.Sessions, formatDuration(p.TimeMin), p.Cost, p.Tools, p.Commits))
	}
	sb.WriteString("\n")

	// Tool Usage
	tools := aggregateToolUsage(sessions)
	if len(tools) > 0 {
		sb.WriteString("## Tool Usage\n\n")
		sb.WriteString("| Tool | Count | % |\n|------|-------|---|\n")
		for _, t := range tools {
			pct := 0.0
			if totalTools > 0 {
				pct = float64(t.Count) / float64(totalTools) * 100
			}
			sb.WriteString(fmt.Sprintf("| %s | %d | %.0f%% |\n", t.Name, t.Count, pct))
		}
		sb.WriteString("\n")
	}

	// Most Touched Files
	files := aggregateFiles(sessions)
	if len(files) > 0 {
		sb.WriteString("## Most Touched Files\n\n")
		sb.WriteString("| File | Times Accessed |\n|------|---------------|\n")
		limit := 20
		if len(files) < limit {
			limit = len(files)
		}
		for _, f := range files[:limit] {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", f.Path, f.Count))
		}
		sb.WriteString("\n")
	}

	// Streaks & Trends
	sb.WriteString("## Streaks & Trends\n\n")
	sb.WriteString(fmt.Sprintf("- **Active days**: %d/%d days\n", activeDays, totalDays))
	busyDay, busyDaySessions, busyDayCost := busiestDay(sessions)
	if busyDay != "" {
		sb.WriteString(fmt.Sprintf("- **Busiest day**: %s (%d sessions, ~$%.2f)\n", busyDay, busyDaySessions, busyDayCost))
	}
	busyHour, busyHourSessions := busiestHour(sessions)
	if busyHourSessions > 0 {
		sb.WriteString(fmt.Sprintf("- **Busiest hour**: %d:00 (%d sessions)\n", busyHour, busyHourSessions))
	}
	if totalSessions > 0 {
		sb.WriteString(fmt.Sprintf("- **Avg session length**: %dmin\n", totalTimeMin/totalSessions))
	}
	sb.WriteString("\n")

	// Cost Analysis
	if totalCost > 0 {
		sb.WriteString("## Cost Analysis\n\n")
		sb.WriteString("| Project | Cost | % of Total |\n|---------|------|------------|\n")
		for _, p := range projects {
			pct := p.Cost / totalCost * 100
			sb.WriteString(fmt.Sprintf("| %s | ~$%.2f | %.0f%% |\n", p.Name, p.Cost, pct))
		}
		sb.WriteString("\n")
	}

	// Daily Breakdown
	days := aggregateDaily(sessions)
	sb.WriteString("## Daily Breakdown\n\n")
	sb.WriteString("| Date | Sessions | Time | Cost | Top Project |\n")
	sb.WriteString("|------|----------|------|------|-------------|\n")
	for _, d := range days {
		sb.WriteString(fmt.Sprintf("| %s | %d | %s | ~$%.2f | %s |\n",
			d.Date, d.Sessions, formatDuration(d.TimeMin), d.Cost, d.TopProject))
	}
	sb.WriteString("\n")

	return sb.String()
}

func buildMonthlyReport(sessions []SessionMeta, monthStart time.Time) string {
	var sb strings.Builder
	monthStr := monthStart.Format("2006-01")
	monthLabel := monthStart.Format("January 2006")

	totalSessions := len(sessions)
	activeDays := uniqueDates(sessions)
	totalTimeMin := sumDuration(sessions)
	totalCost := sumCost(sessions)
	totalTokensIn := sumTokensIn(sessions)
	totalTokensOut := sumTokensOut(sessions)
	totalTools := sumTools(sessions)
	totalCommits := sumCommits(sessions)
	projects := aggregateProjects(sessions)
	weeks := buildWeeklyBreakdown(sessions, monthStart)

	// Compute total days from 1st through last session date
	endDate := monthStart
	for _, s := range sessions {
		t, err := time.Parse("2006-01-02", s.Date)
		if err == nil && t.After(endDate) {
			endDate = t
		}
	}
	totalDays := int(endDate.Sub(monthStart).Hours()/24) + 1

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("month: \"%s\"\n", monthStr))
	sb.WriteString("type: monthly-stats\n")
	sb.WriteString("auto_generated: true\n")
	sb.WriteString("tags:\n  - claude-monthly\n")
	sb.WriteString("---\n\n")

	sb.WriteString(fmt.Sprintf("# Monthly Stats: %s\n\n", monthLabel))
	sb.WriteString("> *Auto-generated. Run `/monthly` for narrative version.*\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Sessions | %d |\n", totalSessions))
	sb.WriteString(fmt.Sprintf("| Active Days | %d/%d |\n", activeDays, totalDays))
	sb.WriteString(fmt.Sprintf("| Total Time | %s |\n", formatDuration(totalTimeMin)))
	sb.WriteString(fmt.Sprintf("| Total Cost | ~$%.2f |\n", totalCost))
	sb.WriteString(fmt.Sprintf("| Tokens | %s in / %s out |\n", formatTokenCount(totalTokensIn), formatTokenCount(totalTokensOut)))
	sb.WriteString(fmt.Sprintf("| Tool Calls | %d |\n", totalTools))
	sb.WriteString(fmt.Sprintf("| Commits | %d |\n", totalCommits))
	sb.WriteString(fmt.Sprintf("| Projects | %d |\n", len(projects)))
	sb.WriteString("\n")

	// Projects
	sb.WriteString("## Projects\n\n")
	sb.WriteString("| Project | Sessions | Time | Cost | Tools | Commits |\n")
	sb.WriteString("|---------|----------|------|------|-------|---------|\n")
	for _, p := range projects {
		sb.WriteString(fmt.Sprintf("| %s | %d | %s | ~$%.2f | %d | %d |\n",
			p.Name, p.Sessions, formatDuration(p.TimeMin), p.Cost, p.Tools, p.Commits))
	}
	sb.WriteString("\n")

	// Weekly Breakdown
	if len(weeks) > 0 {
		sb.WriteString("## Weekly Breakdown\n\n")
		sb.WriteString("| Week | Sessions | Time | Cost | Focus |\n")
		sb.WriteString("|------|----------|------|------|-------|\n")
		for _, w := range weeks {
			focus := w.Focus
			if focus == "" {
				focus = "-"
			}
			sb.WriteString(fmt.Sprintf("| %s | %d | %s | ~$%.2f | %s |\n",
				w.Label, w.Sessions, formatDuration(w.TimeMin), w.Cost, focus))
		}
		sb.WriteString("\n")
	}

	// Tool Usage
	tools := aggregateToolUsage(sessions)
	if len(tools) > 0 {
		sb.WriteString("## Tool Usage\n\n")
		sb.WriteString("| Tool | Count | % |\n|------|-------|---|\n")
		for _, t := range tools {
			pct := 0.0
			if totalTools > 0 {
				pct = float64(t.Count) / float64(totalTools) * 100
			}
			sb.WriteString(fmt.Sprintf("| %s | %d | %.0f%% |\n", t.Name, t.Count, pct))
		}
		sb.WriteString("\n")
	}

	// Tool Usage Evolution
	if len(tools) > 0 && len(weeks) > 1 {
		sb.WriteString("## Tool Usage Evolution\n\n")
		sb.WriteString("| Tool |")
		for _, w := range weeks {
			sb.WriteString(fmt.Sprintf(" %s |", w.Label))
		}
		sb.WriteString(" Total |\n")
		sb.WriteString("|------|")
		for range weeks {
			sb.WriteString("--------|")
		}
		sb.WriteString("-------|\n")
		for _, t := range tools {
			sb.WriteString(fmt.Sprintf("| %s |", t.Name))
			for _, w := range weeks {
				sb.WriteString(fmt.Sprintf(" %d |", w.ToolCounts[t.Name]))
			}
			sb.WriteString(fmt.Sprintf(" %d |\n", t.Count))
		}
		sb.WriteString("\n")
	}

	// Most Touched Files
	files := aggregateFiles(sessions)
	if len(files) > 0 {
		sb.WriteString("## Most Touched Files\n\n")
		sb.WriteString("| File | Times Accessed |\n|------|---------------|\n")
		limit := 20
		if len(files) < limit {
			limit = len(files)
		}
		for _, f := range files[:limit] {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", f.Path, f.Count))
		}
		sb.WriteString("\n")
	}

	// Streaks & Trends
	sb.WriteString("## Streaks & Trends\n\n")
	sb.WriteString(fmt.Sprintf("- **Active days**: %d/%d days\n", activeDays, totalDays))
	busyDay, busyDaySessions, busyDayCost := busiestDay(sessions)
	if busyDay != "" {
		sb.WriteString(fmt.Sprintf("- **Busiest day**: %s (%d sessions, ~$%.2f)\n", busyDay, busyDaySessions, busyDayCost))
	}
	busyHour, busyHourSessions := busiestHour(sessions)
	if busyHourSessions > 0 {
		sb.WriteString(fmt.Sprintf("- **Busiest hour**: %d:00 (%d sessions)\n", busyHour, busyHourSessions))
	}
	if totalSessions > 0 {
		sb.WriteString(fmt.Sprintf("- **Avg session length**: %dmin\n", totalTimeMin/totalSessions))
	}
	sb.WriteString("\n")

	// Cost Analysis
	if totalCost > 0 {
		sb.WriteString("## Cost Analysis\n\n")
		sb.WriteString("| Project | Cost | % of Total |\n|---------|------|------------|\n")
		for _, p := range projects {
			pct := p.Cost / totalCost * 100
			sb.WriteString(fmt.Sprintf("| %s | ~$%.2f | %.0f%% |\n", p.Name, p.Cost, pct))
		}
		if activeDays > 0 {
			sb.WriteString(fmt.Sprintf("\n**Daily average**: ~$%.2f/day (active days only)\n", totalCost/float64(activeDays)))
		}
		if len(weeks) > 0 {
			sb.WriteString(fmt.Sprintf("**Weekly average**: ~$%.2f/week\n", totalCost/float64(len(weeks))))
		}
		sb.WriteString("\n")
	}

	// Daily Breakdown
	days := aggregateDaily(sessions)
	sb.WriteString("## Daily Breakdown\n\n")
	sb.WriteString("| Date | Sessions | Time | Cost | Top Project |\n")
	sb.WriteString("|------|----------|------|------|-------------|\n")
	for _, d := range days {
		sb.WriteString(fmt.Sprintf("| %s | %d | %s | ~$%.2f | %s |\n",
			d.Date, d.Sessions, formatDuration(d.TimeMin), d.Cost, d.TopProject))
	}
	sb.WriteString("\n")

	return sb.String()
}

// --- Aggregation helpers ---

type projectAgg struct {
	Name     string
	Sessions int
	TimeMin  int
	Cost     float64
	Tools    int
	Commits  int
}

type toolAgg struct {
	Name  string
	Count int
}

type fileAgg struct {
	Path  string
	Count int
}

type dailyAgg struct {
	Date       string
	Sessions   int
	TimeMin    int
	Cost       float64
	TopProject string
}

type weeklyAgg struct {
	Label      string
	Sessions   int
	TimeMin    int
	Cost       float64
	Focus      string
	ToolCounts map[string]int
}

func uniqueDates(sessions []SessionMeta) int {
	dates := make(map[string]bool)
	for _, s := range sessions {
		dates[s.Date] = true
	}
	return len(dates)
}

func sumDuration(sessions []SessionMeta) int {
	total := 0
	for _, s := range sessions {
		total += s.DurationMin
	}
	return total
}

func sumCost(sessions []SessionMeta) float64 {
	total := 0.0
	for _, s := range sessions {
		total += s.CostFloat
	}
	return total
}

func sumTokensIn(sessions []SessionMeta) int {
	total := 0
	for _, s := range sessions {
		total += s.TokensIn
	}
	return total
}

func sumTokensOut(sessions []SessionMeta) int {
	total := 0
	for _, s := range sessions {
		total += s.TokensOut
	}
	return total
}

func sumTools(sessions []SessionMeta) int {
	total := 0
	for _, s := range sessions {
		for _, c := range s.Tools {
			total += c
		}
	}
	return total
}

func sumCommits(sessions []SessionMeta) int {
	total := 0
	for _, s := range sessions {
		total += s.Commits
	}
	return total
}

func aggregateProjects(sessions []SessionMeta) []projectAgg {
	byProject := make(map[string]*projectAgg)
	var order []string
	for _, s := range sessions {
		p, ok := byProject[s.Project]
		if !ok {
			p = &projectAgg{Name: s.Project}
			byProject[s.Project] = p
			order = append(order, s.Project)
		}
		p.Sessions++
		p.TimeMin += s.DurationMin
		p.Cost += s.CostFloat
		for _, c := range s.Tools {
			p.Tools += c
		}
		p.Commits += s.Commits
	}
	sort.Slice(order, func(i, j int) bool {
		a, b := byProject[order[i]], byProject[order[j]]
		if a.Sessions != b.Sessions {
			return a.Sessions > b.Sessions
		}
		return a.Name < b.Name
	})
	result := make([]projectAgg, len(order))
	for i, name := range order {
		result[i] = *byProject[name]
	}
	return result
}

func aggregateToolUsage(sessions []SessionMeta) []toolAgg {
	counts := make(map[string]int)
	for _, s := range sessions {
		for name, count := range s.Tools {
			counts[name] += count
		}
	}
	var tools []toolAgg
	for name, count := range counts {
		tools = append(tools, toolAgg{name, count})
	}
	sort.Slice(tools, func(i, j int) bool {
		if tools[i].Count != tools[j].Count {
			return tools[i].Count > tools[j].Count
		}
		return tools[i].Name < tools[j].Name
	})
	return tools
}

func aggregateFiles(sessions []SessionMeta) []fileAgg {
	counts := make(map[string]int)
	for _, s := range sessions {
		for _, f := range s.FilesTouched {
			counts[f]++
		}
	}
	var files []fileAgg
	for path, count := range counts {
		files = append(files, fileAgg{path, count})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Count != files[j].Count {
			return files[i].Count > files[j].Count
		}
		return files[i].Path < files[j].Path
	})
	return files
}

func aggregateDaily(sessions []SessionMeta) []dailyAgg {
	type dayData struct {
		Sessions   int
		TimeMin    int
		Cost       float64
		ProjectMap map[string]int
	}
	byDate := make(map[string]*dayData)
	var dates []string
	for _, s := range sessions {
		d, ok := byDate[s.Date]
		if !ok {
			d = &dayData{ProjectMap: make(map[string]int)}
			byDate[s.Date] = d
			dates = append(dates, s.Date)
		}
		d.Sessions++
		d.TimeMin += s.DurationMin
		d.Cost += s.CostFloat
		d.ProjectMap[s.Project]++
	}
	sort.Strings(dates)
	result := make([]dailyAgg, len(dates))
	for i, date := range dates {
		d := byDate[date]
		topProject := ""
		topCount := 0
		for name, count := range d.ProjectMap {
			if count > topCount || (count == topCount && name < topProject) {
				topProject = name
				topCount = count
			}
		}
		result[i] = dailyAgg{
			Date: date, Sessions: d.Sessions, TimeMin: d.TimeMin,
			Cost: d.Cost, TopProject: topProject,
		}
	}
	return result
}

func busiestDay(sessions []SessionMeta) (string, int, float64) {
	type dayInfo struct {
		count int
		cost  float64
	}
	byDate := make(map[string]*dayInfo)
	for _, s := range sessions {
		d, ok := byDate[s.Date]
		if !ok {
			d = &dayInfo{}
			byDate[s.Date] = d
		}
		d.count++
		d.cost += s.CostFloat
	}
	bestDate := ""
	bestCount := 0
	bestCost := 0.0
	for date, d := range byDate {
		if d.count > bestCount || (d.count == bestCount && date < bestDate) {
			bestDate = date
			bestCount = d.count
			bestCost = d.cost
		}
	}
	return bestDate, bestCount, bestCost
}

func busiestHour(sessions []SessionMeta) (int, int) {
	byHour := make(map[int]int)
	for _, s := range sessions {
		if len(s.StartTime) >= 2 {
			h, err := strconv.Atoi(s.StartTime[:2])
			if err == nil {
				byHour[h]++
			}
		}
	}
	bestHour := 0
	bestCount := 0
	for hour, count := range byHour {
		if count > bestCount || (count == bestCount && hour < bestHour) {
			bestHour = hour
			bestCount = count
		}
	}
	return bestHour, bestCount
}

func weekOfMonth(day int) int {
	return (day-1)/7 + 1
}

func buildWeeklyBreakdown(sessions []SessionMeta, monthStart time.Time) []weeklyAgg {
	type weekData struct {
		sessions   int
		timeMin    int
		cost       float64
		projectMap map[string]int
		toolCounts map[string]int
	}
	byWeek := make(map[int]*weekData)
	maxWeek := 0

	for _, s := range sessions {
		t, err := time.Parse("2006-01-02", s.Date)
		if err != nil {
			continue
		}
		w := weekOfMonth(t.Day())
		if w > maxWeek {
			maxWeek = w
		}
		d, ok := byWeek[w]
		if !ok {
			d = &weekData{
				projectMap: make(map[string]int),
				toolCounts: make(map[string]int),
			}
			byWeek[w] = d
		}
		d.sessions++
		d.timeMin += s.DurationMin
		d.cost += s.CostFloat
		d.projectMap[s.Project]++
		for name, count := range s.Tools {
			d.toolCounts[name] += count
		}
	}

	lastDay := time.Date(monthStart.Year(), monthStart.Month()+1, 0, 0, 0, 0, 0, monthStart.Location()).Day()
	var result []weeklyAgg
	for w := 1; w <= maxWeek; w++ {
		d := byWeek[w]
		if d == nil {
			d = &weekData{toolCounts: make(map[string]int)}
		}
		focus := ""
		topCount := 0
		for name, count := range d.projectMap {
			if count > topCount || (count == topCount && name < focus) {
				focus = name
				topCount = count
			}
		}
		startDay := (w-1)*7 + 1
		endDay := w * 7
		if endDay > lastDay {
			endDay = lastDay
		}
		label := fmt.Sprintf("%s %d-%d", monthStart.Format("Jan"), startDay, endDay)
		result = append(result, weeklyAgg{
			Label: label, Sessions: d.sessions, TimeMin: d.timeMin,
			Cost: d.cost, Focus: focus, ToolCounts: d.toolCounts,
		})
	}
	return result
}
