package gitsync

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/valentinclaes/claude-hooks/internal/config"
)

const lockTimeout = 5 * time.Minute
const syncTimeout = 30 * time.Second

// SyncIfEnabled commits and pushes vault changes if git_auto_push is enabled in config.
// All errors are swallowed silently (matching project convention).
func SyncIfEnabled(vaultDir string) {
	cfg := config.Load()
	if !cfg.GitAutoPush {
		return
	}
	gitRoot := findGitRoot(vaultDir)
	if gitRoot == "" {
		return
	}

	lockPath := filepath.Join(gitRoot, ".git", "claude-sync.lock")
	if !acquireLock(lockPath) {
		return
	}
	defer releaseLock(lockPath)

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	// Stage all changes (use gitRoot so git sees the full repo)
	if err := gitCmd(ctx, gitRoot, "add", "-A"); err != nil {
		return
	}

	// Check if anything staged
	if err := gitCmd(ctx, gitRoot, "diff", "--cached", "--quiet"); err == nil {
		return // exit 0 means nothing staged
	}

	// Commit
	msg := fmt.Sprintf("claude: sync session %s", time.Now().Format("15:04"))
	if err := gitCmd(ctx, gitRoot, "commit", "-m", msg); err != nil {
		return
	}

	// Push (best-effort)
	_ = gitCmd(ctx, gitRoot, "push")
}

// findGitRoot walks up from dir looking for a .git directory.
// Returns the git root path, or "" if not found.
func findGitRoot(dir string) string {
	dir = filepath.Clean(dir)
	for {
		info, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func acquireLock(path string) bool {
	// Check for stale lock
	if info, err := os.Stat(path); err == nil {
		if time.Since(info.ModTime()) > lockTimeout {
			os.Remove(path)
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return false // lock exists, another sync in progress
	}
	f.Close()
	return true
}

func releaseLock(path string) {
	os.Remove(path)
}

func gitCmd(ctx context.Context, dir string, args ...string) error {
	fullArgs := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
