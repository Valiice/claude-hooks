package gitctx

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// Context holds git context for a session.
type Context struct {
	Branch string
	Hash   string
}

// CommitInfo holds a single commit's info.
type CommitInfo struct {
	Hash    string
	Message string
}

// Capture returns the current branch and short HEAD hash for the given directory.
// Returns empty Context on any error (not a git repo, git not installed, timeout).
func Capture(cwd string) Context {
	defer func() { recover() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	branch := runGit(ctx, cwd, "rev-parse", "--abbrev-ref", "HEAD")
	hash := runGit(ctx, cwd, "rev-parse", "--short", "HEAD")

	return Context{Branch: branch, Hash: hash}
}

// CommitsSince returns commits made between startHash and HEAD.
// Returns nil on any error. Caps at 20 commits.
func CommitsSince(cwd, startHash string) []CommitInfo {
	defer func() { recover() }()

	if startHash == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	output := runGit(ctx, cwd, "log", "--oneline", startHash+"..HEAD")
	if output == "" {
		return nil
	}

	lines := strings.Split(output, "\n")
	var commits []CommitInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		ci := CommitInfo{Hash: parts[0]}
		if len(parts) > 1 {
			ci.Message = parts[1]
		}
		commits = append(commits, ci)
		if len(commits) >= 20 {
			break
		}
	}
	return commits
}

func runGit(ctx context.Context, cwd string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
