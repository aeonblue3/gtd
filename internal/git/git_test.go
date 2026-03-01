package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper: Create a temporary git repo with some files to commit
func setupGitRepoWithFiles(t *testing.T) string {
	tmpDir := setupTestRepo(t)

	// Create a test file so we have something to commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0o644)

	return tmpDir
}

func setupEmptyTestRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	// Initialize git repo
	_, _, err := ExecGit(tmpDir, "init")
	if err != nil {
		t.Fatalf("failed to init test repo: %v", err)
	}

	// Configure git (needed for commits)
	ExecGit(tmpDir, "config", "user.email", "test@example.com")
	ExecGit(tmpDir, "config", "user.name", "Test User")

	return tmpDir
}

// helper: Create a temporary git repo for testing
func setupTestRepo(t *testing.T) string {
	tmpDir := setupEmptyTestRepo(t)

	// ADD THIS: Create an initial commit so the branch exists
	testFile := filepath.Join(tmpDir, "initial.txt")
	os.WriteFile(testFile, []byte("initial commit"), 0o644)
	ExecGit(tmpDir, "add", "initial.txt")
	ExecGit(tmpDir, "commit", "-m", "initial commit")

	return tmpDir
}

// Test 1: ExecGit basic success
func TestExecGitSuccess(t *testing.T) {
	tmpDir := t.TempDir()

	// This should work in any directory
	stdout, stderr, err := ExecGit(tmpDir, "--version")
	if err != nil {
		t.Fatalf("ExecGit failed: %v", err)
	}

	if stdout == "" {
		t.Error("expected git version in stdout")
	}

	if stderr != "" {
		t.Error("expected no error output")
	}
}

// Test 2: ExecGit captures stderr
func TestExecGitStderr(t *testing.T) {
	tmpDir := t.TempDir()

	// Run invalid command to get stderr
	_, stderr, err := ExecGit(tmpDir, "invalid-command")

	if err == nil {
		t.Error("expected error for invalid command")
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}

// Test 3: IsGitRepo returns true for git repos
func TestIsGitRepoTrue(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	ExecGit(tmpDir, "init")

	if !IsGitRepo(tmpDir) {
		t.Error("IsGitRepo should return true for git repo")
	}
}

// Test 4: IsGitRepo returns false for non-git dirs
func TestIsGitRepoFalse(t *testing.T) {
	tmpDir := t.TempDir()

	// Don't init git, so it's not a repo
	if IsGitRepo(tmpDir) {
		t.Error("IsGitRepo should return false for non-git directory")
	}
}

// Test 5: GetCurrentBranch works
func TestGetCurrentBranch(t *testing.T) {
	tmpDir := setupTestRepo(t)

	branch, err := GetCurrentBranch(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	// New repos default to 'master' or 'main'
	if branch != "master" && branch != "main" {
		t.Errorf("unexpected branch: %s", branch)
	}
}

// Test 6: GetLastCommitTime returns 0 for empty repo
func TestGetLastCommitTimeEmpty(t *testing.T) {
	tmpDir := setupEmptyTestRepo(t)

	timestamp, err := GetLastCommitTime(tmpDir)
	if err != nil {
		t.Fatalf("GetLastCommitTime failed: %v", err)
	}

	if timestamp != 0 {
		t.Errorf("empty repo should return 0, got %d", timestamp)
	}
}

// Test 7: GetLastCommitTime returns timestamp after commit
func TestGetLastCommitTimeWithCommit(t *testing.T) {
	tmpDir := setupTestRepo(t)

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0o644)
	ExecGit(tmpDir, "add", "test.txt")
	ExecGit(tmpDir, "commit", "-m", "test commit")

	timestamp, err := GetLastCommitTime(tmpDir)
	if err != nil {
		t.Fatalf("GetLastCommitTime failed: %v", err)
	}

	if timestamp == 0 {
		t.Error("should return non-zero timestamp after commit")
	}
}

// ============================================================================
// CommitChanges Tests
// ============================================================================

// Test 1: CommitChanges successfully commits changes
func TestCommitChangesSuccess(t *testing.T) {
	tmpDir := setupGitRepoWithFiles(t)

	// Create a new file to commit
	newFile := filepath.Join(tmpDir, "new.txt")
	os.WriteFile(newFile, []byte("new content"), 0o644)

	// Commit the changes
	err := CommitChanges(tmpDir, "test: add new file")
	if err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Verify commit was made by checking git log
	stdout, _, err := ExecGit(tmpDir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("failed to check git log: %v", err)
	}

	if strings.TrimSpace(stdout) != "test: add new file" {
		t.Errorf("commit message not found in log: %s", strings.TrimSpace(stdout))
	}
}

// Test 2: CommitChanges returns error if not a git repository
func TestCommitChangesNotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file but no git repo
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0o644)

	err := CommitChanges(tmpDir, "test message")

	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

// Test 3: CommitChanges stages and commits modified files
func TestCommitChangesModifiedFiles(t *testing.T) {
	tmpDir := setupGitRepoWithFiles(t)

	// Modify existing file
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("modified content"), 0o644)

	err := CommitChanges(tmpDir, "modify: test.txt")
	if err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Verify commit was made
	stdout, _, err := ExecGit(tmpDir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("failed to check git log: %v", err)
	}

	if strings.TrimSpace(stdout) != "modify: test.txt" {
		t.Errorf("commit message incorrect: %s", strings.TrimSpace(stdout))
	}
}

// Test 4: CommitChanges with empty message still commits
func TestCommitChangesEmptyMessage(t *testing.T) {
	tmpDir := setupGitRepoWithFiles(t)

	// Create file to commit
	newFile := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(newFile, []byte("content"), 0o644)

	err := CommitChanges(tmpDir, "")
	if err != nil {
		t.Fatalf("CommitChanges with empty message failed: %v", err)
	}

	// Verify a commit was made (even with empty message)
	stdout, _, err := ExecGit(tmpDir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("failed to check git log: %v", err)
	}

	// Git allows empty messages (they become empty commits)
	// Just verify the commit exists
	if stdout == "" {
		t.Error("expected a commit to exist")
	}
}

// Test 5: CommitChanges with no changes still commits
func TestCommitChangesNoChanges(t *testing.T) {
	tmpDir := setupTestRepo(t)

	// No new files, no modifications
	// Try to commit anyway
	err := CommitChanges(tmpDir, "empty commit")
	if err != nil {
		t.Fatalf("CommitChanges with no changes failed: %v", err)
	}

	// Should create a commit even with no changes
	stdout, _, err := ExecGit(tmpDir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("failed to check git log: %v", err)
	}

	if strings.TrimSpace(stdout) != "empty commit" {
		t.Errorf("commit message incorrect: %s", strings.TrimSpace(stdout))
	}
}

// ============================================================================
// GetCommitMessage Tests
// ============================================================================

// Test 6: GetCommitMessage formats "add" operation
func TestGetCommitMessageAdd(t *testing.T) {
	msg := GetCommitMessage("add", "Buy groceries")

	expected := "add: Buy groceries"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

// Test 7: GetCommitMessage formats "update" operation
func TestGetCommitMessageUpdate(t *testing.T) {
	msg := GetCommitMessage("update", "Review OSCP notes")

	expected := "update: Review OSCP notes"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

// Test 8: GetCommitMessage formats "delete" operation
func TestGetCommitMessageDelete(t *testing.T) {
	msg := GetCommitMessage("delete", "Old task")

	expected := "delete: Old task"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

// Test 9: GetCommitMessage formats "complete" operation
func TestGetCommitMessageComplete(t *testing.T) {
	msg := GetCommitMessage("complete", "Finish lab 5")

	expected := "complete: Finish lab 5"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

// Test 10: GetCommitMessage returns non-empty string
func TestGetCommitMessageNonEmpty(t *testing.T) {
	msg := GetCommitMessage("add", "Task")

	if msg == "" {
		t.Error("GetCommitMessage returned empty string")
	}
}

// Test 11: GetCommitMessage with long title
func TestGetCommitMessageLongTitle(t *testing.T) {
	longTitle := "This is a very long task title that might be truncated or left as-is"
	msg := GetCommitMessage("add", longTitle)

	if msg == "" {
		t.Error("GetCommitMessage returned empty string for long title")
	}

	// Should contain the operation and at least part of the title
	if !strings.Contains(msg, "add") {
		t.Error("message should contain operation")
	}
}

// Test 12: GetCommitMessage with special characters
func TestGetCommitMessageSpecialChars(t *testing.T) {
	msg := GetCommitMessage("add", "Task with 'quotes' and \"double\"")

	if msg == "" {
		t.Error("GetCommitMessage returned empty string")
	}

	// Should contain the operation
	if !strings.Contains(msg, "add") {
		t.Error("message should contain operation")
	}
}

// ============================================================================
// TryCommit Tests
// ============================================================================

// Test 13: TryCommit succeeds in a git repository
func TestTryCommitSuccess(t *testing.T) {
	tmpDir := setupGitRepoWithFiles(t)

	// Create a new file
	newFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(newFile, []byte("content"), 0o644)

	err := TryCommit(tmpDir, "add", "test task")
	if err != nil {
		t.Fatalf("TryCommit failed: %v", err)
	}

	// Verify commit was made
	stdout, _, err := ExecGit(tmpDir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("failed to check git log: %v", err)
	}

	if !strings.Contains(strings.TrimSpace(stdout), "add") {
		t.Error("commit message should contain operation")
	}
}

// Test 14: TryCommit returns nil if not a git repository
func TestTryCommitNotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file but no git repo
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0o644)

	// TryCommit should NOT return error for non-git directory
	err := TryCommit(tmpDir, "add", "task")
	if err != nil {
		t.Errorf("TryCommit should return nil for non-git dir, got: %v", err)
	}
}

// Test 15: TryCommit uses GetCommitMessage
func TestTryCommitUsesGetCommitMessage(t *testing.T) {
	tmpDir := setupGitRepoWithFiles(t)

	// Create a new file
	newFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(newFile, []byte("content"), 0o644)

	err := TryCommit(tmpDir, "complete", "my task")
	if err != nil {
		t.Fatalf("TryCommit failed: %v", err)
	}

	// Verify message format from GetCommitMessage was used
	stdout, _, err := ExecGit(tmpDir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("failed to check git log: %v", err)
	}

	msg := strings.TrimSpace(stdout)
	expectedFormat := "complete: my task"
	if msg != expectedFormat {
		t.Errorf("expected message format %q, got %q", expectedFormat, msg)
	}
}

// Test 16: TryCommit with different operations
func TestTryCommitDifferentOperations(t *testing.T) {
	operations := []string{"add", "update", "delete", "complete"}

	for _, op := range operations {
		tmpDir := setupGitRepoWithFiles(t)

		// Create a file to commit
		newFile := filepath.Join(tmpDir, "file.txt")
		os.WriteFile(newFile, []byte("content"), 0o644)

		err := TryCommit(tmpDir, op, "test")
		if err != nil {
			t.Errorf("TryCommit with operation %q failed: %v", op, err)
		}

		// Verify commit was made with correct operation
		stdout, _, err := ExecGit(tmpDir, "log", "-1", "--format=%s")
		if err != nil {
			t.Fatalf("failed to check git log: %v", err)
		}

		if !strings.Contains(strings.TrimSpace(stdout), op) {
			t.Errorf("commit message should contain operation %q", op)
		}
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

// Test 17: Multiple commits in sequence
func TestMultipleCommits(t *testing.T) {
	tmpDir := setupGitRepoWithFiles(t)

	// First commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	os.WriteFile(file1, []byte("content1"), 0o644)
	err := TryCommit(tmpDir, "add", "file 1")
	if err != nil {
		t.Fatalf("first commit failed: %v", err)
	}

	// Second commit
	file2 := filepath.Join(tmpDir, "file2.txt")
	os.WriteFile(file2, []byte("content2"), 0o644)
	err = TryCommit(tmpDir, "add", "file 2")
	if err != nil {
		t.Fatalf("second commit failed: %v", err)
	}

	// Verify both commits exist
	stdout, _, err := ExecGit(tmpDir, "log", "--format=%s")
	if err != nil {
		t.Fatalf("failed to check git log: %v", err)
	}

	log := strings.TrimSpace(stdout)
	lines := strings.Split(log, "\n")

	if len(lines) < 2 {
		t.Errorf("expected at least 2 commits, got %d", len(lines))
	}
}

// Test 18: Commit with very long task title
func TestTryCommitLongTitle(t *testing.T) {
	tmpDir := setupGitRepoWithFiles(t)

	newFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(newFile, []byte("content"), 0o644)

	longTitle := "This is a very long task title that contains lots of information and might be problematic in commit messages"
	err := TryCommit(tmpDir, "add", longTitle)
	if err != nil {
		t.Fatalf("TryCommit with long title failed: %v", err)
	}

	// Verify commit was made
	stdout, _, err := ExecGit(tmpDir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatalf("failed to check git log: %v", err)
	}

	if strings.TrimSpace(stdout) == "" {
		t.Error("commit should have a message")
	}
}
