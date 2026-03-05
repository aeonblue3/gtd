# GTD API Endpoints

This document describes the current API surface exposed by `gtd server`.

For non-2xx behavior, see `docs/API_ERROR_CONTRACT.md`.

## Base URL

- Default local base URL: `http://127.0.0.1:8080`
- Public health route: `/health`
- Authenticated API routes: `/api/*`
- Versioned API routes (preferred for new clients): `/api/v1/*`

## Versioning Policy

- `v1` is currently the active API version.
- `/api/v1/*` and `/api/*` are currently equivalent.
- New clients should prefer `/api/v1/*` to reduce future migration risk.

## Authentication

Authenticated routes accept either:

- `Authorization: Bearer <api_key>`
- Cookie token (`gtd_api_key` by default, or fallback `access_token`)

For cookie-authenticated state-changing requests (`POST/PUT/PATCH/DELETE`), send:

- `X-CSRF-Token: <value from csrf cookie/login response>`

---

## Health

### `GET /health`
- Auth: none
- Request body: none
- Response `200`:

```json
{
  "status": "ok"
}
```

---

## Auth Endpoints

### `POST /api/auth/login`
- Auth: none
- Body:

```json
{
  "email": "user@example.com",
  "password": "secret",
  "totp_code": "123456"
}
```

- Response `200`:

```json
{
  "session_id": "uuid",
  "user_id": "user-id",
  "expires_at": 1700000000,
  "csrf_token": "token"
}
```

- Notes:
  - Also sets `access_token` and `refresh_token` HttpOnly cookies.
  - Also sets a readable CSRF cookie and returns `csrf_token`.
  - Requires MFA to be enabled first.
  - Access/refresh tokens are omitted from JSON by default for cookie-first browser flow.
  - To include tokens in response body for non-browser clients, send header `X-Return-Tokens: true`.

### `POST /api/auth/logout`
- Auth: required
- Body: none
- Behavior: revokes the currently authenticated token and clears auth cookies.
  - API key token -> revokes matching API key.
  - Session token -> revokes matching auth session.
- Response `200`:

```json
{
  "logged_out": true,
  "revoked_api_key_id": "uuid"
}
```

### `POST /api/auth/refresh`
- Auth: optional (depends on mode)
- Body (optional):

```json
{
  "description": "rotated-client",
  "refresh_token": "optional-refresh-token"
}
```

- Behavior:
  - If `refresh_token` (or refresh cookie) is provided: rotates session tokens and revokes old session.
  - Otherwise with bearer API key token: creates new key and revokes old key.
- CSRF:
  - If using cookie refresh (no bearer token), `X-CSRF-Token` must match CSRF cookie value.
- Response `200`:

```json
{
  "id": "new-key-id",
  "api_key": "new-plaintext-key-returned-once",
  "description": "rotated-client",
  "revoked_api_key_id": "old-key-id"
}
```

- Session refresh response shape:

```json
{
  "session_id": "uuid",
  "user_id": "user-id",
  "expires_at": 1700000000,
  "revoked_session_id": "old-session-id",
  "csrf_token": "token"
}
```

- Notes:
  - Session refresh omits `access_token`/`refresh_token` from JSON unless `X-Return-Tokens: true` is sent.
  - Session cookies are always rotated/set for successful refresh.

### `POST /api/auth/setup-mfa`
- Auth: none
- Body:

```json
{
  "email": "user@example.com",
  "password": "secret"
}
```

- Response `200`:

```json
{
  "email": "user@example.com",
  "secret": "BASE32SECRET",
  "otpauth_url": "otpauth://totp/..."
}
```

### `POST /api/auth/verify-mfa`
- Auth: none
- Body:

```json
{
  "email": "user@example.com",
  "code": "123456"
}
```

- Response `200`:

```json
{
  "verified": true
}
```

### `POST /api/auth/apikey/create`
- Auth: required
- Body (optional):

```json
{
  "description": "cli-client"
}
```

- Response `201`:

```json
{
  "id": "uuid",
  "api_key": "plaintext_key_returned_once",
  "description": "cli-client"
}
```

### `DELETE /api/auth/apikey/{id}`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "revoked": "uuid"
}
```

### `GET /api/auth/sessions`
- Auth: required
- Body: none
- Response `200`:

```json
[
  {
    "id": "session-id",
    "user_id": "user-id",
    "expires_at": 1700000000,
    "created_at": 1700000000,
    "revoked": false
  }
]
```

- Notes:
  - Expired sessions are automatically pruned by the backend and omitted from results.

### `DELETE /api/auth/sessions/{id}`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "revoked": "session-id"
}
```

### `DELETE /api/auth/sessions/all`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "revoked_count": 2
}
```

### `GET /api/auth/csrf`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "csrf_token": "token"
}
```

- Notes:
  - Rotates/sets CSRF cookie and returns matching token.
  - Useful for SPA bootstrap before first state-changing request.

---

## Task Endpoints

### `GET /api/tasks`
- Auth: required
- Query params (optional):
  - `status` (e.g. `actionable`)
  - `context` (e.g. `work`)
  - `priority` (e.g. `high`)
  - `project_id` (preferred)
  - `projectId` (alias)
- Invalid `status`/`priority` values return `400`.
- Body: none
- Response `200`: array of task objects.

### `POST /api/tasks`
- Auth: required
- Body:

```json
{
  "title": "Task title",
  "description": "Details",
  "context": ["work"],
  "projectId": "project-id",
  "location": "Home Office",
  "status": "inbox",
  "priority": "none",
  "dueDate": "2026-02-18T16:00:00Z",
  "tags": ["tag1"],
  "notes": "optional",
  "linkedTasks": ["task-id"],
  "subtasks": [
    {
      "id": "subtask-id",
      "title": "checklist item",
      "description": "optional",
      "notes": "optional",
      "status": "open",
      "priority": "none",
      "dueDate": "2026-02-18T16:00:00Z",
      "location": "Home Office",
      "createdAt": "2026-02-18T12:00:00Z",
      "completedAt": "2026-02-18T13:00:00Z"
    }
  ],
  "recurrence": "none"
}
```

- Notes:
  - `title` is required.
  - Accepts `context` or `contexts` for context arrays.
  - `projectId` is optional; when provided it must reference an existing project.
  - `location` is optional free-text.
  - Defaults: `status=inbox`, `priority=none`, `recurrence=none`.
  - Subtasks use `status: open|done` and `priority: none|low|medium|high`.
  - Subtask defaults when omitted: `status=open`, `priority=none`, `createdAt=now`.
  - Parent task completion is blocked while any subtask is `open`.
  - Enum validation:
    - `status`: `inbox|actionable|waiting|someday|done`
    - `priority`: `none|low|medium|high`
    - `recurrence`: `none|daily|weekly|monthly`

### `GET /api/tasks/{id}`
- Auth: required
- Body: none
- Response `200`: task object

### `PUT /api/tasks/{id}`
- Auth: required
- Body: partial or full task object (same shape as create)
- Behavior:
  - provided fields overwrite existing values
  - `projectId` and `location` can be set/changed with normal partial update semantics
  - if `status` is set to `done`, `completedAt` is set automatically
  - if `status` is `done` while any subtask is still `open`, the request is rejected with `400`
  - enum fields are validated using the same rules as create
  - set `clearDueDate: true` to explicitly clear a due date

### `DELETE /api/tasks/{id}`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "deleted": "task-id"
}
```

### `POST /api/tasks/{id}/complete`
- Auth: required
- Body: none
- Response `200`: updated task object with `status=done` and `completedAt` set
- Notes:
  - returns `400` when subtasks are still open

---

## Project Endpoints

### `GET /api/projects`
- Auth: required
- Body: none
- Response `200`: array of project objects

### `POST /api/projects`
- Auth: required
- Body:

```json
{
  "name": "Project name",
  "description": "optional"
}
```

- Response `201`:

```json
{
  "id": "uuid",
  "name": "Project name",
  "description": "optional",
  "createdAt": "2026-01-01T00:00:00Z"
}
```

### `GET /api/projects/{id}`
- Auth: required
- Body: none
- Response `200`: project object

### `PUT /api/projects/{id}`
- Auth: required
- Body:

```json
{
  "name": "Renamed project",
  "description": "optional"
}
```

- Response `200`: updated project object

### `DELETE /api/projects/{id}`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "deleted": "project-id"
}
```

- Notes:
  - Deleting a project unassigns related tasks (`projectId` is cleared on tasks).

---

## Utility Endpoints

### `GET /api/inbox`
- Auth: required
- Body: none
- Response `200`: tasks where `status=inbox`

### `GET /api/today`
- Auth: required
- Body: none
- Response `200`: actionable tasks due today

### `GET /api/review`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "summary": {
    "inbox": 0,
    "actionable": 0,
    "waiting": 0,
    "someday": 0,
    "done": 0,
    "completed_this_week": 0
  },
  "sections": {
    "overdue": {
      "count": 0,
      "tasks": []
    },
    "due_today": {
      "count": 0,
      "tasks": []
    },
    "stale_waiting": {
      "count": 0,
      "tasks": []
    },
    "done_recent": {
      "count": 0,
      "tasks": []
    }
  }
}
```

### `GET /api/search?q=keyword`
- Auth: required
- Query params:
  - `q` (required)
- Body: none
- Response `200`: tasks with title/description/notes containing query text

### `GET /api/v1/status`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "service": "gtd-api",
  "api": {
    "version": "v1"
  },
  "runtime": {
    "started_at": "2026-02-18T12:00:00Z",
    "uptime_seconds": 123
  },
  "auth": {
    "api_key": true,
    "password_totp": true
  },
  "notifications": {
    "enabled": true,
    "scheduler_active": true,
    "dry_run": true,
    "restart_required": false
  }
}
```

---

## Notification Endpoints

### `POST /api/v1/notifications/run-now`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "ok": true,
  "stats": {
    "scanned": 0,
    "attempted": 0,
    "sent": 0,
    "dry_run": 0,
    "failed": 0,
    "skipped": 0
  }
}
```

### `GET /api/v1/notifications/config`
- Auth: required
- Body: none
- Response `200`:

```json
{
  "enabled": false,
  "check_seconds": 300,
  "dry_run": true,
  "email_to": "",
  "email_from": "gtd@localhost",
  "smtp_host": "",
  "smtp_port": 587,
  "smtp_username": "",
  "has_smtp_password": false,
  "restart_required": false
}
```

### `PUT /api/v1/notifications/config`
- Auth: required
- Body: patch-style update (all fields optional):

```json
{
  "enabled": true,
  "check_seconds": 60,
  "dry_run": true,
  "email_to": "me@example.com",
  "email_from": "gtd@example.com",
  "smtp_host": "smtp.example.com",
  "smtp_port": 587,
  "smtp_username": "smtp-user",
  "smtp_password": "secret"
}
```

- Response `200`: same shape as `GET /api/v1/notifications/config`
- Note:
  - `restart_required=true` indicates notifications were enabled in config but the scheduler is not currently active (server restart needed).

