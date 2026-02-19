package obsidian

import (
	"strings"
	"testing"
)

// TestBuildFrontmatter_NoResume verifies output matches PowerShell for a new session.
func TestBuildFrontmatter_NoResume(t *testing.T) {
	got := BuildFrontmatter("2026-02-13", "abc-123", "Coding", "13:50", "")
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
	got := BuildFrontmatter("2026-02-13", "abc-123", "Coding", "13:50", "Coding/2026-02-13_1200")
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
	// PS: $project.ToLower() -replace '\s+', '-'
	got := BuildFrontmatter("2026-02-13", "id", "My  Project\tName", "10:00", "")
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

// TestGenerateTitleSlug verifies slug generation from prompt text.
func TestGenerateTitleSlug(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Fix the authentication bug in JWT", "fix-the-authentication-bug-in-jwt"},
		{"hi", "hi"},
		{"", ""},
		{"   ", ""},
		{"Hello, World! How are you doing today and more words", "hello-world-how-are-you-doing"},
		{"Fix: bug #1 (urgent) [NOW]", "fix-bug-1-urgent-now"},
		{"first line\nsecond line should be ignored", "first-line"},
	}
	for _, tc := range cases {
		got := GenerateTitleSlug(tc.input)
		if got != tc.want {
			t.Errorf("GenerateTitleSlug(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestFormatSummaryBlock verifies the summary block format.
func TestFormatSummaryBlock(t *testing.T) {
	if got := FormatSummaryBlock(nil); got != "" {
		t.Errorf("FormatSummaryBlock(nil) = %q, want %q", got, "")
	}
	if got := FormatSummaryBlock([]string{}); got != "" {
		t.Errorf("FormatSummaryBlock([]) = %q, want %q", got, "")
	}
	got := FormatSummaryBlock([]string{"topic one", "topic two"})
	want := "**Topics covered:**\n- topic one\n- topic two\n"
	if got != want {
		t.Errorf("FormatSummaryBlock mismatch\ngot:  %q\nwant: %q", got, want)
	}
}
