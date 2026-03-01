package commands

import (
	"flag"
	"fmt"
	"os"

	"gtd/internal/git"
	"gtd/internal/models"
	"gtd/internal/storage"
)

// Subtask manages task checklists.
func Subtask(store storage.Backend, cfg *models.Config, args []string) error {
	fs := flag.NewFlagSet("subtask", flag.ContinueOnError)
	add := fs.String("add", "", "Subtask title to add")
	done := fs.Int("done", 0, "Subtask number to mark done (1-based)")
	taskID, remainder := SplitLeadingPositional(args)
	if err := fs.Parse(remainder); err != nil {
		return err
	}
	if taskID == "" {
		if len(fs.Args()) > 0 {
			taskID = fs.Args()[0]
		} else {
			return fmt.Errorf("task ID required")
		}
	}

	if *add != "" {
		if err := store.AddSubtask(taskID, *add); err != nil {
			return err
		}
		fmt.Println("✓ Subtask added")
		if cfg.AutoCommit {
			if err := git.TryCommit(cfg.DataPath, "update", "subtask added"); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: auto-commit failed: %v\n", err)
			}
		}
		if cfg.AutoSync {
			if err := git.SyncWithRemote(cfg.DataPath, cfg.GitRemote, cfg.GitBranch); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: auto-sync failed: %v\n", err)
			}
		}
		return nil
	}

	if *done > 0 {
		if err := store.CompleteSubtask(taskID, *done-1); err != nil {
			return err
		}
		fmt.Println("✓ Subtask completed")
		if cfg.AutoCommit {
			if err := git.TryCommit(cfg.DataPath, "complete", "subtask completed"); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: auto-commit failed: %v\n", err)
			}
		}
		if cfg.AutoSync {
			if err := git.SyncWithRemote(cfg.DataPath, cfg.GitRemote, cfg.GitBranch); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: auto-sync failed: %v\n", err)
			}
		}
		return nil
	}

	return fmt.Errorf("no action specified; use --add or --done")
}

// Depends manages task dependencies.
func Depends(store storage.Backend, cfg *models.Config, args []string) error {
	fs := flag.NewFlagSet("depends", flag.ContinueOnError)
	on := fs.String("on", "", "Task ID dependency")
	remove := fs.String("remove", "", "Remove dependency task ID")
	check := fs.Bool("check", false, "Check if task can be started")
	taskID, remainder := SplitLeadingPositional(args)
	if err := fs.Parse(remainder); err != nil {
		return err
	}
	if taskID == "" {
		if len(fs.Args()) > 0 {
			taskID = fs.Args()[0]
		} else {
			return fmt.Errorf("task ID required")
		}
	}

	if *on != "" {
		if err := store.AddDependency(taskID, *on); err != nil {
			return err
		}
		fmt.Println("✓ Dependency added")
		return nil
	}
	if *remove != "" {
		if err := store.RemoveDependency(taskID, *remove); err != nil {
			return err
		}
		fmt.Println("✓ Dependency removed")
		return nil
	}
	if *check {
		ok, err := store.CanStartTask(taskID)
		if err != nil {
			return err
		}
		if ok {
			fmt.Println("✓ All dependencies complete")
		} else {
			fmt.Println("Blocked: incomplete dependencies")
		}
		return nil
	}

	return fmt.Errorf("no action specified; use --on, --remove, or --check")
}
