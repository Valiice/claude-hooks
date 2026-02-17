package obsidian

import (
	"strings"
	"testing"
)

// TestBuildFrontmatter_NoResume verifies output matches PowerShell for a new session.
func TestBuildFrontmatter_NoResume(t *testing.T) {
	got := BuildFrontmatter(FrontmatterData{
		Date: "2026-02-13", SessionID: "abc-123", Project: "Coding", StartTime: "13:50",
	})
	want := "---\n" +
		"date: 2026-02-13\n" +
		"session_id: abc-123\n" +
		"project: Coding\n" +
		"start_time: 13:50\n" +
		"tags:\n" +
		"  - claude-session\n" +
		"  - coding\n" +
		"---\n" +
		"\n# Claude Session - Coding\n" +
		"\n---\n"
	if got != want {
		t.Errorf("BuildFrontmatter (no resume) mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestBuildFrontmatter_WithResume verifies resumed session output.
func TestBuildFrontmatter_WithResume(t *testing.T) {
	got := BuildFrontmatter(FrontmatterData{
		Date: "2026-02-13", SessionID: "abc-123", Project: "Coding", StartTime: "13:50",
		ResumedFrom: "Coding/2026-02-13_1200",
	})
	want := "---\n" +
		"date: 2026-02-13\n" +
		"session_id: abc-123\n" +
		"project: Coding\n" +
		"start_time: 13:50\n" +
		"resumed_from: \"[[Coding/2026-02-13_1200]]\"\n" +
		"tags:\n" +
		"  - claude-session\n" +
		"  - coding\n" +
		"---\n" +
		"\n# Claude Session - Coding\n" +
		"Resumed from [[Coding/2026-02-13_1200|2026-02-13_1200]]\n" +
		"\n---\n"
	if got != want {
		t.Errorf("BuildFrontmatter (resume) mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestFormatPromptEntry verifies prompt callout matches PS format exactly.
func TestFormatPromptEntry(t *testing.T) {
	got := FormatPromptEntry(1, "13:50:22", `C:\Coding`, "can you help me")
	want := "\n> [!user]+ #1 - You (13:50:22)\n" +
		"> **cwd**: ``C:\\Coding``\n" +
		">\n" +
		"> can you help me\n" +
		"\n---\n"
	if got != want {
		t.Errorf("FormatPromptEntry mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestFormatPromptEntry_MultiLine verifies multi-line prompt gets > prefix on each line.
func TestFormatPromptEntry_MultiLine(t *testing.T) {
	got := FormatPromptEntry(2, "14:00:00", `C:\Work`, "line one\nline two\nline three")
	want := "\n> [!user]+ #2 - You (14:00:00)\n" +
		"> **cwd**: ``C:\\Work``\n" +
		">\n" +
		"> line one\n> line two\n> line three\n" +
		"\n---\n"
	if got != want {
		t.Errorf("FormatPromptEntry multiline mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestFormatResponseEntry verifies response callout matches PS format.
func TestFormatResponseEntry(t *testing.T) {
	got := FormatResponseEntry("13:52:50", "Done. All changes applied.")
	want := "\n> [!claude]- Claude (13:52:50)\n" +
		"> Done. All changes applied.\n" +
		"\n---\n"
	if got != want {
		t.Errorf("FormatResponseEntry mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestFormatPlanEntry verifies plan callout matches PS format.
func TestFormatPlanEntry(t *testing.T) {
	got := FormatPlanEntry("13:52:50", "Step 1: Do X\nStep 2: Do Y")
	want := "\n> [!plan]- Claude's Plan (13:52:50)\n" +
		"> Step 1: Do X\n> Step 2: Do Y\n" +
		"\n---\n"
	if got != want {
		t.Errorf("FormatPlanEntry mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestTruncate verifies truncation with char count matches PS behavior.
func TestTruncate(t *testing.T) {
	short := "hello"
	if got := Truncate(short, 5000); got != short {
		t.Errorf("Truncate short: got %q, want %q", got, short)
	}

	long := strings.Repeat("x", 5001)
	got := Truncate(long, 5000)
	want := strings.Repeat("x", 5000) + "\n\n... (truncated, 5001 chars total)"
	if got != want {
		t.Errorf("Truncate long: got length %d, want length %d", len(got), len(want))
	}
}

// TestTruncateSimple verifies plan truncation matches PS behavior.
func TestTruncateSimple(t *testing.T) {
	long := strings.Repeat("y", 5001)
	got := TruncateSimple(long, 5000)
	want := strings.Repeat("y", 5000) + "\n\n... (truncated)"
	if got != want {
		t.Errorf("TruncateSimple mismatch: got length %d, want length %d", len(got), len(want))
	}
}

// TestProjectTag_WhitespaceCollapse verifies projectTag handles tabs/multiple spaces.
func TestProjectTag_WhitespaceCollapse(t *testing.T) {
	got := BuildFrontmatter(FrontmatterData{
		Date: "2026-02-13", SessionID: "id", Project: "My  Project\tName", StartTime: "10:00",
	})
	if !strings.Contains(got, "  - my-project-name\n") {
		t.Errorf("projectTag should collapse whitespace to single hyphen, got:\n%s", got)
	}
}

// TestStripSystemTags verifies system tag removal matches PS behavior.
func TestStripSystemTags(t *testing.T) {
	input := "Hello <system-reminder>hidden stuff</system-reminder> world"
	got := StripSystemTags(input)
	want := "Hello  world"
	if got != want {
		t.Errorf("StripSystemTags: got %q, want %q", got, want)
	}
}

// === New Phase 1/2 tests ===

// TestBuildFrontmatter_WithTools verifies tool counts in frontmatter.
func TestBuildFrontmatter_WithTools(t *testing.T) {
	got := BuildFrontmatter(FrontmatterData{
		Date: "2026-02-13", SessionID: "abc-123", Project: "Coding", StartTime: "13:50",
		Tools:    map[string]int{"Edit": 12, "Read": 15},
		TokensIn: 45230, TokensOut: 12840,
		EstCost: "$0.23",
	})
	if !strings.Contains(got, "tools:\n  Edit: 12\n  Read: 15\n") {
		t.Errorf("Missing tools in frontmatter:\n%s", got)
	}
	if !strings.Contains(got, "tokens_in: 45230\n") {
		t.Errorf("Missing tokens_in:\n%s", got)
	}
	if !strings.Contains(got, "tokens_out: 12840\n") {
		t.Errorf("Missing tokens_out:\n%s", got)
	}
	if !strings.Contains(got, "estimated_cost: \"$0.23\"\n") {
		t.Errorf("Missing estimated_cost:\n%s", got)
	}
}

// TestBuildFrontmatter_WithBranch verifies branch in frontmatter.
func TestBuildFrontmatter_WithBranch(t *testing.T) {
	got := BuildFrontmatter(FrontmatterData{
		Date: "2026-02-13", SessionID: "abc-123", Project: "Coding", StartTime: "13:50",
		Branch: "feature/new-thing",
	})
	if !strings.Contains(got, "branch: feature/new-thing\n") {
		t.Errorf("Missing branch in frontmatter:\n%s", got)
	}
}

// TestBuildFrontmatter_WithFilesTouched verifies files_touched in frontmatter.
func TestBuildFrontmatter_WithFilesTouched(t *testing.T) {
	got := BuildFrontmatter(FrontmatterData{
		Date: "2026-02-13", SessionID: "abc-123", Project: "Coding", StartTime: "13:50",
		FilesTouched: []string{"cmd/main.go", "internal/pkg/pkg.go"},
	})
	if !strings.Contains(got, "files_touched:\n  - cmd/main.go\n  - internal/pkg/pkg.go\n") {
		t.Errorf("Missing files_touched in frontmatter:\n%s", got)
	}
}

// TestBuildFrontmatter_WithModelAndCache verifies model and cache fields in frontmatter.
func TestBuildFrontmatter_WithModelAndCache(t *testing.T) {
	got := BuildFrontmatter(FrontmatterData{
		Date: "2026-02-17", SessionID: "abc-123", Project: "Coding", StartTime: "14:30",
		Model:         "opus",
		TokensIn:      3000, TokensOut: 1500,
		CacheRead:     125000, CacheCreation: 14829,
		EstCost: "$0.10",
	})
	if !strings.Contains(got, "model: opus\n") {
		t.Errorf("Missing model in frontmatter:\n%s", got)
	}
	if !strings.Contains(got, "cache_read: 125000\n") {
		t.Errorf("Missing cache_read in frontmatter:\n%s", got)
	}
	if !strings.Contains(got, "cache_creation: 14829\n") {
		t.Errorf("Missing cache_creation in frontmatter:\n%s", got)
	}
}

// TestBuildFrontmatter_ZeroStatsOmitted verifies zero stats are not written.
func TestBuildFrontmatter_ZeroStatsOmitted(t *testing.T) {
	got := BuildFrontmatter(FrontmatterData{
		Date: "2026-02-13", SessionID: "abc-123", Project: "Coding", StartTime: "13:50",
	})
	if strings.Contains(got, "tools:") {
		t.Error("Empty tools should be omitted")
	}
	if strings.Contains(got, "tokens_in:") {
		t.Error("Zero tokens_in should be omitted")
	}
	if strings.Contains(got, "estimated_cost:") {
		t.Error("Empty estimated_cost should be omitted")
	}
	if strings.Contains(got, "model:") {
		t.Error("Empty model should be omitted")
	}
	if strings.Contains(got, "cache_read:") {
		t.Error("Zero cache_read should be omitted")
	}
	if strings.Contains(got, "cache_creation:") {
		t.Error("Zero cache_creation should be omitted")
	}
}

// TestFormatStatsLine verifies the compact stats summary.
func TestFormatStatsLine(t *testing.T) {
	tools := map[string]int{"Edit": 12, "Read": 15, "Bash": 8}
	got := FormatStatsLine(tools, 45230, 12840, "$0.23")
	if !strings.Contains(got, "**35 tool calls**") {
		t.Errorf("Missing tool count in stats line:\n%s", got)
	}
	if !strings.Contains(got, "**45K in / 12K out tokens**") {
		t.Errorf("Missing token counts in stats line:\n%s", got)
	}
	if !strings.Contains(got, "~$0.23") {
		t.Errorf("Missing cost in stats line:\n%s", got)
	}
	if !strings.Contains(got, "Read(15)") {
		t.Errorf("Missing tool breakdown in stats line:\n%s", got)
	}
}

// TestFormatStatsLine_Empty returns empty for zero stats.
func TestFormatStatsLine_Empty(t *testing.T) {
	got := FormatStatsLine(nil, 0, 0, "")
	if got != "" {
		t.Errorf("Expected empty stats line for zero stats, got: %q", got)
	}
}

// TestFormatCommitsEntry verifies git commit callout format.
func TestFormatCommitsEntry(t *testing.T) {
	got := FormatCommitsEntry("14:52", []string{"a1b2c3d Add git context", "e4f5g6h Update frontmatter"})
	if !strings.Contains(got, "[!git]- Commits (14:52)") {
		t.Errorf("Missing git callout header:\n%s", got)
	}
	if !strings.Contains(got, "> - `a1b2c3d Add git context`") {
		t.Errorf("Missing first commit:\n%s", got)
	}
}

// TestFormatCommitsEntry_Empty returns empty for no commits.
func TestFormatCommitsEntry_Empty(t *testing.T) {
	got := FormatCommitsEntry("14:52", nil)
	if got != "" {
		t.Errorf("Expected empty for no commits, got: %q", got)
	}
}

// TestUpdateFrontmatterStats verifies in-place frontmatter update.
func TestUpdateFrontmatterStats(t *testing.T) {
	content := "---\ndate: 2026-02-13\nsession_id: abc-123\nproject: Coding\nstart_time: 13:50\ntags:\n  - claude-session\n  - coding\n---\n\n# Claude Session - Coding\n"
	got := UpdateFrontmatterStats(content, map[string]int{"Edit": 5}, 1000, 500, 0, 0, "$0.02", nil, nil, "", "")
	if !strings.Contains(got, "tools:\n  Edit: 5\n") {
		t.Errorf("Missing tools in updated frontmatter:\n%s", got)
	}
	if !strings.Contains(got, "tokens_in: 1000\n") {
		t.Errorf("Missing tokens_in:\n%s", got)
	}
	// Should still have the rest intact
	if !strings.Contains(got, "tags:\n  - claude-session\n") {
		t.Errorf("Tags should be preserved:\n%s", got)
	}
	if !strings.Contains(got, "# Claude Session - Coding") {
		t.Errorf("Content after frontmatter should be preserved:\n%s", got)
	}
}

// TestUpdateFrontmatterStats_Idempotent verifies re-updating replaces old stats.
func TestUpdateFrontmatterStats_Idempotent(t *testing.T) {
	content := "---\ndate: 2026-02-13\nsession_id: abc-123\nproject: Coding\nstart_time: 13:50\ntools:\n  Edit: 3\ntokens_in: 500\ntags:\n  - claude-session\n---\n\n# Session\n"
	got := UpdateFrontmatterStats(content, map[string]int{"Edit": 10, "Read": 5}, 2000, 1000, 0, 0, "$0.05", nil, nil, "", "")
	if !strings.Contains(got, "tools:\n  Edit: 10\n  Read: 5\n") {
		t.Errorf("Tools should be replaced:\n%s", got)
	}
	if !strings.Contains(got, "tokens_in: 2000\n") {
		t.Errorf("tokens_in should be updated:\n%s", got)
	}
	// Old values should be gone
	if strings.Contains(got, "tokens_in: 500") {
		t.Errorf("Old tokens_in should be replaced:\n%s", got)
	}
}

// TestUpdateFrontmatterStats_CleansColonLines verifies tool entries and Windows
// file paths (which contain colons) are properly removed on re-update.
func TestUpdateFrontmatterStats_CleansColonLines(t *testing.T) {
	// Simulate frontmatter after a prior update — tools have colons, files have Windows paths
	content := "---\ndate: 2026-02-17\nsession_id: abc\nproject: Test\nstart_time: 04:07\n" +
		"tools:\n  Bash: 1\n  Edit: 6\n  Read: 10\n" +
		"tokens_in: 20000\ntokens_out: 5000\n" +
		"estimated_cost: \"$0.10\"\n" +
		"files_touched:\n  - B:/Downloads/Coding/foo.go\n  - B:/Downloads/Coding/bar.go\n" +
		"commits:\n  - a1b2c3d Fix: something\n" +
		"branch: main\n" +
		"tags:\n  - claude-session\n---\n\n# Session\n"

	got := UpdateFrontmatterStats(content,
		map[string]int{"Bash": 5, "Edit": 12},
		40000, 10000, 5000, 2000, "$0.20",
		[]string{"B:/New/path.go"}, []string{"b2c3d4e New commit"}, "develop", "opus",
	)

	// New values should be present
	if !strings.Contains(got, "tools:\n  Bash: 5\n  Edit: 12\n") {
		t.Errorf("New tools missing:\n%s", got)
	}
	if !strings.Contains(got, "tokens_in: 40000\n") {
		t.Errorf("New tokens_in missing:\n%s", got)
	}
	if !strings.Contains(got, "cache_read: 5000\n") {
		t.Errorf("New cache_read missing:\n%s", got)
	}
	if !strings.Contains(got, "cache_creation: 2000\n") {
		t.Errorf("New cache_creation missing:\n%s", got)
	}
	if !strings.Contains(got, "model: opus\n") {
		t.Errorf("New model missing:\n%s", got)
	}
	if !strings.Contains(got, "files_touched:\n  - B:/New/path.go\n") {
		t.Errorf("New files_touched missing:\n%s", got)
	}
	if !strings.Contains(got, "branch: develop\n") {
		t.Errorf("New branch missing:\n%s", got)
	}

	// Old values must be completely gone
	if strings.Contains(got, "Read: 10") {
		t.Errorf("Old tool Read:10 should be removed:\n%s", got)
	}
	if strings.Contains(got, "foo.go") {
		t.Errorf("Old file foo.go should be removed:\n%s", got)
	}
	if strings.Contains(got, "bar.go") {
		t.Errorf("Old file bar.go should be removed:\n%s", got)
	}
	if strings.Contains(got, "a1b2c3d") {
		t.Errorf("Old commit should be removed:\n%s", got)
	}
	if strings.Contains(got, "branch: main") {
		t.Errorf("Old branch should be removed:\n%s", got)
	}

	// Tags must survive
	if !strings.Contains(got, "tags:\n  - claude-session\n") {
		t.Errorf("Tags should be preserved:\n%s", got)
	}
}
