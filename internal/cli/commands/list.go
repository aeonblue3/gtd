package commands

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"gtd/internal/models"
	"gtd/internal/storage"
)

// List displays tasks with filtering options
func List(store storage.Backend, cfg *models.Config, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)

	contextStr := fs.String("context", "", "Filter by context(s) (comma-separated)")
	c := fs.String("c", "", "Filter by context (shorthand)")
	statusStr := fs.String("status", "", "Filter by status")
	priorityStr := fs.String("priority", "", "Filter by priority")
	overdueOnly := fs.Bool("overdue", false, "Only overdue tasks")
	dueToday := fs.Bool("due-today", false, "Only tasks due today")
	upcomingDays := fs.Int("upcoming", 0, "Tasks due within N days")
	format := fs.String("format", "table", "Output format (table, json, csv)")
	f := fs.String("f", "table", "Output format (shorthand)")

	fs.Parse(args)

	// Handle shorthand
	if *c != "" {
		contextStr = c
	}
	if *f != "table" {
		format = f
	}

	// Get all tasks
	tasks := store.GetAllTasks()

	// Filter by context
	if *contextStr != "" {
		contexts := strings.Split(*contextStr, ",")
		for i, ctx := range contexts {
			contexts[i] = strings.TrimSpace(ctx)
		}
		tasks = filterByContexts(tasks, contexts)
	}

	// Filter by status
	if *statusStr != "" {
		status := models.Status(*statusStr)
		tasks = filterByStatus(tasks, status)
	}

	// Filter by priority
	if *priorityStr != "" {
		priority := models.Priority(*priorityStr)
		tasks = filterByPriority(tasks, priority)
	}
	if *overdueOnly {
		tasks = filterOverdue(tasks)
	}
	if *dueToday {
		tasks = filterDueToday(tasks)
	}
	if *upcomingDays > 0 {
		tasks = filterUpcoming(tasks, *upcomingDays)
	}

	// Sort by due date first (soonest), then context, then priority.
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].DueDate != nil && tasks[j].DueDate != nil && !tasks[i].DueDate.Equal(*tasks[j].DueDate) {
			return tasks[i].DueDate.Before(*tasks[j].DueDate)
		}
		if tasks[i].DueDate != nil && tasks[j].DueDate == nil {
			return true
		}
		if tasks[i].DueDate == nil && tasks[j].DueDate != nil {
			return false
		}

		if len(tasks[i].Contexts) > 0 && len(tasks[j].Contexts) > 0 {
			if tasks[i].Contexts[0] != tasks[j].Contexts[0] {
				return tasks[i].Contexts[0] < tasks[j].Contexts[0]
			}
		}
		// Then by priority (high > medium > low > none)
		priorityOrder := map[models.Priority]int{
			models.PriorityHigh:   0,
			models.PriorityMedium: 1,
			models.PriorityLow:    2,
			models.PriorityNone:   3,
		}
		return priorityOrder[tasks[i].Priority] < priorityOrder[tasks[j].Priority]
	})

	if len(tasks) == 0 {
		fmt.Println("No tasks found")
		return nil
	}

	// Format output
	switch *format {
	case "table":
		return printTableFormat(tasks)
	case "json":
		return printJSONFormat(tasks)
	case "csv":
		return printCSVFormat(tasks)
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}
}

// filterByContexts filters tasks by context
func filterByContexts(tasks []*models.Task, contexts []string) []*models.Task {
	contextMap := make(map[string]bool)
	for _, c := range contexts {
		contextMap[c] = true
	}

	var result []*models.Task
	for _, task := range tasks {
		for _, ctx := range task.Contexts {
			if contextMap[ctx] {
				result = append(result, task)
				break
			}
		}
	}
	return result
}

// filterByStatus filters tasks by status
func filterByStatus(tasks []*models.Task, status models.Status) []*models.Task {
	var result []*models.Task
	for _, task := range tasks {
		if task.Status == status {
			result = append(result, task)
		}
	}
	return result
}

// filterByPriority filters tasks by priority
func filterByPriority(tasks []*models.Task, priority models.Priority) []*models.Task {
	var result []*models.Task
	for _, task := range tasks {
		if task.Priority == priority {
			result = append(result, task)
		}
	}
	return result
}

func filterOverdue(tasks []*models.Task) []*models.Task {
	var result []*models.Task
	for _, task := range tasks {
		if task.IsOverdue() {
			result = append(result, task)
		}
	}
	return result
}

func filterDueToday(tasks []*models.Task) []*models.Task {
	var result []*models.Task
	now := time.Now()
	y, m, d := now.Date()
	for _, task := range tasks {
		if task.DueDate == nil {
			continue
		}
		dy, dm, dd := task.DueDate.Date()
		if y == dy && m == dm && d == dd {
			result = append(result, task)
		}
	}
	return result
}

func filterUpcoming(tasks []*models.Task, days int) []*models.Task {
	var result []*models.Task
	now := time.Now()
	limit := now.Add(time.Duration(days) * 24 * time.Hour)
	for _, task := range tasks {
		if task.DueDate == nil {
			continue
		}
		if task.DueDate.After(now) && (task.DueDate.Before(limit) || task.DueDate.Equal(limit)) {
			result = append(result, task)
		}
	}
	return result
}

// printTableFormat prints tasks in table format
func printTableFormat(tasks []*models.Task) error {
	// Simple table format
	fmt.Printf("%-8s  %-10s  %-10s  %-8s  %-10s  %s\n", "ID", "Status", "Priority", "Context", "Due", "Title")
	fmt.Println(strings.Repeat("-", 96))

	for _, task := range tasks {
		id := task.ID[:8]
		status := string(task.Status)
		priority := string(task.Priority)
		context := ""
		if len(task.Contexts) > 0 {
			context = task.Contexts[0]
		}
		due := ""
		if task.DueDate != nil {
			due = task.DueDate.Format("2006-01-02")
		}
		title := task.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		overdue := ""
		if task.IsOverdue() {
			overdue = " [OVERDUE]"
		}

		fmt.Printf("%-8s  %-10s  %-10s  %-8s  %-10s  %s%s\n", id, status, priority, context, due, title, overdue)
	}

	return nil
}

// printJSONFormat prints tasks in JSON format
func printJSONFormat(tasks []*models.Task) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(tasks)
}

// printCSVFormat prints tasks in CSV format
func printCSVFormat(tasks []*models.Task) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	if err := w.Write([]string{"ID", "Title", "Status", "Priority", "Context", "DueDate", "DaysUntilDue"}); err != nil {
		return err
	}
	for _, task := range tasks {
		context := ""
		if len(task.Contexts) > 0 {
			context = task.Contexts[0]
		}
		dueDate := ""
		if task.DueDate != nil {
			dueDate = task.DueDate.Format("2006-01-02")
		}
		days := ""
		if task.DueDate != nil {
			days = strconv.Itoa(task.DaysUntilDue())
		}
		if err := w.Write([]string{task.ID, task.Title, string(task.Status), string(task.Priority), context, dueDate, days}); err != nil {
			return err
		}
	}
	return nil
}
