# Backend Readiness (Plan vs Implemented)

This document maps the original migration plan in `plan.md` to what is currently implemented.

## Overall Status

- Backend foundation is strong and test-backed.
- API/auth scaffolding is beyond "skeleton" and usable for a real SPA integration phase.
- UI is not started yet (intentionally).
- CLI remote migration and deployment/backup phases are still pending.
- Architectural direction is now explicit: server-first backend with CLI as thin API client.

## Locked Decisions

These decisions supersede parts of the original `plan.md`:

- No JSON -> SQLite data migration will be implemented.
- CLI will not use local SQLite as an intermediate step.
- Long-term CLI model is thin client -> remote API on Linode.
- Thin client implementation is out-of-repo and will be built as a separate project.
- Dependency model remains normalized with `task_dependencies`.
- Notification system is now on roadmap (server-side due-date email reminders).

## Phase-by-Phase Mapping

### Phase 0 (Added During Implementation)

Status: **Done**

- Added storage abstraction: `internal/storage/backend.go`
- Kept existing JSON local flow stable.
- Added SQLite implementation without forcing CLI migration.
- Added file-based server config scaffolding.

### Phase 1: Database Layer

Status: **Done for server-side foundation** (with agreed deviations)

Implemented:

- `internal/database/` exists with `Open()` and `Migrate()`.
- SQLite schema exists for:
  - `tasks`
  - `task_subtasks`
  - `task_dependencies` (normalized dependency model)
  - `auth_sessions`
  - `api_keys`
  - `auth_events`
  - `users`
- `internal/storage/sqlite_store.go` provides a full task backend (all current task fields preserved).
- Store-level tests exist for SQLite round-trip and cycle detection.

Changed by decision:

- JSON -> SQLite migration script was intentionally skipped.
- CLI will not be migrated to local SQLite; target is remote API mode.
- Dependencies are normalized (`task_dependencies`) instead of `linked_tasks` JSON column.

### Phase 2: HTTP API Server

Status: **Mostly Done**

Implemented:

- New API package and structure under `internal/api/`.
- Middleware chain includes:
  - logging
  - rate limiting
  - auth validation
  - CSRF checks for cookie-authenticated writes
- Route coverage in `internal/api/server.go` includes:
  - Auth: login/logout/refresh/setup-mfa/verify-mfa/apikey create+revoke/sessions list+revoke+revoke all/csrf
  - Tasks: list/create/get/update/delete/complete
  - Utility: inbox/today/review/search
  - System: health
- Auth capabilities:
  - API key lifecycle (create, validate, revoke, rotate)
  - password + TOTP setup and verification
  - session token issuance on login
  - refresh via refresh token
  - logout for API keys and sessions
  - auth event logging to `auth_events`
- Security hardening:
  - cookie secure toggle in config (`cookie_secure`)
  - CSRF cookie/header strategy for browser flows
  - expired session pruning policy
- API integration tests are extensive and passing.
- API version compatibility routes exist for both `/api/*` and `/api/v1/*`.

Differences from original wording:

- `jwt_secret` exists in config, but implementation currently uses opaque random session tokens in DB (not JWT).
- Refresh route supports both refresh-token session flow and API-key rotation fallback.

### Phase 3: Web Frontend

Status: **Not Started** (intentional)

- No `web/` SPA implementation yet.
- No embedded static serving route for web assets yet.

### Phase 4: CLI Client Migration

Status: **Deferred in this repo** (moved out of scope)

Implemented context in this repo:

- Server API contracts and auth flows are being prepared for external clients.

Deferred/out-of-scope for this repo:

- `internal/client/api_client.go`
- `storage/remote.go`
- CLI command routing based on `mode=server`
- Full CLI parity over HTTP backend

Updated scope:

- Thin client will be developed separately against this server API.

### Phase 5: Deployment & TLS

Status: **Not Started**

- No systemd/nginx/certbot automation in repo yet.

### Phase 6: Git Backup Automation

Status: **Not Started**

- No DB backup script/cron/debounce automation implemented yet.

### Phase 7: Initial User Setup

Status: **Mostly Done** (adapted command name)

Implemented:

- Interactive setup flow via `gtd server setup`:
  - prompt email/password
  - generate TOTP secret + QR PNG
  - verify TOTP code
  - generate initial API key
  - write credentials file (`0600`)
  - initialize DB/migrations through startup path

Difference:

- Command is `gtd server setup` instead of `gtd-server setup`.
- QR is written to file (`totp-setup.png`) instead of terminal-rendered QR.

### Phase 8: Notification System (New)

Status: **Partially Done**

Goal:

- Notify user when tasks are due soon or overdue via email.

Implemented:

- `task_notifications` table and indexes added to DB migrations.
- In-process scheduler service added in `internal/notify/`:
  - periodic scan for due windows (`due_24h`, `due_1h`, `overdue`)
  - dedupe via unique notification key
  - persistence of delivery state (`pending`, `dry_run`, `sent`, `failed`)
- Dry-run mode implemented and wired to server startup.
- SMTP transport abstraction implemented for non-dry-run delivery.
- Unit tests added for reminder creation and dedupe behavior.

Remaining:

1. Add notification settings management endpoint/UI hooks.
2. Add richer reminder policy controls (custom windows, per-context toggles).
3. Add retry/backoff for failed deliveries.
4. Add end-to-end tests for SMTP path (or mock transport integration tests).
5. Add observability counters/log structure for operations.

Notes:

- Initial implementation can be single-user and in-process.
- If reliability requirements grow, move to queue/worker model later.

## Testing Strategy Status

- Phase 1-like tests: present (SQLite store behavior + dependency cycle checks).
- Phase 2-like tests: present (API integration tests for auth/tasks/csrf/refresh/session behaviors).
- Phase 3+ manual/device/deployment testing: pending.

## Backend Readiness Before UI Planning/Build

Current readiness: **High**

Recommended remaining backend tasks before UI implementation:

1. Finalize auth contract details for SPA:
   - settle exact cookie names/lifetimes
   - decide whether to keep API-key rotation fallback on `/api/auth/refresh`
2. Add explicit error code contract docs (per endpoint) for frontend handling.
3. Decide whether `jwt_secret` remains (future JWT) or remove to avoid config ambiguity.
4. Implement Phase 8 notification foundation (schema + scheduler + email transport abstraction).

After those are agreed, the backend is in a good position to begin the planned SPA implementation.

