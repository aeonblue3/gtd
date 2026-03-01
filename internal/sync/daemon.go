package sync

import (
	"context"
	"fmt"
	"io"
	"time"

	"gtd/internal/git"
)

// Service periodically synchronizes the local GTD repo with remote.
type Service struct {
	DataPath string
	Remote   string
	Branch   string
	Interval time.Duration
}

// SyncOnce performs a single synchronization attempt.
func (s *Service) SyncOnce() error {
	return git.SyncWithRemote(s.DataPath, s.Remote, s.Branch)
}

// Run starts a ticker-based sync loop until the context is canceled.
func (s *Service) Run(ctx context.Context, out io.Writer) error {
	if s.Interval <= 0 {
		s.Interval = 30 * time.Minute
	}

	if err := s.SyncOnce(); err != nil && out != nil {
		fmt.Fprintf(out, "Initial sync failed: %v\n", err)
	}

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.SyncOnce(); err != nil && out != nil {
				fmt.Fprintf(out, "Sync failed: %v\n", err)
			} else if out != nil {
				fmt.Fprintln(out, "Sync completed")
			}
		}
	}
}
