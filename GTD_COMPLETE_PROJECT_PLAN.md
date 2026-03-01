# GTD CLI: Complete Project Plan & Vision

**Project Owner:** Chris (Security Engineer at Mailchimp)  
**Start Date:** February 2025  
**Current Status:** Phase 2, Exercise 2.1 Complete  
**Last Updated:** February 2026

---

## Table of Contents

1. [Vision & Motivation](#vision--motivation)
2. [Project Goals](#project-goals)
3. [Architecture Overview](#architecture-overview)
4. [Phase Breakdown](#phase-breakdown)
5. [Learning Journey](#learning-journey)
6. [Technical Stack](#technical-stack)
7. [Integration Plans](#integration-plans)
8. [Success Metrics](#success-metrics)
9. [Timeline](#timeline)
10. [Future Enhancements](#future-enhancements)

---

## Vision & Motivation

### The Core Idea

Build a **Getting Things Done (GTD) CLI application** that serves as both:
1. **A real productivity tool** for managing tasks and projects
2. **A comprehensive learning platform** for mastering Go programming

### Why This Project?

#### Personal Context
Chris has been a Security Engineer at Mailchimp since June 2017 with extensive experience in security reviews. He's transitioning toward penetration testing and wants to deepen his Go programming skills through practical application.

#### Technical Motivation
- **Build something real:** Not toy examples or tutorials - an actual tool used daily
- **Learn by doing:** Practical application of Go concepts as you build real features
- **Progressive complexity:** Start simple, incrementally add complexity
- **Professional patterns:** Industry-standard architecture and best practices
- **Resume-worthy:** Demonstrate deep Go knowledge and system design thinking

#### Personal Productivity Motivation
- **Overcome friction:** Current bullet journaling workflow needs digital integration
- **Everywhere access:** Works on Mac, Linux, SSH sessions - anywhere terminal exists
- **Version control:** Built-in git history of all tasks (never lose data)
- **Customization:** Fully under your control (not relying on third-party services)
- **Integration ready:** Foundation for syncing to ReMarkable tablet, webhooks, etc.

### The Unique Approach

This isn't a typical GTD app. It's:
- **Text-based, file-based persistence** - your data is portable
- **Git-synchronized** - every change tracked in git history
- **Terminal-native** - works everywhere, no UI dependencies
- **Extensible** - designed for future enhancements and integrations
- **Transparent** - you understand every line of code

---

## Project Goals

### Primary Goals

#### 1. Build a Production-Quality GTD System
**Objective:** Create a fully functional Getting Things Done CLI tool

**Requirements:**
- Task management (create, read, update, delete)
- Status tracking (inbox, actionable, waiting, someday, done)
- Priority levels and context filtering
- Due date management and reminders
- Recurring tasks
- Task dependencies and subtasks
- Local persistence with git synchronization
- Cross-device sync (git-based)

**Success Criteria:**
- All core GTD workflows implemented
- Used daily by the developer
- Clean, well-organized data storage
- Reliable, no data loss

#### 2. Master Go Programming
**Objective:** Develop deep, practical Go skills through hands-on implementation

**Learning Outcomes:**
- Core language concepts (packages, interfaces, error handling)
- Standard library expertise (os/exec, io, json, time, etc.)
- Concurrency patterns (goroutines, channels, context)
- Testing strategies (TDD, table-driven tests, integration testing)
- System programming (file I/O, command execution, process management)
- Software architecture (design patterns, interfaces, composition)
- Git integration and automation
- CLI tool development best practices

**Success Criteria:**
- Can explain every line of code written
- Can implement similar systems independently
- Professional-quality code organization
- Comprehensive test coverage

#### 3. Develop Professional Software Engineering Practices
**Objective:** Demonstrate enterprise-level development skills

**Practices:**
- Test-driven development (TDD)
- Proper error handling with context
- Clean code organization
- Git workflow and meaningful commits
- Documentation and comments
- Architecture design
- Graceful degradation
- Backward compatibility

**Success Criteria:**
- Code reviewable by professional developers
- Clear commit history telling a story
- Comprehensive documentation
- Maintainable, extensible codebase

---

## Project Goals (Continued)

### Secondary Goals

#### 4. Create a Reusable Learning Framework
**Objective:** Build a structured learning system others could follow

**Components:**
- Progressive exercises from simple to complex
- Test-driven curriculum
- Clear specifications
- Detailed explanations of concepts
- Working reference implementations

#### 5. Enable Future Integrations
**Objective:** Design architecture that supports real-world integrations

**Planned Integrations:**
- Git synchronization (phase 2)
- ReMarkable tablet sync (phase 7)
- Webhook support for task updates
- Shell completion (bash/zsh)
- Third-party calendar integration
- API for mobile clients

---

## Architecture Overview

### System Design Philosophy

**Core Principles:**
1. **Simplicity First** - Use the simplest approach that works
2. **File-Based Storage** - Data is portable, version-controllable
3. **No External Dependencies** - Minimal third-party libraries
4. **Git as Sync Layer** - Leverage existing, proven technology
5. **Graceful Degradation** - Features work with or without git
6. **Extensible Foundation** - Easy to add features later

### High-Level Architecture

```
┌─────────────────────────────────────────────────────┐
│                   GTD CLI Application               │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────┐     ┌──────────────┐            │
│  │   Commands   │────→│  Storage     │            │
│  │ (add, list,  │     │  (JSON Files)│            │
│  │  update...)  │     │              │            │
│  └──────────────┘     └──────────────┘            │
│         ▲                     │                    │
│         │                     ▼                    │
│  ┌──────────────────────────────────────┐         │
│  │        Git Integration Layer         │         │
│  │ (commit, sync, history)              │         │
│  └──────────────────────────────────────┘         │
│         │                                         │
│         ▼                                         │
│  ┌──────────────────────────────────────┐         │
│  │   ~/.gtd/ Directory                  │         │
│  │  ├─ tasks/                           │         │
│  │  │  ├─ task_id_1.json                │         │
│  │  │  ├─ task_id_2.json                │         │
│  │  │  └─ ...                           │         │
│  │  ├─ config.json                      │         │
│  │  └─ .git/ (git repository)           │         │
│  └──────────────────────────────────────┘         │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### Data Model

**Task Structure:**
```
Task:
  ├─ ID: unique identifier (UUID)
  ├─ Title: task description
  ├─ Status: inbox | actionable | waiting | someday | done
  ├─ Priority: none | low | medium | high
  ├─ Contexts: [work, oscp, home, music, ...]
  ├─ DueDate: optional deadline
  ├─ CreatedAt: timestamp
  ├─ CompletedAt: optional completion timestamp
  ├─ Tags: [project1, review, urgent, ...]
  ├─ Notes: additional details
  ├─ LinkedTasks: references to related tasks
  └─ Recurrence: daily | weekly | monthly | none
```

### Storage Strategy

**File-Based JSON:**
- Each task = one JSON file (e.g., `~/.gtd/tasks/uuid.json`)
- Simple, portable, git-friendly
- In-memory map for fast queries
- Directory structure organized by status (optional)

**Git Integration:**
- Automatic commits on task changes
- Clear commit messages showing what changed
- Full history for auditing and recovery
- Optional remote sync (GitHub, GitLab, self-hosted)

---

## Phase Breakdown

### Phase 1: Core Framework (✅ Complete)

**Status:** Complete  
**Duration:** Initial setup  
**Deliverables:**

- ✅ Task models with statuses and priorities
- ✅ File-based storage system
- ✅ CLI command router
- ✅ Core commands implemented:
  - add, list, view, update, done, delete
  - inbox, today, review, search
- ✅ Configuration management
- ✅ Date/time parsing utilities

**Skills Learned:**
- Go project structure
- Models and data types
- File I/O operations
- JSON marshaling/unmarshaling
- CLI argument parsing
- Basic testing

---

### Phase 2: Git Integration & Sync

**Status:** In Progress (Exercise 2.1 ✅, Exercise 2.2→)  
**Duration:** ~2-3 weeks  
**Deliverables:**

#### Exercise 2.1: Git Command Wrapper (✅ Complete)
- ✅ ExecGit() - safely execute git commands
- ✅ IsGitRepo() - check if directory is a git repo
- ✅ GetCurrentBranch() - get current branch name
- ✅ GetLastCommitTime() - get last commit timestamp

**Skills:** os/exec, error handling, testing, TDD

#### Exercise 2.2: Auto-Commit Integration (→ In Progress)
- CommitChanges() - stage and commit changes
- GetCommitMessage() - format commit messages
- TryCommit() - attempt commit with graceful degradation

**Skills:** Git workflows, graceful degradation, error types

#### Exercise 2.3: Background Sync Daemon (→ Coming)
- Background goroutine polling for changes
- Automatic syncing on intervals (15-30 min)
- Conflict detection and handling
- Configuration of sync behavior

**Skills:** Goroutines, channels, timing, concurrency

#### Exercise 2.4: Installation & Setup (→ Coming)
- Automated setup script
- Initialize git repository
- Configuration prompts
- Installation verification

**Skills:** Bash scripting, system integration, user experience

#### Exercise 2.5: Configuration Management (→ Coming)
- User preferences (contexts, defaults, sync settings)
- Configuration file management
- Settings persistence
- Configuration validation

**Skills:** Configuration patterns, validation, user preferences

**Learning Outcomes:**
- Git integration in Go
- Concurrency patterns
- Graceful degradation
- User-facing features

---

### Phase 3: Advanced Filtering & Reporting

**Duration:** ~2-3 weeks  
**Planned Exercises:** 3

#### Exercise 3.1: Due Date Filtering
- Query builder pattern
- Filter by date ranges
- Upcoming tasks view
- Overdue task alerts

#### Exercise 3.2: Weekly Review Report
- Generate weekly summary
- Completed task statistics
- Project progress tracking
- Next week planning view

#### Exercise 3.3: Search & Filtering Polish
- Exact match, fuzzy, regex search
- Search result scoring
- Filter combination logic
- Search history

**Learning Outcomes:**
- Data structures and algorithms
- Query optimization
- Statistical analysis
- Text processing

---

### Phase 4: Data Integrity & Testing

**Duration:** ~2-3 weeks  
**Planned Exercises:** 2

#### Exercise 4.1: Comprehensive Test Suite
- Table-driven test patterns
- Integration tests
- Test fixtures and helpers
- Edge case testing

#### Exercise 4.2: Conflict Resolution
- Merge strategies for git conflicts
- Detecting conflicts
- Automatic conflict resolution
- Manual conflict resolution UI

**Learning Outcomes:**
- Advanced testing patterns
- Merge algorithms
- Conflict handling
- Data integrity

---

### Phase 5: Rich Workflows

**Duration:** ~2-3 weeks  
**Planned Exercises:** 3

#### Exercise 5.1: Subtasks & Checklists
- Nested task structures
- Subtask tracking
- Progress calculation
- Subtask completion workflows

#### Exercise 5.2: Task Dependencies
- Graph structures for dependencies
- Cycle detection
- Topological sorting
- Dependency visualization

#### Exercise 5.3: Recurrence & Scheduling
- Recurrence patterns
- Automatic task regeneration
- State machines for scheduling
- Temporal logic

**Learning Outcomes:**
- Complex data structures
- Graph algorithms
- State machines
- Temporal logic

---

### Phase 6: Server & Sync Architecture (Optional)

**Duration:** ~2-3 weeks  
**Status:** Optional enhancement  
**Planned Exercises:** 2

#### Exercise 6.1: HTTP API Basics
- REST API design
- JSON endpoints
- Request validation
- Error responses

#### Exercise 6.2: Client-Server Integration
- Dual-mode operation (CLI + server)
- Fallback strategies
- Network resilience
- Real-time sync option

**Learning Outcomes:**
- Web development in Go
- API design
- Distributed systems
- Network resilience

**Note:** Phase 6 is optional. Git-based sync (Phase 2) handles multi-device sync effectively. Phase 6 adds optional server-based sync for real-time features and mobile clients.

---

### Phase 7: Integrations

**Duration:** ~2-3 weeks  
**Planned Exercises:** 2

#### Exercise 7.1: Shell Completion
- Bash completion generation
- Zsh completion generation
- Dynamic argument completion
- Installation and setup

#### Exercise 7.2: ReMarkable Integration
- File sync with ReMarkable tablet
- Webhook support for task updates
- Document synchronization
- Tablet-to-GTD workflow

**Learning Outcomes:**
- Third-party API integration
- Webhook development
- Device integration
- User workflow integration

---

## Learning Journey

### Progressive Skill Building

The 7-phase curriculum is designed with intentional progression:

**Phase 1: Foundations**
- Project setup
- Basic Go patterns
- Data modeling
- File I/O

**Phase 2: System Integration**
- External process execution
- Git integration
- Background operations
- Error handling patterns

**Phase 3: Advanced Data Operations**
- Complex queries
- Data analysis
- Reporting

**Phase 4: Robustness**
- Comprehensive testing
- Conflict handling
- Data integrity

**Phase 5: Complex Features**
- Nested data structures
- Graph algorithms
- State management

**Phase 6: Distributed Systems**
- Network operations
- Server architecture
- API design

**Phase 7: Real-World Integration**
- Third-party APIs
- Device integration
- Practical workflow patterns

### Learning Methodology

#### Test-Driven Development
- Write tests first
- Tests guide implementation
- Validates correctness
- Documents behavior

#### Build-As-You-Learn
- Each exercise completes a real feature
- Used immediately by the developer
- Motivation from practical value
- Organic problem discovery

#### Incremental Complexity
- One function at a time
- Build on previous knowledge
- Avoid overwhelming complexity
- Clear success criteria

#### Code Ownership
- Write every line yourself
- No copy-paste shortcuts
- Understand every decision
- Learn by doing

---

## Technical Stack

### Languages & Tools

**Primary Language:** Go 1.21+
- Chosen for: simplicity, performance, strong standard library
- Fits workflow: single binary, cross-platform compilation
- Learning objective: master Go thoroughly

**Version Control:** Git
- Central to project design
- Sync mechanism
- History and audit trail
- Widely familiar tool

**Testing:** Go's built-in testing framework
- Table-driven test patterns
- No external dependencies
- Comprehensive coverage
- TDD approach

**Documentation:** Markdown
- Readable and portable
- Version-controllable
- Supports all documentation
- Platform-independent

### Dependencies

**Minimal external dependencies:**
- `github.com/google/uuid` - Task ID generation
- Standard library for everything else

**Rationale:**
- Learn Go thoroughly
- Minimal external complexity
- Maximum portability
- Understanding every dependency

### Development Environment

**Platforms:**
- macOS (primary development)
- Linux (testing and deployment)
- SSH sessions (remote access)

**Tools:**
- Neovim (editor with Go integration)
- Terminal-based workflow
- Git CLI
- Standard Unix tools

---

## Integration Plans

### Phase 2: Git-Based Sync

**Mechanism:**
- Auto-commit on task changes
- Background sync daemon
- Pull changes from remote
- Conflict detection and resolution

**Benefits:**
- Multi-device synchronization
- Full history in git log
- Works offline
- Free hosting options (GitHub, GitLab)

**Implementation:**
- Private GitHub repository
- Auto-commit with cron/launchd
- 15-30 minute sync interval
- Graceful degradation (works without git)

### Phase 7: ReMarkable Tablet

**Objective:** Extend GTD system to handwriting support

**Implementation:**
- Sync task list to ReMarkable
- Capture handwritten updates
- Webhook to update GTD when marked complete
- Bidirectional synchronization

**Benefits:**
- Analog + digital hybrid
- Writing for planning
- Portable review medium
- Integrated workflow

### Optional Phase 6: HTTP Server

**Objective:** Enable real-time sync and mobile support

**Features:**
- REST API for task management
- Real-time WebSocket updates
- Optional server deployment
- Mobile web client (future)

**Benefits:**
- Real-time across devices
- Mobile-friendly interface
- Cloud deployment option
- Professional architecture

---

## Success Metrics

### For the GTD Application

**Functionality:**
- ✅ All Phase 1 features working
- → Phase 2 auto-commit integrated
- All core GTD workflows supported
- Zero data loss incidents
- Used daily by developer

**Quality:**
- >90% test coverage
- All tests passing
- Clean git history
- Well-documented code
- Professional code organization

**Usability:**
- Fast command execution (<100ms)
- Intuitive command structure
- Clear error messages
- Helpful usage documentation

### For the Learning Outcome

**Knowledge:**
- Deep Go proficiency
- System design understanding
- Professional development practices
- Ability to build similar systems

**Skill Demonstration:**
- Public portfolio project
- Clean, reviewable code
- Comprehensive test suite
- Technical documentation

**Code Quality:**
- Idiomatic Go patterns
- Proper error handling
- Clear architecture
- Maintainable design

---

## Timeline

### Historical Progress

| Date | Milestone | Status |
|------|-----------|--------|
| Feb 2025 | Project initiation | ✅ Complete |
| Feb 2025 | Phase 1 complete | ✅ Complete |
| Feb 2025 | Exercise 2.1 complete | ✅ Complete |
| Feb 2025→ | Exercise 2.2 | 🔄 In Progress |

### Planned Timeline

**Phase 2 (Git Integration):** ~2-3 weeks
- Exercises 2.1→2.5
- 4-5 hours/week time commitment
- Completion target: End of February 2025

**Phase 3 (Advanced Filtering):** ~2-3 weeks
- Exercises 3.1→3.3
- Building momentum

**Phases 4-7:** ~8-10 weeks
- Incremental progress
- Real usage refines design
- Estimated completion: Mid-May 2025

**Total Project Duration:** ~12-14 weeks
- ~4-5 hours/week
- 50-60 hours total
- Flexible schedule

### Flexible Approach

- No hard deadlines
- Progress based on learning
- Real-world needs drive priorities
- Can adjust phases based on interests
- Quality over speed

---

## Future Enhancements

### Short Term (After Phase 7)

**Polish & Optimization:**
- Performance optimization
- Command-line argument improvements
- Better error messages
- Enhanced documentation

**User Experience:**
- Colored output
- Better formatting
- Interactive mode
- Improved shell integration

### Medium Term

**Advanced Features:**
- Time tracking
- Pomodoro integration
- Analytics and insights
- Goal setting and progress tracking
- Habit tracking alongside tasks

**Integrations:**
- Calendar apps (iCal, Google Calendar)
- Slack notifications
- Email integration
- Note-taking apps

### Long Term

**Professional Features:**
- Team collaboration
- Shared projects
- Access control
- Audit logging

**Enterprise Features:**
- High-availability setup
- Scalable architecture
- Advanced reporting
- Custom integrations

### Research & Experimentation

**Potential Explorations:**
- AI-powered task categorization
- Natural language processing for task creation
- Predictive priority based on history
- Automated scheduling
- Mobile client (React Native)
- Web interface
- Browser extension

---

## Project Philosophy

### Why This Approach?

#### Learning Effectiveness
- **Concrete goals:** Building real software beats abstract learning
- **Progressive complexity:** Each phase prepares for the next
- **Practical motivation:** Tool gets used immediately
- **Problem-driven:** Challenges emerge naturally

#### Professional Development
- **Portfolio-quality:** Demonstrates real skills
- **Best practices:** Industry-standard patterns
- **Ownership:** Complete understanding
- **Reviewable code:** Professional standards

#### Sustainability
- **Realistic pace:** 4-5 hours/week is manageable
- **Flexibility:** Adjust as priorities change
- **Enjoyment:** Building something useful
- **Growth:** Skills apply to work projects

#### Technical Excellence
- **Minimal dependencies:** Focus on core skills
- **Clean architecture:** Well-organized, maintainable
- **Comprehensive testing:** Quality assurance
- **Good documentation:** Knowledge transfer

---

## Conclusion

This GTD CLI project represents more than task management software. It's a structured approach to:

1. **Mastering Go Programming** - Through practical, progressive exercises
2. **Building Professional Skills** - Industry-standard practices and architecture
3. **Creating Useful Software** - A tool you'll use daily
4. **Establishing Good Habits** - Testing, documentation, clean code
5. **Learning by Doing** - Theory applied immediately to practice

The journey from Exercise 2.1 (git command wrapper) through Phase 7 (integrations) builds progressively deeper skills while creating increasingly valuable software.

### Key Success Factors

✅ **Clear progression** - Each exercise builds on previous knowledge  
✅ **Practical value** - Tool is used immediately  
✅ **Comprehensive guidance** - Detailed specs, tests, and documentation  
✅ **Quality focus** - Professional standards throughout  
✅ **Realistic pace** - Sustainable 4-5 hours/week  
✅ **Flexibility** - Adjust plans as you learn and grow  

### Vision

By the end of this project, you'll have:
- A production-quality GTD CLI tool you use daily
- Deep, practical Go programming expertise
- Professional software engineering skills
- A portfolio project demonstrating real capabilities
- Knowledge to build similar systems independently

---

**This is your learning journey. Make it yours. Build something great.** 🚀

---

*Document Version: 1.0*  
*Last Updated: February 2026*  
*Project Status: Phase 2, Exercise 2.1 Complete*