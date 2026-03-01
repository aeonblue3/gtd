package remarkable

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gtd/internal/models"
	"gtd/internal/storage"
)

// Export writes tasks into a markdown checklist for tablet workflows.
func Export(store storage.Backend, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, _ = fmt.Fprintln(f, "# GTD Export")
	_, _ = fmt.Fprintln(f, "")
	for _, task := range store.GetAllTasks() {
		box := " "
		if task.Status == models.StatusDone {
			box = "x"
		}
		_, _ = fmt.Fprintf(f, "- [%s] %s | id=%s\n", box, task.Title, task.ID)
	}
	return nil
}

// ImportCompletions reads a checklist markdown file and marks completed tasks done.
func ImportCompletions(store storage.Backend, inputPath string) (int, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	done := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "- [x]") {
			continue
		}
		id := parseID(line)
		if id == "" {
			continue
		}
		task, err := store.GetTask(id)
		if err != nil {
			continue
		}
		now := time.Now()
		task.Status = models.StatusDone
		task.CompletedAt = &now
		if err := store.UpdateTask(task); err == nil {
			done++
		}
	}
	if err := scanner.Err(); err != nil {
		return done, err
	}
	return done, nil
}

func parseID(line string) string {
	idx := strings.Index(line, "id=")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(line[idx+3:])
}

// DefaultExportPath returns a default file path used for tablet export.
func DefaultExportPath(dataPath string) string {
	return filepath.Join(dataPath, "remarkable_tasks.md")
}
