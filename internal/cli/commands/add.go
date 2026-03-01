package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gtd/internal/git"
	"gtd/internal/models"
	"gtd/internal/parser"
	"gtd/internal/storage"
)

// Add creates a new task
func Add(store storage.Backend, cfg *models.Config, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)

	contextStr := fs.String("context", cfg.DefaultContext, "Context(s) (comma-separated)")
	c := fs.String("c", cfg.DefaultContext, "Context (shorthand)")
	priorityStr := fs.String("priority", "none", "Priority (none, low, medium, high)")
	p := fs.String("p", "none", "Priority (shorthand)")
	recurrenceStr := fs.String("recurrence", "none", "Recurrence (none, daily, weekly, monthly)")
	dueStr := fs.String("due", "", "Due date")
	d := fs.String("d", "", "Due date (shorthand)")
	tagsStr := fs.String("tags", "", "Tags (comma-separated)")
	t := fs.String("t", "", "Tags (shorthand)")
	description := fs.String("description", "", "Task description")

	remaining := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		titlePos := 0
		for i, arg := range args {
			if strings.HasPrefix(arg, "-") {
				titlePos = i
				break
			}
			titlePos = i + 1
		}
		remaining = args[:titlePos]
		if titlePos < len(args) {
			if err := fs.Parse(args[titlePos:]); err != nil {
				return err
			}
		}
	} else {
		if err := fs.Parse(args); err != nil {
			return err
		}
		remaining = fs.Args()
	}

	if len(remaining) == 0 {
		return fmt.Errorf("task title is required; use: gtd add \"title\" [flags]")
	}

	title := strings.Join(remaining, " ")

	// Handle shorthand flags
	if *c != cfg.DefaultContext {
		contextStr = c
	}
	if *p != "none" {
		priorityStr = p
	}
	if *d != "" {
		dueStr = d
	}
	if *t != "" {
		tagsStr = t
	}

	// Create task
	task := models.NewTask(title)
	task.Description = *description

	// Parse and set contexts
	if *contextStr != "" {
		contexts := strings.Split(*contextStr, ",")
		for i, ctx := range contexts {
			contexts[i] = strings.TrimSpace(ctx)
		}
		task.Contexts = contexts
	}

	// Set priority
	priority := models.Priority(*priorityStr)
	switch priority {
	case models.PriorityNone, models.PriorityLow, models.PriorityMedium, models.PriorityHigh:
		task.Priority = priority
	default:
		return fmt.Errorf("invalid priority: %s", *priorityStr)
	}

	// Parse and set due date
	if *dueStr != "" {
		dueDate, err := parser.ParseDueDate(*dueStr)
		if err != nil {
			return fmt.Errorf("invalid due date: %w", err)
		}
		task.DueDate = dueDate
	}

	recurrence := models.Recurrence(*recurrenceStr)
	switch recurrence {
	case models.RecurrenceNone, models.RecurrenceDaily, models.RecurrenceWeekly, models.RecurrenceMonthly:
		task.Recurrence = recurrence
	default:
		return fmt.Errorf("invalid recurrence: %s", *recurrenceStr)
	}

	// Parse and set tags
	if *tagsStr != "" {
		tags := strings.Split(*tagsStr, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		task.Tags = tags
	}

	// Save task
	if err := store.AddTask(task); err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}
	if cfg.AutoCommit {
		if err := git.TryCommit(cfg.DataPath, "add", task.Title); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: task added but auto-commit failed: %v\n", err)
		}
	}
	if cfg.AutoSync {
		if err := git.SyncWithRemote(cfg.DataPath, cfg.GitRemote, cfg.GitBranch); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-sync failed: %v\n", err)
		}
	}

	fmt.Printf("✓ Added task: %s\n", task.ID)
	return nil
}
