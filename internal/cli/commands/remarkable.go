package commands

import (
	"flag"
	"fmt"

	"gtd/internal/integrations/remarkable"
	"gtd/internal/models"
	"gtd/internal/storage"
)

// Remarkable syncs tasks with markdown-based ReMarkable workflow.
func Remarkable(store storage.Backend, cfg *models.Config, args []string) error {
	fs := flag.NewFlagSet("remarkable", flag.ContinueOnError)
	export := fs.Bool("export", false, "Export tasks to markdown checklist")
	importFile := fs.String("import", "", "Import completions from markdown checklist")
	path := fs.String("path", remarkable.DefaultExportPath(cfg.DataPath), "Markdown file path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *export {
		if err := remarkable.Export(store, *path); err != nil {
			return err
		}
		fmt.Printf("✓ Exported tasks to %s\n", *path)
		return nil
	}
	if *importFile != "" {
		count, err := remarkable.ImportCompletions(store, *importFile)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Imported %d completions\n", count)
		return nil
	}
	return fmt.Errorf("use --export or --import <file>")
}
