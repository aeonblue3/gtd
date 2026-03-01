package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ExecGit executes a git command in the data directory
// Returns stdout, stderr, and error
//
// Example:
//
//	stdout, stderr, err := ExecGit("/home/user/.gtd", "status")
func ExecGit(dataPath string, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("git", args...)
	cmd.Dir = dataPath

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())
	if err != nil {
		return out, errOut, fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}

	return out, errOut, nil
}

// IsGitRepo checks if dataPath is a git repository
// Returns true if it's a git repo, false otherwise
// Does not return errors - just returns a boolean
func IsGitRepo(dataPath string) bool {
	_, _, err := ExecGit(dataPath, "rev-parse", "--git-dir")
	return err == nil
}

// GetCurrentBranch returns the current branch name
// Returns the branch name or an error if not a git repo
func GetCurrentBranch(dataPath string) (string, error) {
	branch, _, err := ExecGit(dataPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(branch), nil
}

// GetLastCommitTime returns Unix timestamp of last commit
// Returns 0 if there are no commits yet (not an error)
func GetLastCommitTime(dataPath string) (int64, error) {
	var ts int64

	ct, stderr, err := ExecGit(dataPath, "log", "-1", "--format=%ct")
	if err != nil {
		if strings.Contains(stderr, "does not have any commits yet") {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get last commit time: %w", err)
	}

	ct = strings.TrimSpace(ct)
	if ct == "" {
		return 0, nil
	}

	ts, err = strconv.ParseInt(ct, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return ts, nil
}

// CommitChanges stages all changes and creates a commit.
func CommitChanges(dataPath string, message string) error {
	if !IsGitRepo(dataPath) {
		return fmt.Errorf("not a git repository: %s", dataPath)
	}
	if strings.TrimSpace(message) == "" {
		message = "update: task"
	}

	if _, stderr, err := ExecGit(dataPath, "add", "-A"); err != nil {
		return fmt.Errorf("failed to stage changes: %v (%s)", err, stderr)
	}

	if _, stderr, err := ExecGit(dataPath, "commit", "--allow-empty", "--allow-empty-message", "-m", message); err != nil {
		return fmt.Errorf("failed to commit changes: %v (%s)", err, stderr)
	}

	return nil
}

// GetCommitMessage creates a consistent commit message for task operations.
func GetCommitMessage(operation, taskTitle string) string {
	operation = strings.TrimSpace(operation)
	taskTitle = strings.TrimSpace(taskTitle)
	if operation == "" {
		operation = "update"
	}
	if taskTitle == "" {
		taskTitle = "task"
	}
	return fmt.Sprintf("%s: %s", operation, taskTitle)
}

// TryCommit commits changes only when dataPath is a git repository.
func TryCommit(dataPath, operation, taskTitle string) error {
	if !IsGitRepo(dataPath) {
		return nil
	}
	return CommitChanges(dataPath, GetCommitMessage(operation, taskTitle))
}

// InitRepo initializes a git repository in dataPath when absent.
func InitRepo(dataPath string) error {
	if IsGitRepo(dataPath) {
		return nil
	}
	if _, stderr, err := ExecGit(dataPath, "init"); err != nil {
		return fmt.Errorf("failed to initialize git repository: %v (%s)", err, stderr)
	}
	return nil
}

// HasRemote returns true when the git remote exists.
func HasRemote(dataPath, remote string) bool {
	if strings.TrimSpace(remote) == "" {
		return false
	}
	_, _, err := ExecGit(dataPath, "remote", "get-url", remote)
	return err == nil
}

// SyncWithRemote pulls and pushes against a configured remote/branch.
func SyncWithRemote(dataPath, remote, branch string) error {
	if !IsGitRepo(dataPath) {
		return nil
	}
	if !HasRemote(dataPath, remote) {
		return nil
	}
	if strings.TrimSpace(branch) == "" {
		branch = "main"
	}

	if _, stderr, err := ExecGit(dataPath, "fetch", remote); err != nil {
		return fmt.Errorf("failed to fetch from %s: %v (%s)", remote, err, stderr)
	}
	if _, stderr, err := ExecGit(dataPath, "pull", "--rebase", remote, branch); err != nil {
		return fmt.Errorf("failed to pull %s/%s: %v (%s)", remote, branch, err, stderr)
	}
	if _, stderr, err := ExecGit(dataPath, "push", remote, "HEAD:"+branch); err != nil {
		return fmt.Errorf("failed to push to %s/%s: %v (%s)", remote, branch, err, stderr)
	}
	return nil
}

// DetectConflicts returns files currently in merge conflict state.
func DetectConflicts(dataPath string) ([]string, error) {
	stdout, _, err := ExecGit(dataPath, "ls-files", "-u")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(stdout) == "" {
		return []string{}, nil
	}
	uniq := map[string]bool{}
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			uniq[fields[3]] = true
		}
	}
	out := make([]string, 0, len(uniq))
	for f := range uniq {
		out = append(out, f)
	}
	return out, nil
}

// ResolveConflicts resolves tracked conflicts using ours/theirs strategy.
func ResolveConflicts(dataPath, strategy string) error {
	if strategy != "ours" && strategy != "theirs" {
		return fmt.Errorf("invalid conflict strategy: %s", strategy)
	}
	files, err := DetectConflicts(dataPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		file = filepath.Clean(file)
		if _, stderr, err := ExecGit(dataPath, "checkout", "--"+strategy, "--", file); err != nil {
			return fmt.Errorf("failed to checkout %s for %s: %v (%s)", strategy, file, err, stderr)
		}
		if _, stderr, err := ExecGit(dataPath, "add", file); err != nil {
			return fmt.Errorf("failed to add resolved file %s: %v (%s)", file, err, stderr)
		}
	}
	return nil
}
