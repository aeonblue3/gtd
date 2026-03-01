# GTD CLI - Learning Project

A complete learning system to build your Go programming skills while creating a real Getting Things Done (GTD) tool.

## What's Inside

This archive contains:

1. **Phase 1 Framework** - Complete Go CLI application
   - Models, storage, commands
   - All core functionality ready to extend

2. **Exercise 2.1** - Your first coding challenge
   - Test file with 7 test cases
   - Stub code ready to implement
   - Detailed learning guides

3. **Complete Documentation** - Everything you need to learn
   - Learning concepts and guides
   - Step-by-step implementation instructions
   - Reference solutions and hints

## Quick Start

1. Extract this archive
2. Run: `./setup.sh` to set up directories
3. Read: `docs/START_HERE_EXERCISE_2_1.md`
4. Read: `docs/EXERCISE_2_1_GUIDE.md`
5. Code: Implement `internal/git/git.go`
6. Test: `go test ./internal/git -v`

## Current Command Highlights

The CLI currently supports core task workflows plus Phase 2/3 capabilities:
For complete command/flag reference, see `docs/COMMANDS.md`.

- `gtd add "Task" --context work --priority high --due tomorrow`
- `gtd list --context work --overdue`
- `gtd list --upcoming 7 --format table|json|csv`
- `gtd search "query" --exact|--regex "pattern" --json`
- `gtd config` (show current settings)
- `gtd config --auto-commit=false --sync-interval=20`
- `gtd sync --init` (initialize git repo in data path)
- `gtd sync` (one-time pull/push)
- `gtd sync --daemon --interval 30`
- `gtd sync --resolve ours|theirs` (conflict strategy helper)
- `gtd subtask <task-id> --add "first checklist item"`
- `gtd subtask <task-id> --done 1`
- `gtd depends <task-id> --on <dependency-id>`
- `gtd depends <task-id> --check`
- `gtd serve --addr 127.0.0.1:8080` (optional HTTP API)
- `gtd completion bash|zsh`
- `gtd remarkable --export --path ~/.gtd/remarkable_tasks.md`
- `gtd remarkable --import ~/.gtd/remarkable_tasks.md`

## Directory Structure

```
gtd/
├── cmd/
│   └── main.go           # Entry point
├── internal/
│   ├── cli/
│   │   ├── cli.go        # CLI router
│   │   └── commands/     # Command implementations (add, list, etc)
│   ├── config/
│   │   └── config.go     # Configuration management
│   ├── models/
│   │   └── task.go       # Data models
│   ├── storage/
│   │   └── store.go      # File persistence
│   ├── parser/
│   │   └── parser.go     # Date/time parsing
│   └── git/              # ← EXERCISE 2.1 (YOU BUILD THIS)
│       ├── git.go        # Stub code - implement these functions
│       └── git_test.go   # Tests (7 test cases)
├── docs/                 # Documentation files
├── scripts/              # Utility scripts
├── go.mod               # Go module definition
├── setup.sh             # Setup script
└── README.md            # This file
```

## Exercise 2.1: Git Command Wrapper

Your first task is to implement 4 functions in `internal/git/git.go`:

1. **ExecGit()** - Execute git commands safely
   - Capture stdout and stderr
   - Handle errors properly

2. **IsGitRepo()** - Check if a directory is a git repo
   - Return true/false
   - Uses ExecGit internally

3. **GetCurrentBranch()** - Get current git branch name
   - Parse string output
   - Return branch name

4. **GetLastCommitTime()** - Get last commit timestamp
   - Parse integer output
   - Return Unix timestamp

**Time:** 1-2 hours
**Difficulty:** ⭐⭐ (Medium)
**Status:** Tests ready, stub code provided

## Learning Resources

The `docs/` directory contains:

- `YOU_ARE_HERE.md` - Visual roadmap of your journey
- `START_HERE_EXERCISE_2_1.md` - Quick start guide
- `EXERCISE_2_1_GUIDE.md` - Detailed learning guide
- `EXERCISE_2_1_IMPLEMENTATION.md` - Step-by-step instructions
- `EXERCISE_2_1_REFERENCE.md` - Help and solutions
- `GTD_LEARNING_PLAN.md` - All 7 phases detailed
- And more...

## Getting Started

### Option 1: Recommended Path
1. Read `docs/START_HERE_EXERCISE_2_1.md` (10 min)
2. Read `docs/EXERCISE_2_1_GUIDE.md` (30 min)
3. Open `internal/git/git.go` and start implementing
4. Keep `docs/EXERCISE_2_1_IMPLEMENTATION.md` open while coding

### Option 2: Jump Right In
```bash
# Navigate to project
cd gtd

# See tests fail (expected!)
go test ./internal/git -v

# Open the stub file
nano internal/git/git.go

# Start implementing
```

## Requirements

- Go 1.21 or later (https://golang.org/dl/)
- Git installed (for testing)
- A text editor (VSCode, Vim, etc.)

## Installation

1. Extract the archive:
   ```bash
   tar -xzf gtd-learning.tar.gz
   cd gtd
   ```

2. Run setup:
   ```bash
   ./setup.sh
   ```

3. Download Go dependencies:
   ```bash
   go mod download
   ```

## Next Steps

1. **Right now:** Read the documentation starting with `docs/START_HERE_EXERCISE_2_1.md`
2. **Then:** Implement Exercise 2.1 (the git wrapper)
3. **After:** Move to Exercise 2.2 (auto-commit integration)
4. **Continue:** Work through Phase 2 (5 exercises total)
5. **Future:** Phases 3-7 (15+ more exercises)

## What You'll Learn

### Exercise 2.1 (You're Starting Here)
- Go's os/exec package
- Error handling and wrapping
- Testing with TDD
- Working with git from Go

### Phase 2 (After Ex. 2.1)
- Goroutines and channels
- Background synchronization
- Installation automation
- User configuration

### Phases 3-7 (Future)
- Data structures and algorithms
- Web APIs and HTTP servers
- Testing strategies
- System design
- Third-party integrations

## Support

Everything you need is in the `docs/` directory:

- **Stuck?** Check `docs/EXERCISE_2_1_REFERENCE.md`
- **Need concepts?** Read `docs/EXERCISE_2_1_GUIDE.md`
- **Lost?** See `docs/YOU_ARE_HERE.md`
- **Planning ahead?** Check `docs/GTD_LEARNING_PLAN.md`

## Success Criteria

When Exercise 2.1 is complete:
- ✅ All 7 tests pass: `go test ./internal/git -v`
- ✅ You wrote all code yourself
- ✅ You understand every line
- ✅ 4-5 focused commits made
- ✅ Ready for Exercise 2.2

## Timeline

- **Phase 2** (5 exercises): ~2 weeks
- **Phases 3-7** (10+ exercises): ~10 weeks
- **Total:** ~12 weeks to mastery

## Key Philosophy

- **Build real code** - Not toy examples
- **Learn by doing** - Tests first, code second
- **Understand deeply** - No copy-paste shortcuts
- **Commit frequently** - Small, focused commits
- **Ship features** - Each exercise completes something real

## Good Luck! 🚀

You have everything you need. No searching online. No confusion. Just follow the guides and write the code.

Start with `docs/START_HERE_EXERCISE_2_1.md` now!

---

*GTD Learning System - v1.0*
*Created: February 2026*
*Status: Ready to implement*
