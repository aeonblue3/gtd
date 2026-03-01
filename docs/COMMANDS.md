# GTD CLI Command Reference

This document mirrors the live command help exposed by `gtd help <command>`.

## Global Usage

```text
gtd <command> [options]
```

## Commands

- `add` - Add a new task
- `list` - List tasks
- `view` - View task details
- `update` - Update a task
- `done` - Mark task as complete
- `delete` - Delete a task
- `inbox` - Show inbox tasks
- `today` - Show today's tasks
- `review` - Weekly review
- `search` - Search tasks
- `sync` - Sync local repo with remote
- `config` - Show or update configuration
- `subtask` - Manage task subtasks
- `depends` - Manage task dependencies
- `serve` - Start optional HTTP API server
- `completion` - Generate shell completion script
- `remarkable` - ReMarkable markdown sync helpers

## `add`

```text
Usage: gtd add "title" [flags]
Flags:
  --context, -c       Comma-separated contexts
  --priority, -p      none|low|medium|high
  --due, -d           today|tomorrow|next week|YYYY-MM-DD
  --tags, -t          Comma-separated tags
  --recurrence        none|daily|weekly|monthly
  --description       Long description
```

## `list`

```text
Usage: gtd list [flags]
Flags:
  --context, -c       Filter by context(s)
  --status            Filter by status
  --priority          Filter by priority
  --overdue           Overdue tasks only
  --due-today         Due today only
  --upcoming N        Due within N days
  --format, -f        table|json|csv
```

## `update`

```text
Usage: gtd update <task-id> [flags]
Flags:
  --title             New title
  --status            inbox|actionable|waiting|someday|done
  --priority          none|low|medium|high
```

## `search`

```text
Usage: gtd search <query> [flags]
Flags:
  --exact             Exact phrase match
  --regex PATTERN     Regex search
  --json              JSON output
```

## `sync`

```text
Usage: gtd sync [flags]
Flags:
  --init              Initialize git repo at data path
  --daemon            Run periodic sync loop
  --interval N        Sync interval minutes
  --remote NAME       Remote name (default origin)
  --branch NAME       Branch name
  --resolve STRATEGY  Conflict strategy: ours|theirs
```

## `config`

```text
Usage: gtd config [flags]
Flags:
  --default-context VALUE
  --add-context VALUE
  --auto-commit true|false
  --auto-sync true|false
  --sync-interval MINUTES
  --git-remote NAME
  --git-branch NAME
```

## `subtask`

```text
Usage: gtd subtask <task-id> [flags]
Flags:
  --add "title"       Add subtask
  --done N            Mark subtask N complete (1-based)
```

## `depends`

```text
Usage: gtd depends <task-id> [flags]
Flags:
  --on <task-id>      Add dependency
  --remove <task-id>  Remove dependency
  --check             Check whether task is unblocked
```

## `serve`

```text
Usage: gtd serve [--addr 127.0.0.1:8080]
Endpoints:
  GET   /healthz
  GET   /tasks
  POST  /tasks
  GET   /tasks/{id}
  PATCH /tasks/{id}
  DELETE /tasks/{id}
  POST  /webhooks/remarkable/complete
```

## `completion`

```text
Usage: gtd completion bash|zsh
```

## `remarkable`

```text
Usage: gtd remarkable [flags]
Flags:
  --export
  --import <file>
  --path <file>
```

## Keeping This Updated

When command flags are changed, update this file and `printCommandUsage()` in `internal/cli/cli.go` together.
