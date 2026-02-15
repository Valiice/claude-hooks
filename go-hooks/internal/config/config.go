package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds hook settings from ~/.claude/hooks/config.json.
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

// Load reads config from ~/.claude/hooks/config.json.
// Returns defaults on any error (missing file, bad JSON, etc.).
func Load() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaults()
	}
	return loadFrom(filepath.Join(home, ".claude", "hooks", "config.json"))
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
