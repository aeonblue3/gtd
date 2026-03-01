package git

import "testing"

func TestResolveConflictsInvalidStrategy(t *testing.T) {
	if err := ResolveConflicts(t.TempDir(), "invalid"); err == nil {
		t.Fatal("expected invalid strategy error")
	}
}

func TestDetectConflictsNoRepo(t *testing.T) {
	_, err := DetectConflicts(t.TempDir())
	if err == nil {
		t.Fatal("expected git error in non-repo")
	}
}
