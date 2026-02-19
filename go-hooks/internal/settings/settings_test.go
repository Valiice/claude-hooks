package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFrontmatter(t *testing.T) {
	content := "---\nvault_path: C:\\Vault\\Claude\ngit_auto_push: true\n---\n\n# Notes\nSome body text."
	got := extractFrontmatter(content)
	want := "vault_path: C:\\Vault\\Claude\ngit_auto_push: true"
	if got != want {
		t.Errorf("extractFrontmatter:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestExtractFrontmatter_Empty(t *testing.T) {
	got := extractFrontmatter("no frontmatter here")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractFrontmatter_SingleDash(t *testing.T) {
	got := extractFrontmatter("---\nvault_path: test\n")
	if got != "" {
		t.Errorf("expected empty for unclosed frontmatter, got %q", got)
	}
}

func TestParseYAMLLine(t *testing.T) {
	tests := []struct {
		line    string
		wantKey string
		wantVal string
		wantOk  bool
	}{
		{"vault_path: C:\\Vault", "vault_path", "C:\\Vault", true},
		{"git_auto_push: true", "git_auto_push", "true", true},
		{"skip_when_focused: false", "skip_when_focused", "false", true},
		{"vault_path: \"C:\\My Vault\"", "vault_path", "C:\\My Vault", true},
		{"# comment", "", "", false},
		{"", "", "", false},
		{"no-colon", "", "", false},
	}
	for _, tt := range tests {
		key, val, ok := parseYAMLLine(tt.line)
		if key != tt.wantKey || val != tt.wantVal || ok != tt.wantOk {
			t.Errorf("parseYAMLLine(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.line, key, val, ok, tt.wantKey, tt.wantVal, tt.wantOk)
		}
	}
}

func TestParseBool(t *testing.T) {
	trueVals := []string{"true", "True", "TRUE", "yes", "on", "1"}
	for _, v := range trueVals {
		if !parseBool(v) {
			t.Errorf("parseBool(%q) = false, want true", v)
		}
	}
	falseVals := []string{"false", "False", "no", "off", "0", ""}
	for _, v := range falseVals {
		if parseBool(v) {
			t.Errorf("parseBool(%q) = true, want false", v)
		}
	}
}

func TestReadFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-hooks.local.md")
	content := "---\nvault_path: D:\\Obsidian\\Claude\nskip_when_focused: true\ngit_auto_push: false\n---\n\n# Config\n"
	os.WriteFile(path, []byte(content), 0644)

	s := readFrom(path)
	if s.VaultPath != "D:\\Obsidian\\Claude" {
		t.Errorf("VaultPath = %q, want %q", s.VaultPath, "D:\\Obsidian\\Claude")
	}
	if s.SkipWhenFocused == nil || *s.SkipWhenFocused != true {
		t.Errorf("SkipWhenFocused = %v, want true", s.SkipWhenFocused)
	}
	if s.GitAutoPush == nil || *s.GitAutoPush != false {
		t.Errorf("GitAutoPush = %v, want false", s.GitAutoPush)
	}
}

func TestReadFrom_MissingFile(t *testing.T) {
	s := readFrom("/nonexistent/path/file.md")
	if s.VaultPath != "" || s.SkipWhenFocused != nil || s.GitAutoPush != nil {
		t.Errorf("expected zero Settings for missing file, got %+v", s)
	}
}

func TestReadFrom_PartialSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-hooks.local.md")
	content := "---\nvault_path: C:\\Vault\n---\n"
	os.WriteFile(path, []byte(content), 0644)

	s := readFrom(path)
	if s.VaultPath != "C:\\Vault" {
		t.Errorf("VaultPath = %q, want %q", s.VaultPath, "C:\\Vault")
	}
	if s.SkipWhenFocused != nil {
		t.Errorf("SkipWhenFocused should be nil for unset, got %v", s.SkipWhenFocused)
	}
	if s.GitAutoPush != nil {
		t.Errorf("GitAutoPush should be nil for unset, got %v", s.GitAutoPush)
	}
}

func TestMerge(t *testing.T) {
	trueVal := true
	falseVal := false

	dst := Settings{
		VaultPath:       "global-vault",
		SkipWhenFocused: &trueVal,
	}
	src := Settings{
		VaultPath:   "project-vault",
		GitAutoPush: &falseVal,
	}

	merge(&dst, &src)

	if dst.VaultPath != "project-vault" {
		t.Errorf("VaultPath = %q, want %q", dst.VaultPath, "project-vault")
	}
	if dst.SkipWhenFocused == nil || *dst.SkipWhenFocused != true {
		t.Errorf("SkipWhenFocused should remain true from dst")
	}
	if dst.GitAutoPush == nil || *dst.GitAutoPush != false {
		t.Errorf("GitAutoPush should be false from src")
	}
}

func TestReadAll_WithProjectDir(t *testing.T) {
	// Create a temp project dir with settings
	projDir := t.TempDir()
	claudeDir := filepath.Join(projDir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	content := "---\nvault_path: E:\\ProjectVault\ngit_auto_push: true\n---\n"
	os.WriteFile(filepath.Join(claudeDir, settingsFileName), []byte(content), 0644)

	t.Setenv("CLAUDE_PROJECT_DIR", projDir)

	s := ReadAll()
	if s.VaultPath != "E:\\ProjectVault" {
		t.Errorf("VaultPath = %q, want %q", s.VaultPath, "E:\\ProjectVault")
	}
	if s.GitAutoPush == nil || *s.GitAutoPush != true {
		t.Errorf("GitAutoPush should be true")
	}
}

func TestReadFrom_QuotedValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-hooks.local.md")
	content := "---\nvault_path: \"C:\\My Obsidian Vault\\Claude\"\n---\n"
	os.WriteFile(path, []byte(content), 0644)

	s := readFrom(path)
	if s.VaultPath != "C:\\My Obsidian Vault\\Claude" {
		t.Errorf("VaultPath = %q, want %q", s.VaultPath, "C:\\My Obsidian Vault\\Claude")
	}
}
