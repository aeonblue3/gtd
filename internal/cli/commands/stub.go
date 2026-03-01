package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gtd/internal/git"
	"gtd/internal/models"
	"gtd/internal/storage"
)

// View displays detailed task information
func View(store storage.Backend, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID required")
	}

	task, err := store.GetTask(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("ID:          %s\n", task.ID)
	fmt.Printf("Title:       %s\n", task.Title)
	fmt.Printf("Status:      %s\n", task.Status)
	fmt.Printf("Priority:    %s\n", task.Priority)
	fmt.Printf("Contexts:    %v\n", task.Contexts)
	fmt.Printf("Created:     %s\n", task.CreatedAt.Format(time.RFC3339))
	if task.DueDate != nil {
		fmt.Printf("Due:         %s\n", task.DueDate.Format("2006-01-02"))
		if task.IsOverdue() {
			fmt.Printf("Status:      OVERDUE\n")
		}
	}
	if task.Description != "" {
		fmt.Printf("Description: %s\n", task.Description)
	}
	if len(task.Tags) > 0 {
		fmt.Printf("Tags:        %v\n", task.Tags)
	}
	if len(task.LinkedTasks) > 0 {
		fmt.Printf("Depends On:  %v\n", task.LinkedTasks)
	}
	if len(task.Subtasks) > 0 {
		fmt.Printf("Subtasks:\n")
		for i, sub := range task.Subtasks {
			state := " "
			if sub.CompletedAt != nil {
				state = "x"
			}
			fmt.Printf("  [%s] %d. %s\n", state, i+1, sub.Title)
		}
	}
	if task.Notes != "" {
		fmt.Printf("Notes:       %s\n", task.Notes)
	}

	return nil
}

// Update modifies an existing task
func Update(store storage.Backend, cfg *models.Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID required")
	}

	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	title := fs.String("title", "", "New title")
	status := fs.String("status", "", "New status")
	priority := fs.String("priority", "", "New priority")

	fs.Parse(args[1:])

	task, err := store.GetTask(args[0])
	if err != nil {
		return err
	}

	if *title != "" {
		task.Title = *title
	}
	if *status != "" {
		task.Status = models.Status(*status)
	}
	if *priority != "" {
		task.Priority = models.Priority(*priority)
	}

	if err := store.UpdateTask(task); err != nil {
		return err
	}
	if cfg.AutoCommit {
		if err := git.TryCommit(cfg.DataPath, "update", task.Title); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: task updated but auto-commit failed: %v\n", err)
		}
	}
	if cfg.AutoSync {
		if err := git.SyncWithRemote(cfg.DataPath, cfg.GitRemote, cfg.GitBranch); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-sync failed: %v\n", err)
		}
	}
	return nil
}

// Done marks a task as complete
func Done(store storage.Backend, cfg *models.Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID required")
	}

	task, err := store.GetTask(args[0])
	if err != nil {
		return err
	}

	task.Status = models.StatusDone
	now := time.Now()
	task.CompletedAt = &now

	if err := store.UpdateTask(task); err != nil {
		return err
	}
	if cfg.AutoCommit {
		if err := git.TryCommit(cfg.DataPath, "complete", task.Title); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: task completed but auto-commit failed: %v\n", err)
		}
	}
	if cfg.AutoSync {
		if err := git.SyncWithRemote(cfg.DataPath, cfg.GitRemote, cfg.GitBranch); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-sync failed: %v\n", err)
		}
	}

	fmt.Printf("✓ Task marked as done: %s\n", task.Title)

	if next := task.NextRecurringInstance(); next != nil {
		if err := store.AddTask(next); err != nil {
			return fmt.Errorf("task completed, but failed to create next recurring instance: %w", err)
		}
		fmt.Printf("↺ Created next recurring task: %s\n", next.Title)
	}

	return nil
}

// Delete removes a task
func Delete(store storage.Backend, cfg *models.Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID required")
	}

	task, err := store.GetTask(args[0])
	if err != nil {
		return err
	}

	if err := store.DeleteTask(args[0]); err != nil {
		return err
	}
	if cfg.AutoCommit {
		if err := git.TryCommit(cfg.DataPath, "delete", task.Title); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: task deleted but auto-commit failed: %v\n", err)
		}
	}
	if cfg.AutoSync {
		if err := git.SyncWithRemote(cfg.DataPath, cfg.GitRemote, cfg.GitBranch); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-sync failed: %v\n", err)
		}
	}

	fmt.Printf("✓ Task deleted: %s\n", task.Title)
	return nil
}

// Inbox shows tasks in inbox status
func Inbox(store storage.Backend, args []string) error {
	tasks := store.GetTasksByStatus(models.StatusInbox)

	if len(tasks) == 0 {
		fmt.Println("Inbox is empty!")
		return nil
	}

	fmt.Printf("Inbox (%d items):\n", len(tasks))
	for _, task := range tasks {
		fmt.Printf("  %s  %s\n", task.ID[:8], task.Title)
	}

	return nil
}

// Today shows today's tasks
func Today(store storage.Backend, args []string) error {
	tasks := store.GetTasksByStatus(models.StatusActionable)
	now := time.Now()
	var filtered []*models.Task
	for _, task := range tasks {
		if task.DueDate == nil {
			continue
		}
		y1, m1, d1 := now.Date()
		y2, m2, d2 := task.DueDate.Date()
		if y1 == y2 && m1 == m2 && d1 == d2 {
			filtered = append(filtered, task)
		}
	}
	if len(filtered) == 0 {
		filtered = tasks
	}

	if len(filtered) == 0 {
		fmt.Println("No actionable tasks for today")
		return nil
	}

	fmt.Printf("Today's Tasks (%d items):\n", len(filtered))
	for _, task := range filtered {
		fmt.Printf("  %s  %s\n", task.ID[:8], task.Title)
	}

	return nil
}

// Review shows a weekly review
func Review(store storage.Backend, args []string) error {
	fmt.Println("=== WEEKLY REVIEW ===")
	fmt.Println()

	actionable := store.GetTasksByStatus(models.StatusActionable)
	waiting := store.GetTasksByStatus(models.StatusWaiting)
	someday := store.GetTasksByStatus(models.StatusSomeday)
	done := store.GetTasksByStatus(models.StatusDone)

	fmt.Printf("Actionable:  %d\n", len(actionable))
	fmt.Printf("Waiting:     %d\n", len(waiting))
	fmt.Printf("Someday:     %d\n", len(someday))
	fmt.Printf("Done:        %d\n", len(done))
	fmt.Printf("Inbox:       %d\n", len(store.GetTasksByStatus(models.StatusInbox)))

	weekAgo := time.Now().AddDate(0, 0, -7)
	completedThisWeek := 0
	for _, task := range done {
		if task.CompletedAt != nil && task.CompletedAt.After(weekAgo) {
			completedThisWeek++
		}
	}
	fmt.Printf("Completed this week: %d\n", completedThisWeek)

	overdueCount := 0
	for _, task := range store.GetAllTasks() {
		if task.IsOverdue() {
			overdueCount++
		}
	}
	fmt.Printf("Overdue:     %d\n", overdueCount)

	return nil
}

// Search finds tasks matching a query
func Search(store storage.Backend, args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	exact := fs.Bool("exact", false, "Exact phrase search")
	regexPattern := fs.String("regex", "", "Regex pattern search")
	jsonOut := fs.Bool("json", false, "Print search results as JSON")
	query, remainder := SplitLeadingPositional(args)
	if err := fs.Parse(remainder); err != nil {
		return err
	}
	if query == "" {
		remaining := fs.Args()
		if len(remaining) > 0 {
			query = remaining[0]
		}
	}

	if query == "" && *regexPattern == "" {
		return fmt.Errorf("search query required")
	}
	tasks := store.GetAllTasks()

	var results []*models.Task
	var rx *regexp.Regexp
	var err error
	if *regexPattern != "" {
		rx, err = regexp.Compile(*regexPattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	for _, task := range tasks {
		titleMatch := matchesQuery(task.Title, query, *exact, rx)
		descMatch := matchesQuery(task.Description, query, *exact, rx)
		notesMatch := matchesQuery(task.Notes, query, *exact, rx)
		if titleMatch || descMatch || notesMatch {
			results = append(results, task)
		}
	}

	if len(results) == 0 {
		if *regexPattern != "" {
			fmt.Printf("No tasks found matching regex: %s\n", *regexPattern)
		} else {
			fmt.Printf("No tasks found matching: %s\n", query)
		}
		return nil
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	fmt.Printf("Found %d tasks:\n", len(results))
	for _, task := range results {
		fmt.Printf("  %s  %s\n", task.ID[:8], task.Title)
	}

	return nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func matchesQuery(text, query string, exact bool, rx *regexp.Regexp) bool {
	if rx != nil {
		return rx.MatchString(text)
	}
	if exact {
		return strings.EqualFold(strings.TrimSpace(text), strings.TrimSpace(query))
	}
	return contains(text, query)
}
