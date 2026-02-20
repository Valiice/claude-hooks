package obsidian

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRebuildDailyIndex_SortOrder verifies case-insensitive project sorting matches PS.
func TestRebuildDailyIndex_SortOrder(t *testing.T) {
	// Create temp vault with project dirs matching the real vault
	tmpDir := t.TempDir()
	date := "2026-02-12"

	projects := []struct {
		name     string
		sessions []string // filenames
	}{
		{"Coding", []string{date + "_1152.md"}},
		{"cubilis-claude", []string{date + "_1741.md"}},
		{"MewsPMSClaude", []string{date + "_1157.md"}},
	}

	for _, p := range projects {
		dir := filepath.Join(tmpDir, p.name)
		os.MkdirAll(dir, 0755)
		for _, s := range p.sessions {
			content := "---\ndate: " + date + "\nsession_id: test-" + p.name + "\nproject: " + p.name + "\nstart_time: 11:52\ntags:\n  - claude-session\n---\n\n# Claude Session\n\n---\n\n> [!user]+ #1 - You (11:52:12)\n> test\n\n---\n"
			os.WriteFile(filepath.Join(dir, s), []byte(content), 0644)
		}
	}

	err := RebuildDailyIndex(tmpDir, date)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, date+".md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(content)

	// PS Sort-Object is case-insensitive: Coding < cubilis-claude < MewsPMSClaude
	codingIdx := strings.Index(got, "## Coding")
	cubilisIdx := strings.Index(got, "## cubilis-claude")
	mewsIdx := strings.Index(got, "## MewsPMSClaude")

	if codingIdx < 0 || cubilisIdx < 0 || mewsIdx < 0 {
		t.Fatalf("Missing project sections in:\n%s", got)
	}

	if !(codingIdx < cubilisIdx && cubilisIdx < mewsIdx) {
		t.Errorf("Case-insensitive sort order wrong. Expected Coding < cubilis-claude < MewsPMSClaude\nGot:\n%s", got)
	}
}

// TestRebuildDailyIndex_Format verifies the daily index format matches PS output exactly.
func TestRebuildDailyIndex_Format(t *testing.T) {
	tmpDir := t.TempDir()
	date := "2026-02-12"

	// Single project with one session
	dir := filepath.Join(tmpDir, "Coding")
	os.MkdirAll(dir, 0755)
	sessionContent := "---\ndate: " + date + "\nsession_id: test-id\nproject: Coding\nstart_time: 17:42\nduration: 10min\ntags:\n  - claude-session\n---\n\n# Claude Session\n\n---\n\n> [!user]+ #1 - You (17:42:00)\n> test\n\n---\n"
	os.WriteFile(filepath.Join(dir, date+"_1742.md"), []byte(sessionContent), 0644)

	// Write a temp session file so prompt count is found
	mapFile := filepath.Join(os.TempDir(), "claude_session_test-id.txt")
	os.WriteFile(mapFile, []byte(filepath.Join(dir, date+"_1742.md")+"\n4"), 0644)
	defer os.Remove(mapFile)

	err := RebuildDailyIndex(tmpDir, date)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, date+".md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(content)

	// Expected PS output format
	want := "---\ndate: 2026-02-12\ntags:\n  - claude-daily\n---\n\n# Claude Sessions - 2026-02-12\n" +
		"\n## Coding\n" +
		"- [[Coding/2026-02-12_1742|17:42]] (10min, 4 prompts)\n"

	if got != want {
		t.Errorf("Daily index format mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestRebuildDailyIndex_WithToolsAndCost verifies enhanced format with tools and cost.
func TestRebuildDailyIndex_WithToolsAndCost(t *testing.T) {
	tmpDir := t.TempDir()
	date := "2026-02-12"

	dir := filepath.Join(tmpDir, "Coding")
	os.MkdirAll(dir, 0755)
	sessionContent := "---\ndate: " + date + "\nsession_id: test-id2\nproject: Coding\nstart_time: 17:42\nduration: 10min\ntools:\n  Edit: 12\n  Read: 15\n  Bash: 8\ntokens_in: 45230\ntokens_out: 12840\nestimated_cost: \"$0.23\"\ntags:\n  - claude-session\n---\n\n# Claude Session\n\n---\n\n> [!user]+ #1 - You (17:42:00)\n> test\n\n---\n"
	os.WriteFile(filepath.Join(dir, date+"_1742.md"), []byte(sessionContent), 0644)

	mapFile := filepath.Join(os.TempDir(), "claude_session_test-id2.txt")
	os.WriteFile(mapFile, []byte(filepath.Join(dir, date+"_1742.md")+"\n4"), 0644)
	defer os.Remove(mapFile)

	err := RebuildDailyIndex(tmpDir, date)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, date+".md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(content)

	// Should include tools and cost in the entry
	want := "- [[Coding/2026-02-12_1742|17:42]] (10min, 4 prompts, 35 tools, ~$0.23)\n"
	if !strings.Contains(got, want) {
		t.Errorf("Enhanced daily index format mismatch\ngot:\n%s\nwant line:\n%s", got, want)
	}
}
