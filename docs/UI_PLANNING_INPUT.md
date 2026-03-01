# UI Planning Input

This document translates the current backend capabilities into a practical web UI implementation plan.

## 1) Product Scope (Initial UI Release)

Primary goal:

- Personal daily-use web interface for task capture, review, and execution.

Initial release includes:

- Auth flow (email/password/TOTP login + logout)
- Task list with filters/search
- Task create/edit/delete/complete
- Inbox, Today, Review views
- Notification visibility and controls (not full admin panel)

Deferred from initial release:

- Advanced settings UX polish
- Rich editor features (markdown preview, advanced linking UI)
- PWA/add-to-home-screen polish

## 2) Backend Contract Baseline

Authoritative docs:

- `docs/API_ENDPOINTS.md`
- `docs/API_ERROR_CONTRACT.md`

API base:

- Prefer `/api/v1/*` for all UI calls.

Auth model:

- Login returns tokens and sets cookies.
- Cookie-authenticated writes require `X-CSRF-Token`.
- `GET /api/v1/auth/csrf` can rotate/bootstrap CSRF token.

Operational status:

- `GET /api/v1/status` provides capability/runtime info for boot diagnostics.

## 3) User Journeys

### A. First Login + Session Start

1. Open app.
2. If unauthenticated -> show login screen.
3. Submit email/password/TOTP.
4. Save CSRF token in memory state for write requests.
5. Route to Task List.

### B. Capture + Organize

1. Add new task quickly from list/inbox.
2. Edit status/priority/context/due date.
3. Use search/filters to narrow active work.
4. Mark done from list row.

### C. Daily Execution

1. Open Today view.
2. Work through due/actionable tasks.
3. Mark complete or reclassify status.

### D. Session Maintenance

1. App refreshes session via `/auth/refresh` using cookie refresh flow.
2. If refresh fails -> return to login.
3. Logout clears session/cookies and returns to login.

### E. Notification Operations

1. View notification status from Settings.
2. Trigger `run-now` manually to verify reminders.
3. Update notification config fields.
4. If response indicates `restart_required=true`, show explicit guidance.

## 4) Screen Map

### 4.1 Login Screen

Fields:

- Email
- Password
- TOTP code

Actions:

- Login submit
- Error message area (auth failures, rate limits)

### 4.2 Main Task List Screen

Sections:

- Header/nav (List, Inbox, Today, Review, Settings, Logout)
- Filter row (status/context/priority)
- Search bar
- Task table/cards
- Quick add form

Row actions:

- Complete
- Edit inline/basic modal
- Delete

### 4.3 Inbox Screen

- Focused list of inbox tasks
- Fast triage controls:
  - actionable / waiting / someday / delete

### 4.4 Today Screen

- Shows due/actionable tasks for today
- Same row actions as list

### 4.5 Review Screen

- Shows weekly summary counts from `/review`
- Links to filtered list states

### 4.6 Settings Screen (Initial)

- Session/API runtime status (`/status`)
- Notification settings read/update (`/notifications/config`)
- Run notifications now button (`/notifications/run-now`)
- Logout button

## 5) API Mapping by Screen

### Boot/Auth

- `GET /api/v1/status` (auth health/capabilities)
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/csrf`

### Task Management

- `GET /api/v1/tasks`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`
- `POST /api/v1/tasks/{id}/complete`
- `GET /api/v1/search`
- `GET /api/v1/inbox`
- `GET /api/v1/today`
- `GET /api/v1/review`

### Notification Controls

- `GET /api/v1/notifications/config`
- `PUT /api/v1/notifications/config`
- `POST /api/v1/notifications/run-now`

## 6) Frontend Technical Plan (Vanilla JS)

Suggested structure:

```
web/
├── index.html
├── css/
│   └── style.css
├── js/
│   ├── app.js          # Router/state/bootstrap
│   ├── api.js          # Fetch wrapper + CSRF header handling
│   ├── auth.js         # Login/logout/refresh/session helpers
│   ├── tasks.js        # Task list/detail/inbox/today/review UI
│   └── settings.js     # Notification/settings UI
└── assets/
```

Runtime state:

- `authState`: authenticated, csrfToken, lastRefreshAt
- `viewState`: route/view + filters/search
- `taskState`: current list and selected task
- `settingsState`: notification config/status snapshot

## 7) Auth/CSRF Browser Handling Rules

- After login:
  - accept returned `csrf_token`
  - cache in JS memory (and optionally sessionStorage)
- For all non-GET requests with cookie auth:
  - send `X-CSRF-Token` header
- On `401`:
  - try single refresh attempt
  - if refresh fails, route to login
- On `403` CSRF errors:
  - call `GET /api/v1/auth/csrf`, retry once

## 8) Error UX Contract

UI should map status codes consistently:

- `400`: inline validation/problem detail near form
- `401`: auth/session expired -> login flow
- `403`: CSRF/permission issue -> recover/retry guidance
- `404`: missing resource -> toast/banner + list refresh
- `429`: rate limit -> retry timer message
- `500`: generic fault -> non-blocking error banner + retry action

## 9) Responsive/Mobile Planning

Minimum UX goals:

- Tap targets >= 44px
- Collapsible filter panel on small screens
- Task rows become cards on narrow viewports
- Sticky top nav + bottom safe spacing for mobile browsers

Breakpoints (suggested):

- `<= 640px`: mobile
- `641px - 1024px`: tablet
- `> 1024px`: desktop

## 10) Implementation Phases (UI)

### Phase UI-1: Foundation

- Static shell (`index.html`, nav, empty views)
- API client wrapper
- Auth + session bootstrap behavior

### Phase UI-2: Core Task Flow

- Task list + filters + search
- Add/edit/delete/complete interactions

### Phase UI-3: GTD Views

- Inbox / Today / Review

### Phase UI-4: Settings + Notifications

- Status panel
- Notification config editor
- Run-now action + result display

### Phase UI-5: Polish

- Mobile refinements
- Accessibility pass
- Visual consistency + empty/loading states

## 11) Open UI Planning Decisions

Before implementation starts, confirm:

1. List density preference:
   - table-first vs card-first default
2. Edit UX:
   - inline edits vs modal editor
3. Navigation model:
   - hash routing vs simple in-memory view switching
4. Session UX:
   - auto-refresh cadence + idle timeout behavior
5. Notification settings surface:
   - minimal fields first vs full SMTP panel

## 12) Definition of Ready (UI Build Start)

UI implementation can start once:

- These planning decisions are resolved.
- Initial wireframe/screen sketch is accepted.
- API docs are treated as locked for UI-1 through UI-3.
- A test checklist is defined for desktop + mobile manual validation.

