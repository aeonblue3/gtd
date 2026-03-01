package cli

import (
	"fmt"
	"os"

	"gtd/internal/cli/commands"
	"gtd/internal/config"
	"gtd/internal/git"
	"gtd/internal/storage"
)

// Execute runs the CLI application
func Execute() error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage
	store, err := storage.NewStore(cfg.DataPath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	if cfg.AutoCommit || cfg.AutoSync {
		_ = git.InitRepo(cfg.DataPath)
	}

	// Parse command line arguments
	if len(os.Args) < 2 {
		return printUsage()
	}

	command := os.Args[1]
	args := os.Args[2:]
	if command == "--help" || command == "-h" {
		return printUsage()
	}
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		return printCommandUsage(command)
	}

	// Route to command handler
	switch command {
	case "add":
		return commands.Add(store, cfg, args)
	case "list":
		return commands.List(store, cfg, args)
	case "view":
		return commands.View(store, args)
	case "update":
		return commands.Update(store, cfg, args)
	case "done":
		return commands.Done(store, cfg, args)
	case "delete":
		return commands.Delete(store, cfg, args)
	case "inbox":
		return commands.Inbox(store, args)
	case "today":
		return commands.Today(store, args)
	case "review":
		return commands.Review(store, args)
	case "search":
		return commands.Search(store, args)
	case "sync":
		return commands.Sync(cfg, args)
	case "config":
		return commands.Config(cfg, args)
	case "subtask":
		return commands.Subtask(store, cfg, args)
	case "depends":
		return commands.Depends(store, cfg, args)
	case "serve":
		return commands.Serve(store, args)
	case "server":
		return commands.Server(args)
	case "completion":
		return commands.Completion(args)
	case "remarkable":
		return commands.Remarkable(store, cfg, args)
	case "help":
		if len(args) > 0 {
			return printCommandUsage(args[0])
		}
		return printUsage()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// printUsage displays help text
func printUsage() error {
	usage := `gtd - A minimal GTD CLI system

Usage:
  gtd <command> [options]

Commands:
  add         Add a new task
  list        List tasks
  view        View task details
  update      Update a task
  done        Mark task as complete
  delete      Delete a task
  inbox       Show inbox tasks
  today       Show today's tasks
  review      Weekly review
  search      Search tasks
  sync        Sync local repo with remote
  config      Show or update configuration
  subtask     Manage task subtasks
  depends     Manage task dependencies
  serve       Start optional HTTP API server
  server      Start next-generation API skeleton
  completion  Generate shell completion script
  remarkable  ReMarkable markdown sync helpers
  help        Show this help message

Examples:
  gtd add "Task title" --context work --priority high
  gtd list --context work --status actionable
  gtd done <task-id>
  gtd review --week
  gtd sync --init
  gtd config --auto-commit=false
  gtd subtask <task-id> --add "Item"
  gtd depends <task-id> --on <dependency-id>
  gtd completion zsh

Use 'gtd help <command>' for command details.
`
	fmt.Print(usage)
	return nil
}

func printCommandUsage(command string) error {
	help := map[string]string{
		"add": `Usage: gtd add "title" [flags]
Flags:
  --context, -c       Comma-separated contexts
  --priority, -p      none|low|medium|high
  --due, -d           today|tomorrow|next week|YYYY-MM-DD
  --tags, -t          Comma-separated tags
  --recurrence        none|daily|weekly|monthly
  --description       Long description
`,
		"list": `Usage: gtd list [flags]
Flags:
  --context, -c       Filter by context(s)
  --status            Filter by status
  --priority          Filter by priority
  --overdue           Overdue tasks only
  --due-today         Due today only
  --upcoming N        Due within N days
  --format, -f        table|json|csv
`,
		"update": `Usage: gtd update <task-id> [flags]
Flags:
  --title             New title
  --status            inbox|actionable|waiting|someday|done
  --priority          none|low|medium|high
`,
		"view": `Usage: gtd view <task-id>
Shows full task details including due date, notes, subtasks, and dependencies.
`,
		"done": `Usage: gtd done <task-id>
Marks a task as complete and sets completed timestamp.
For recurring tasks, creates the next instance automatically.
`,
		"delete": `Usage: gtd delete <task-id>
Deletes a task permanently from local storage.
`,
		"inbox": `Usage: gtd inbox
Lists tasks currently in inbox status.
`,
		"today": `Usage: gtd today
Shows actionable tasks due today (or all actionable tasks when none are due today).
`,
		"review": `Usage: gtd review
Prints a weekly GTD summary: status counts, completed-this-week, and overdue totals.
`,
		"search": `Usage: gtd search <query> [flags]
Flags:
  --exact             Exact phrase match
  --regex PATTERN     Regex search
  --json              JSON output
`,
		"sync": `Usage: gtd sync [flags]
Flags:
  --init              Initialize git repo at data path
  --daemon            Run periodic sync loop
  --interval N        Sync interval minutes
  --remote NAME       Remote name (default origin)
  --branch NAME       Branch name
  --resolve STRATEGY  Conflict strategy: ours|theirs
`,
		"config": `Usage: gtd config [flags]
Flags:
  --default-context VALUE
  --add-context VALUE
  --auto-commit true|false
  --auto-sync true|false
  --sync-interval MINUTES
  --git-remote NAME
  --git-branch NAME
`,
		"subtask": `Usage: gtd subtask <task-id> [flags]
Flags:
  --add "title"       Add subtask
  --done N            Mark subtask N complete (1-based)
`,
		"depends": `Usage: gtd depends <task-id> [flags]
Flags:
  --on <task-id>      Add dependency
  --remove <task-id>  Remove dependency
  --check             Check whether task is unblocked
`,
		"serve": `Usage: gtd serve [--addr 127.0.0.1:8080]
Endpoints:
  GET   /healthz
  GET   /tasks
  POST  /tasks
  GET   /tasks/{id}
  PATCH /tasks/{id}
  DELETE /tasks/{id}
  POST  /webhooks/remarkable/complete
`,
		"server": `Usage: gtd server [setup]
Subcommands:
  setup    Run interactive server bootstrap (password + TOTP + initial API key)
Without subcommands, starts the authenticated API server using ~/.gtd/server-config.json.
`,
		"completion": `Usage: gtd completion bash|zsh`,
		"remarkable": `Usage: gtd remarkable [flags]
Flags:
  --export
  --import <file>
  --path <file>
`,
		"help": `Usage: gtd help [command]
Without a command, prints global help.
With a command (for example: gtd help serve), prints command-specific usage and flags.
`,
	}

	text, ok := help[command]
	if !ok {
		return fmt.Errorf("unknown help topic: %s", command)
	}
	fmt.Print(text)
	return nil
}
