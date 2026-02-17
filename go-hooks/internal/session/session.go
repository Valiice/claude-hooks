package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SessionData holds the session-to-file mapping.
type SessionData struct {
	FilePath  string
	PromptNum int
	Branch    string
	StartHash string
	Cwd       string
}

func mapPath(sessionID string) string {
	return filepath.Join(os.TempDir(), "claude_session_"+sessionID+".txt")
}

// Read reads the session mapping file. Returns nil if not found.
// Handles both old (2-line) and new (4-line) formats gracefully.
func Read(sessionID string) (*SessionData, error) {
	data, err := os.ReadFile(mapPath(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid session map format")
	}
	num, err := strconv.Atoi(strings.TrimSpace(lines[1]))
	if err != nil {
		return nil, err
	}
	sd := &SessionData{FilePath: strings.TrimSpace(lines[0]), PromptNum: num}
	// Extended format: branch and start hash on lines 3-4, cwd on line 5
	if len(lines) >= 4 {
		sd.Branch = strings.TrimSpace(lines[2])
		sd.StartHash = strings.TrimSpace(lines[3])
	}
	if len(lines) >= 5 {
		sd.Cwd = strings.TrimSpace(lines[4])
	}
	return sd, nil
}

// Write writes the session mapping file.
// Format: filepath\npromptNum\nbranch\nstartHash\ncwd (5 lines, UTF-8 no BOM).
func Write(sessionID, filePath string, promptNum int, extra ...string) error {
	content := filePath + "\n" + strconv.Itoa(promptNum)
	// Optional: branch, startHash, cwd
	get := func(i int) string {
		if i < len(extra) {
			return extra[i]
		}
		return ""
	}
	content += "\n" + get(0) + "\n" + get(1) + "\n" + get(2)
	return os.WriteFile(mapPath(sessionID), []byte(content), 0644)
}

// CleanupStale removes session temp files older than 24 hours.
func CleanupStale() {
	pattern := filepath.Join(os.TempDir(), "claude_session_*.txt")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(m)
		}
	}
}
