package gitsync

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// setConfigHome points ~/.claude/hooks/config.json to a temp dir with the given JSON.
func setConfigHome(t *testing.T, json string) {
	t.Helper()
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".claude", "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "config.json"), []byte(json), 0644)
	t.Setenv("USERPROFILE", dir) // os.UserHomeDir() reads this on Windows
	t.Setenv("HOME", dir)        // os.UserHomeDir() reads this on Unix
}

func TestFindGitRoot(t *testing.T) {
	// Directory with .git — returns itself
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".git"), 0755)
	if got := findGitRoot(dir); got != filepath.Clean(dir) {
		t.Errorf("expected %q, got %q", dir, got)
	}

	// Subdirectory — walks up to parent with .git
	sub := filepath.Join(dir, "sub", "deep")
	os.MkdirAll(sub, 0755)
	if got := findGitRoot(sub); got != filepath.Clean(dir) {
		t.Errorf("expected %q, got %q", dir, got)
	}

	// Directory without .git anywhere — returns ""
	dir2 := t.TempDir()
	if got := findGitRoot(dir2); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestSyncIfEnabled_NotEnabled(t *testing.T) {
	setConfigHome(t, `{"git_auto_push": false}`)
	// Should return immediately without error even with invalid dir
	SyncIfEnabled("/nonexistent/path")
}

func TestSyncIfEnabled_NotGitRepo(t *testing.T) {
	setConfigHome(t, `{"git_auto_push": true}`)
	dir := t.TempDir()
	// No .git dir — should return immediately
	SyncIfEnabled(dir)
}

func initBareAndClone(t *testing.T) (bare, clone string) {
	t.Helper()
	bare = filepath.Join(t.TempDir(), "bare.git")
	clone = filepath.Join(t.TempDir(), "work")

	run(t, "", "git", "init", "--bare", bare)
	run(t, "", "git", "clone", bare, clone)
	run(t, clone, "git", "config", "user.email", "test@test.com")
	run(t, clone, "git", "config", "user.name", "Test")

	// Create initial commit so push works
	os.WriteFile(filepath.Join(clone, "init.txt"), []byte("init"), 0644)
	run(t, clone, "git", "add", "-A")
	run(t, clone, "git", "commit", "-m", "initial")
	run(t, clone, "git", "push")

	return bare, clone
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func TestSyncIfEnabled_CommitsAndPushes(t *testing.T) {
	setConfigHome(t, `{"git_auto_push": true}`)
	bare, clone := initBareAndClone(t)

	// Create a new file in the clone
	os.WriteFile(filepath.Join(clone, "session.md"), []byte("# Session"), 0644)

	SyncIfEnabled(clone)

	// Verify commit exists in clone
	cmd := exec.Command("git", "-C", clone, "log", "--oneline", "-1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if got := string(out); !contains(got, "claude: sync session") {
		t.Errorf("expected commit message containing 'claude: sync session', got: %s", got)
	}

	// Verify pushed to bare
	cmd = exec.Command("git", "-C", bare, "log", "--oneline", "-1")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log on bare failed: %v\n%s", err, out)
	}
	if got := string(out); !contains(got, "claude: sync session") {
		t.Errorf("expected pushed commit in bare, got: %s", got)
	}
}

func TestSyncIfEnabled_FromSubdirectory(t *testing.T) {
	setConfigHome(t, `{"git_auto_push": true}`)
	bare, clone := initBareAndClone(t)

	// Create a subdirectory (like CLAUDE_VAULT pointing to a subfolder)
	sub := filepath.Join(clone, "Claude")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "session.md"), []byte("# Session"), 0644)

	// Pass the subdirectory — should walk up and find the git root
	SyncIfEnabled(sub)

	// Verify commit exists
	cmd := exec.Command("git", "-C", clone, "log", "--oneline", "-1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, out)
	}
	if got := string(out); !contains(got, "claude: sync session") {
		t.Errorf("expected commit message containing 'claude: sync session', got: %s", got)
	}

	// Verify pushed to bare
	cmd = exec.Command("git", "-C", bare, "log", "--oneline", "-1")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log on bare failed: %v\n%s", err, out)
	}
	if got := string(out); !contains(got, "claude: sync session") {
		t.Errorf("expected pushed commit in bare, got: %s", got)
	}
}

func TestSyncIfEnabled_NothingToCommit(t *testing.T) {
	setConfigHome(t, `{"git_auto_push": true}`)
	_, clone := initBareAndClone(t)

	// No changes — should complete without error
	SyncIfEnabled(clone)

	// Verify no new commit beyond the initial one
	cmd := exec.Command("git", "-C", clone, "rev-list", "--count", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list failed: %v\n%s", err, out)
	}
	if got := string(out); !contains(got, "1") {
		t.Errorf("expected 1 commit, got: %s", got)
	}
}

func TestAcquireLock_Exclusive(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// First acquire should succeed
	if !acquireLock(lockPath) {
		t.Fatal("first acquire should succeed")
	}

	// Second acquire should fail (lock held)
	if acquireLock(lockPath) {
		t.Fatal("second acquire should fail while lock held")
	}

	// Release and reacquire
	releaseLock(lockPath)
	if !acquireLock(lockPath) {
		t.Fatal("acquire after release should succeed")
	}
	releaseLock(lockPath)
}

func TestAcquireLock_StaleRemoval(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// Create a lock file with old mtime
	os.WriteFile(lockPath, []byte{}, 0644)
	old := time.Now().Add(-(lockTimeout + time.Minute))
	os.Chtimes(lockPath, old, old)

	// Should remove stale lock and acquire
	if !acquireLock(lockPath) {
		t.Fatal("should acquire after removing stale lock")
	}
	releaseLock(lockPath)
}

// cloneRepo clones bare into path and configures git user identity.
func cloneRepo(t *testing.T, bare, path string) {
	t.Helper()
	run(t, "", "git", "clone", bare, path)
	run(t, path, "git", "config", "user.email", "test@test.com")
	run(t, path, "git", "config", "user.name", "Test")
}

func TestSyncIfEnabled_PushRejectedThenRecovered(t *testing.T) {
	setConfigHome(t, `{"git_auto_push": true}`)
	bare, clone1 := initBareAndClone(t)

	// Device A: push a new commit directly (simulating another device already pushed)
	os.WriteFile(filepath.Join(clone1, "device_a.md"), []byte("device A"), 0644)
	run(t, clone1, "git", "add", "-A")
	run(t, clone1, "git", "commit", "-m", "device A commit")
	run(t, clone1, "git", "push")

	// Device B: a fresh clone that hasn't pulled device A's commit
	clone2 := filepath.Join(t.TempDir(), "device_b")
	cloneRepo(t, bare, clone2)

	// Simulate device B getting behind: device A pushes another commit after clone2 was cloned
	os.WriteFile(filepath.Join(clone1, "device_a2.md"), []byte("device A again"), 0644)
	run(t, clone1, "git", "add", "-A")
	run(t, clone1, "git", "commit", "-m", "device A second commit")
	run(t, clone1, "git", "push")

	// Device B writes its own file and calls SyncIfEnabled (push will be rejected, then recovered)
	os.WriteFile(filepath.Join(clone2, "device_b.md"), []byte("device B"), 0644)
	SyncIfEnabled(clone2)

	// Verify device B's commit reached bare
	cmd := exec.Command("git", "-C", bare, "log", "--oneline")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log on bare failed: %v\n%s", err, out)
	}
	log := string(out)
	if !contains(log, "claude: sync session") {
		t.Errorf("expected device B's sync commit in bare, got:\n%s", log)
	}
	// Both device A commits should also be present
	if !contains(log, "device A second commit") {
		t.Errorf("expected device A's commit in bare, got:\n%s", log)
	}
}

func TestSyncIfEnabled_PullConflict_AbortsCleanly(t *testing.T) {
	setConfigHome(t, `{"git_auto_push": true}`)
	bare, clone1 := initBareAndClone(t)

	// Both clones will modify the same line of the same file — unresolvable conflict
	sharedFile := "shared.md"
	// Write initial version of the shared file from clone1
	os.WriteFile(filepath.Join(clone1, sharedFile), []byte("original line\n"), 0644)
	run(t, clone1, "git", "add", "-A")
	run(t, clone1, "git", "commit", "-m", "add shared file")
	run(t, clone1, "git", "push")

	// Clone2 clones after the shared file exists
	clone2 := filepath.Join(t.TempDir(), "device_b")
	cloneRepo(t, bare, clone2)

	// Device A (clone1) modifies the shared file and pushes
	os.WriteFile(filepath.Join(clone1, sharedFile), []byte("device A line\n"), 0644)
	run(t, clone1, "git", "add", "-A")
	run(t, clone1, "git", "commit", "-m", "device A changes shared file")
	run(t, clone1, "git", "push")

	// Device B (clone2) modifies the same line — will conflict on rebase
	os.WriteFile(filepath.Join(clone2, sharedFile), []byte("device B line\n"), 0644)

	// SyncIfEnabled should handle the conflict gracefully without panicking
	SyncIfEnabled(clone2)

	// Verify no rebase is in progress (clean state)
	rebaseHead := filepath.Join(clone2, ".git", "REBASE_HEAD")
	if _, err := os.Stat(rebaseHead); err == nil {
		t.Error("expected no REBASE_HEAD after SyncIfEnabled (rebase should have been aborted)")
	}

	// Verify no merge in progress either
	mergeHead := filepath.Join(clone2, ".git", "MERGE_HEAD")
	if _, err := os.Stat(mergeHead); err == nil {
		t.Error("expected no MERGE_HEAD after SyncIfEnabled")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
