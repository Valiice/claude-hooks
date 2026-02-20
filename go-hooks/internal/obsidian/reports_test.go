package obsidian

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScanSessions(t *testing.T) {
	tmpDir := t.TempDir()
	projDir := filepath.Join(tmpDir, "TestProject")
	os.MkdirAll(projDir, 0755)

	content := `---
date: 2026-02-17
session_id: abc-123
project: TestProject
start_time: 14:30
duration: 25min
branch: main
tools:
  Read: 10
  Edit: 5
tokens_in: 45000
tokens_out: 12000
estimated_cost: "$0.23"
files_touched:
  - cmd/main.go
  - internal/pkg.go
commits:
  - a1b2c3d Add feature
tags:
  - claude-session
---

# Claude Session - TestProject

---

> [!user]+ #1 - You (14:30:00)
> test prompt

---

> [!user]+ #2 - You (14:35:00)
> another prompt

---
`
	os.WriteFile(filepath.Join(projDir, "2026-02-17_1430.md"), []byte(content), 0644)
	// File outside range
	os.WriteFile(filepath.Join(projDir, "2026-02-10_0900.md"), []byte("---\ndate: 2026-02-10\n---\n"), 0644)

	start := time.Date(2026, 2, 16, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 2, 17, 0, 0, 0, 0, time.Local)

	sessions := ScanSessions(tmpDir, start, end)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.Project != "TestProject" {
		t.Errorf("project: got %q, want %q", s.Project, "TestProject")
	}
	if s.Date != "2026-02-17" {
		t.Errorf("date: got %q, want %q", s.Date, "2026-02-17")
	}
	if s.StartTime != "14:30" {
		t.Errorf("start_time: got %q, want %q", s.StartTime, "14:30")
	}
	if s.DurationMin != 25 {
		t.Errorf("duration_min: got %d, want 25", s.DurationMin)
	}
	if s.Branch != "main" {
		t.Errorf("branch: got %q, want %q", s.Branch, "main")
	}
	if s.Tools["Read"] != 10 || s.Tools["Edit"] != 5 {
		t.Errorf("tools: got %v, want Read:10 Edit:5", s.Tools)
	}
	if s.TokensIn != 45000 {
		t.Errorf("tokens_in: got %d, want 45000", s.TokensIn)
	}
	if s.TokensOut != 12000 {
		t.Errorf("tokens_out: got %d, want 12000", s.TokensOut)
	}
	if s.CostFloat != 0.23 {
		t.Errorf("cost: got %f, want 0.23", s.CostFloat)
	}
	if len(s.FilesTouched) != 2 {
		t.Errorf("files_touched: got %d, want 2", len(s.FilesTouched))
	}
	if s.Commits != 1 {
		t.Errorf("commits: got %d, want 1", s.Commits)
	}
	if s.Prompts != 2 {
		t.Errorf("prompts: got %d, want 2", s.Prompts)
	}
}

func TestScanSessions_CRLF(t *testing.T) {
	tmpDir := t.TempDir()
	projDir := filepath.Join(tmpDir, "TestProject")
	os.MkdirAll(projDir, 0755)

	// Same content as TestScanSessions but with \r\n line endings
	content := "---\r\ndate: 2026-02-17\r\nsession_id: abc-123\r\nproject: TestProject\r\nstart_time: 14:30\r\nduration: 25min\r\nbranch: main\r\ntools:\r\n  Read: 10\r\n  Edit: 5\r\ntokens_in: 45000\r\ntokens_out: 12000\r\nestimated_cost: \"$0.23\"\r\nfiles_touched:\r\n  - cmd/main.go\r\n  - internal/pkg.go\r\ncommits:\r\n  - a1b2c3d Add feature\r\ntags:\r\n  - claude-session\r\n---\r\n\r\n# Claude Session - TestProject\r\n\r\n---\r\n\r\n> [!user]+ #1 - You (14:30:00)\r\n> test prompt\r\n\r\n---\r\n\r\n> [!user]+ #2 - You (14:35:00)\r\n> another prompt\r\n\r\n---\r\n"
	os.WriteFile(filepath.Join(projDir, "2026-02-17_1430.md"), []byte(content), 0644)

	start := time.Date(2026, 2, 16, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 2, 17, 0, 0, 0, 0, time.Local)

	sessions := ScanSessions(tmpDir, start, end)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.StartTime != "14:30" {
		t.Errorf("start_time: got %q, want %q", s.StartTime, "14:30")
	}
	if s.DurationMin != 25 {
		t.Errorf("duration_min: got %d, want 25", s.DurationMin)
	}
	if s.Branch != "main" {
		t.Errorf("branch: got %q, want %q", s.Branch, "main")
	}
	if s.Tools["Read"] != 10 || s.Tools["Edit"] != 5 {
		t.Errorf("tools: got %v, want Read:10 Edit:5", s.Tools)
	}
	if s.TokensIn != 45000 {
		t.Errorf("tokens_in: got %d, want 45000", s.TokensIn)
	}
	if s.CostFloat != 0.23 {
		t.Errorf("cost: got %f, want 0.23", s.CostFloat)
	}
	if len(s.FilesTouched) != 2 {
		t.Errorf("files_touched: got %d, want 2", len(s.FilesTouched))
	}
	if s.Commits != 1 {
		t.Errorf("commits: got %d, want 1", s.Commits)
	}
	if s.Prompts != 2 {
		t.Errorf("prompts: got %d, want 2", s.Prompts)
	}
}

func TestScanSessions_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	sessions := ScanSessions(tmpDir, time.Now(), time.Now())
	if sessions != nil {
		t.Errorf("expected nil, got %v", sessions)
	}
}

func TestParseDurationMin(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"25min", 25},
		{"1h 30min", 90},
		{"2h", 120},
		{"5min", 5},
		{"", 0},
		{"0min", 0},
	}
	for _, tt := range tests {
		got := parseDurationMin(tt.input)
		if got != tt.want {
			t.Errorf("parseDurationMin(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0m"},
		{30, "30m"},
		{60, "~1h"},
		{90, "~1h 30m"},
		{150, "~2h 30m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.input)
		if got != tt.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWeekStart(t *testing.T) {
	tests := []struct {
		name  string
		input time.Time
	}{
		{"monday", time.Date(2026, 2, 16, 12, 30, 0, 0, time.UTC)},
		{"tuesday", time.Date(2026, 2, 17, 12, 30, 0, 0, time.UTC)},
		{"wednesday", time.Date(2026, 2, 18, 12, 30, 0, 0, time.UTC)},
		{"sunday", time.Date(2026, 2, 22, 12, 30, 0, 0, time.UTC)},
		{"saturday", time.Date(2026, 2, 21, 12, 30, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := weekStart(tt.input)
			if got.Weekday() != time.Monday {
				t.Errorf("weekStart(%s) = %s (%s), want Monday",
					tt.input.Format("2006-01-02"), got.Format("2006-01-02"), got.Weekday())
			}
			if got.After(tt.input) {
				t.Errorf("weekStart should not be after input")
			}
			diff := int(tt.input.Sub(got).Hours() / 24)
			if diff > 6 {
				t.Errorf("weekStart too far back: %d days", diff)
			}
			if got.Hour() != 0 || got.Minute() != 0 {
				t.Errorf("weekStart should be midnight, got %s", got.Format("15:04"))
			}
		})
	}
}

func TestIsStaleToday(t *testing.T) {
	now := time.Now()

	// Non-existent file -> stale
	if !isStaleToday("/nonexistent/path.md", now) {
		t.Error("non-existent file should be stale")
	}

	// Fresh file -> not stale
	tmpFile := filepath.Join(t.TempDir(), "test.md")
	os.WriteFile(tmpFile, []byte("test"), 0644)
	if isStaleToday(tmpFile, now) {
		t.Error("just-created file should not be stale today")
	}

	// File from yesterday -> stale
	yesterday := now.AddDate(0, 0, -1)
	os.Chtimes(tmpFile, yesterday, yesterday)
	if !isStaleToday(tmpFile, now) {
		t.Error("yesterday's file should be stale")
	}
}

func TestRebuildWeeklyStatsIfStale_SkipsIfFresh(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Date(2026, 2, 17, 14, 0, 0, 0, time.Local)
	start := weekStart(now)

	fileName := "Weekly-" + start.Format("2006-01-02") + "-to-" + now.Format("2006-01-02") + ".md"
	filePath := filepath.Join(tmpDir, fileName)

	// Create a fresh file (mtime = now)
	os.WriteFile(filePath, []byte("existing content"), 0644)

	err := RebuildWeeklyStatsIfStale(tmpDir, now)
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filePath)
	if string(content) != "existing content" {
		t.Error("fresh file should not be overwritten")
	}
}

func TestRebuildWeeklyStatsIfStale_CleansOldFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// Simulate: yesterday was Feb 18, today is Feb 19 (same week starting Feb 16)
	now := time.Date(2026, 2, 19, 10, 0, 0, 0, time.Local)
	start := weekStart(now)

	// Create an old file from yesterday's run
	oldFileName := "Weekly-" + start.Format("2006-01-02") + "-to-2026-02-18.md"
	oldFilePath := filepath.Join(tmpDir, oldFileName)
	os.WriteFile(oldFilePath, []byte("old content"), 0644)
	// Set mtime to yesterday so it's stale
	yesterday := now.AddDate(0, 0, -1)
	os.Chtimes(oldFilePath, yesterday, yesterday)

	// Run rebuild — no sessions, so no new file, but old file should still be cleaned
	_ = RebuildWeeklyStatsIfStale(tmpDir, now)

	if _, err := os.Stat(oldFilePath); !os.IsNotExist(err) {
		t.Error("old weekly file should have been removed")
	}
}

func TestBuildWeeklyReport(t *testing.T) {
	sessions := []SessionMeta{
		{
			Project: "Alpha", Date: "2026-02-16", StartTime: "09:00",
			Duration: "30min", DurationMin: 30,
			Tools:    map[string]int{"Read": 10, "Edit": 5},
			TokensIn: 20000, TokensOut: 5000,
			CostFloat: 0.10, Commits: 2, Prompts: 3,
		},
		{
			Project: "Alpha", Date: "2026-02-17", StartTime: "14:00",
			Duration: "45min", DurationMin: 45,
			Tools:    map[string]int{"Read": 8, "Bash": 3},
			TokensIn: 30000, TokensOut: 8000,
			CostFloat: 0.15, FilesTouched: []string{"cmd/main.go"},
			Commits: 1, Prompts: 5,
		},
		{
			Project: "Beta", Date: "2026-02-17", StartTime: "16:00",
			Duration: "20min", DurationMin: 20,
			Tools:    map[string]int{"Edit": 12},
			TokensIn: 15000, TokensOut: 4000,
			CostFloat: 0.08, Prompts: 2,
		},
	}

	start := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC)
	report := buildWeeklyReport(sessions, start, end)

	// Frontmatter
	if !strings.Contains(report, "type: weekly-stats") {
		t.Error("missing type: weekly-stats")
	}
	if !strings.Contains(report, "auto_generated: true") {
		t.Error("missing auto_generated: true")
	}

	// Overview
	if !strings.Contains(report, "| Sessions | 3 |") {
		t.Error("missing session count")
	}
	if !strings.Contains(report, "| Active Days | 2/2 |") {
		t.Error("missing active days")
	}
	if !strings.Contains(report, "| Tool Calls | 38 |") {
		t.Errorf("missing tool calls total, report:\n%s", report)
	}
	if !strings.Contains(report, "| Commits | 3 |") {
		t.Error("missing commits total")
	}

	// Projects
	if !strings.Contains(report, "| Alpha |") {
		t.Error("missing Alpha project")
	}
	if !strings.Contains(report, "| Beta |") {
		t.Error("missing Beta project")
	}

	// Tool Usage
	if !strings.Contains(report, "## Tool Usage") {
		t.Error("missing tool usage section")
	}
	if !strings.Contains(report, "| Read |") {
		t.Error("missing Read tool")
	}

	// Daily Breakdown
	if !strings.Contains(report, "| 2026-02-16 |") {
		t.Error("missing daily breakdown for Feb 16")
	}
	if !strings.Contains(report, "| 2026-02-17 |") {
		t.Error("missing daily breakdown for Feb 17")
	}

	// Cost Analysis
	if !strings.Contains(report, "## Cost Analysis") {
		t.Error("missing cost analysis section")
	}

	// Streaks
	if !strings.Contains(report, "## Streaks & Trends") {
		t.Error("missing streaks section")
	}
}

func TestBuildMonthlyReport(t *testing.T) {
	sessions := []SessionMeta{
		{
			Project: "Alpha", Date: "2026-02-03", StartTime: "09:00",
			Duration: "30min", DurationMin: 30,
			Tools:    map[string]int{"Read": 10},
			TokensIn: 20000, TokensOut: 5000,
			CostFloat: 0.10, Prompts: 3,
		},
		{
			Project: "Alpha", Date: "2026-02-10", StartTime: "14:00",
			Duration: "45min", DurationMin: 45,
			Tools:    map[string]int{"Read": 5, "Edit": 8},
			TokensIn: 30000, TokensOut: 8000,
			CostFloat: 0.15, Prompts: 5,
		},
	}

	monthStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	report := buildMonthlyReport(sessions, monthStart)

	// Frontmatter
	if !strings.Contains(report, "type: monthly-stats") {
		t.Error("missing type: monthly-stats")
	}
	if !strings.Contains(report, "month: \"2026-02\"") {
		t.Error("missing month field")
	}

	// Summary
	if !strings.Contains(report, "| Total Sessions | 2 |") {
		t.Error("missing session count")
	}
	if !strings.Contains(report, "| Projects | 1 |") {
		t.Error("missing project count")
	}

	// Weekly Breakdown
	if !strings.Contains(report, "## Weekly Breakdown") {
		t.Error("missing weekly breakdown")
	}

	// Tool Usage Evolution (2 weeks with data -> should appear)
	if !strings.Contains(report, "## Tool Usage Evolution") {
		t.Error("missing tool usage evolution")
	}
	if !strings.Contains(report, "| Read |") {
		t.Error("missing Read in tool evolution")
	}

	// Monthly-specific: daily/weekly averages
	if !strings.Contains(report, "Daily average") {
		t.Errorf("missing daily average in report:\n%s", report)
	}
}
