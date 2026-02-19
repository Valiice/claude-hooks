package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/valentinclaes/claude-hooks/internal/settings"
)

// Config holds hook settings from config.json and/or .claude/claude-hooks.local.md.
type Config struct {
	SkipWhenFocused bool `json:"skip_when_focused"`
	GitAutoPush     bool `json:"git_auto_push"`
}

func defaults() Config {
	return Config{
		SkipWhenFocused: true,
		GitAutoPush:     false,
	}
}

// Load reads config from ~/.claude/hooks/config.json, then overlays
// any settings from .claude/claude-hooks.local.md (project-level then user-global).
// Returns defaults on any error (missing file, bad JSON, etc.).
func Load() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return applySettings(defaults())
	}
	cfg := loadFrom(filepath.Join(home, ".claude", "hooks", "config.json"))
	return applySettings(cfg)
}

func loadFrom(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaults()
	}

	cfg := defaults()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaults()
	}
	return cfg
}

// applySettings overlays .local.md settings on top of the config.
// Settings file values take priority over config.json when set.
func applySettings(cfg Config) Config {
	s := settings.ReadAll()
	if s.SkipWhenFocused != nil {
		cfg.SkipWhenFocused = *s.SkipWhenFocused
	}
	if s.GitAutoPush != nil {
		cfg.GitAutoPush = *s.GitAutoPush
	}
	return cfg
}
