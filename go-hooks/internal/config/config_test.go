package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	cfg := loadFrom(filepath.Join(t.TempDir(), "nonexistent.json"))
	if !cfg.SkipWhenFocused {
		t.Error("expected SkipWhenFocused=true for missing file")
	}
	if cfg.GitAutoPush {
		t.Error("expected GitAutoPush=false for missing file")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{"skip_when_focused": false, "git_auto_push": true}`), 0644)

	cfg := loadFrom(path)
	if cfg.SkipWhenFocused {
		t.Error("expected SkipWhenFocused=false")
	}
	if !cfg.GitAutoPush {
		t.Error("expected GitAutoPush=true")
	}
}

func TestLoad_PartialFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{"git_auto_push": true}`), 0644)

	cfg := loadFrom(path)
	if !cfg.SkipWhenFocused {
		t.Error("expected SkipWhenFocused=true (default) when key absent")
	}
	if !cfg.GitAutoPush {
		t.Error("expected GitAutoPush=true")
	}
}

func TestLoad_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{not valid json`), 0644)

	cfg := loadFrom(path)
	if !cfg.SkipWhenFocused {
		t.Error("expected SkipWhenFocused=true for malformed JSON")
	}
	if cfg.GitAutoPush {
		t.Error("expected GitAutoPush=false for malformed JSON")
	}
}
