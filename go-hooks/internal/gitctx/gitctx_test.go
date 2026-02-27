package gitctx

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCapture_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	ctx := Capture(dir)
	if ctx.Branch != "" {
		t.Errorf("Branch in non-repo: got %q, want empty", ctx.Branch)
	}
	if ctx.Hash != "" {
		t.Errorf("Hash in non-repo: got %q, want empty", ctx.Hash)
	}
}

func TestCapture_ValidRepo(t *testing.T) {
	dir := setupTestRepo(t)
	ctx := Capture(dir)
	if ctx.Branch == "" {
		t.Error("Branch should not be empty in valid repo")
	}
	if ctx.Hash == "" {
		t.Error("Hash should not be empty in valid repo")
	}
}

func TestCommitsSince(t *testing.T) {
	dir := setupTestRepo(t)

	// Get current hash
	startCtx := Capture(dir)
	startHash := startCtx.Hash

	// Make a new commit
	testFile := filepath.Join(dir, "newfile.txt")
	os.WriteFile(testFile, []byte("new content"), 0644)
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "second commit")

	commits := CommitsSince(dir, startHash)
	if len(commits) == 0 {
		t.Error("Expected at least 1 commit")
	}
	if len(commits) > 0 && commits[0].Message != "second commit" {
		t.Errorf("Commit message: got %q, want %q", commits[0].Message, "second commit")
	}
}

func TestCommitsSince_EmptyHash(t *testing.T) {
	dir := setupTestRepo(t)
	commits := CommitsSince(dir, "")
	if commits != nil {
		t.Error("Expected nil for empty start hash")
	}
}

func setupTestRepo(t *testing.T) string {
	t.Helper()

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@test.com")
	runCmd(t, dir, "git", "config", "user.name", "Test")

	// Create initial commit
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0644)
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "initial commit")

	return dir
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %s %v failed: %v\n%s", name, args, err, out)
	}
}
