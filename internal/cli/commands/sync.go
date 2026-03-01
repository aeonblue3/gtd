package commands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gtd/internal/git"
	"gtd/internal/models"
	"gtd/internal/sync"
)

// Sync initializes, runs one sync, or starts daemon sync.
func Sync(cfg *models.Config, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	initRepo := fs.Bool("init", false, "Initialize git repo in data path")
	daemon := fs.Bool("daemon", false, "Run background sync loop")
	interval := fs.Int("interval", cfg.SyncIntervalMinutes, "Sync interval in minutes")
	remote := fs.String("remote", cfg.GitRemote, "Remote name")
	branch := fs.String("branch", cfg.GitBranch, "Branch name")
	resolve := fs.String("resolve", "", "Resolve conflicts with strategy: ours|theirs")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *initRepo {
		if err := git.InitRepo(cfg.DataPath); err != nil {
			return err
		}
		fmt.Println("✓ Initialized git repository")
		if !*daemon && len(args) == 1 {
			return nil
		}
	}

	if !git.IsGitRepo(cfg.DataPath) {
		fmt.Println("Skipping sync: data path is not a git repository")
		fmt.Println("Run: gtd sync --init")
		return nil
	}
	if *resolve != "" {
		if err := git.ResolveConflicts(cfg.DataPath, *resolve); err != nil {
			return err
		}
		fmt.Printf("✓ Conflicts resolved using %q strategy\n", *resolve)
		return nil
	}

	if !git.HasRemote(cfg.DataPath, *remote) {
		fmt.Printf("Skipping sync: git remote %q not configured\n", *remote)
		return nil
	}

	svc := &sync.Service{
		DataPath: cfg.DataPath,
		Remote:   *remote,
		Branch:   *branch,
		Interval: time.Duration(*interval) * time.Minute,
	}

	if *daemon {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		fmt.Printf("Starting sync daemon (every %d minutes). Press Ctrl+C to stop.\n", *interval)
		return svc.Run(ctx, os.Stdout)
	}

	if err := svc.SyncOnce(); err != nil {
		return err
	}
	fmt.Println("✓ Sync completed")
	return nil
}
