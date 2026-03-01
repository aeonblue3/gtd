# GTD Server Migration: Phased Implementation Plan

## Phase 1: Database Layer (Foundation)

**Goal:** Replace flat-file JSON storage with SQLite while maintaining existing CLI functionality.

**Schema Design:**
```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    contexts TEXT,  -- JSON array
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    due_date INTEGER,  -- Unix timestamp, nullable
    created_at INTEGER NOT NULL,
    completed_at INTEGER,
    tags TEXT,  -- JSON array
    notes TEXT,
    linked_tasks TEXT,  -- JSON array
    recurrence TEXT
);

CREATE TABLE auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,  -- future-proof even though single-user
    access_token TEXT NOT NULL UNIQUE,
    refresh_token TEXT NOT NULL UNIQUE,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    revoked BOOLEAN DEFAULT 0
);

CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    key_hash TEXT NOT NULL UNIQUE,  -- bcrypt hash
    description TEXT,  -- e.g., "CLI client"
    created_at INTEGER NOT NULL,
    last_used_at INTEGER,
    revoked BOOLEAN DEFAULT 0
);

CREATE TABLE auth_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,  -- login_success, login_failure, token_refresh, etc.
    ip_address TEXT,
    user_agent TEXT,
    timestamp INTEGER NOT NULL,
    metadata TEXT  -- JSON for additional context
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);
CREATE INDEX idx_sessions_access_token ON auth_sessions(access_token);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
```

**Implementation:**
- Create `internal/database/` package
- Implement `db.Open()`, `db.Migrate()` for schema initialization
- Write adapter functions that match your current `storage.Store` interface
- Update `internal/storage/store.go` to use SQLite instead of JSON files
- Keep the same public API so existing CLI commands don't break

**Migration Path:**
- On first run after upgrade, detect `~/.gtd/tasks/*.json` files
- Read them, insert into SQLite, move originals to `~/.gtd/backup/`
- Simple one-time migration script

**Validation:**
All existing CLI commands (`add`, `list`, `update`, `done`, etc.) should work identically to before. Run your existing tests.

---

## Phase 2: HTTP API Server (Core Backend)

**Goal:** Build the REST API that both web UI and CLI client will consume.

**Project Structure:**
```
internal/
├── api/
│   ├── server.go         # HTTP server setup, middleware registration
│   ├── handlers/
│   │   ├── tasks.go      # CRUD endpoints
│   │   ├── auth.go       # Login, logout, token refresh
│   │   └── health.go     # Health check endpoint
│   ├── middleware/
│   │   ├── auth.go       # JWT/API key validation
│   │   ├── ratelimit.go  # Rate limiting
│   │   └── logging.go    # Request logging
│   └── responses.go      # Standard response helpers
├── auth/
│   ├── session.go        # JWT generation/validation
│   ├── apikey.go         # API key generation/validation
│   └── totp.go           # TOTP MFA handling
```

**Endpoints:**

```
# Authentication
POST   /api/auth/login          # Email/password + TOTP code → session tokens
POST   /api/auth/logout         # Revoke current session
POST   /api/auth/refresh        # Exchange refresh token for new access token
POST   /api/auth/setup-mfa      # Generate TOTP secret, return QR code URI
POST   /api/auth/verify-mfa     # Verify TOTP code during setup
POST   /api/auth/apikey/create  # Generate new API key (returns once)
DELETE /api/auth/apikey/:id     # Revoke specific API key
GET    /api/auth/sessions       # List active sessions
DELETE /api/auth/sessions/:id   # Revoke specific session
DELETE /api/auth/sessions/all   # Revoke all sessions

# Tasks
GET    /api/tasks               # List with query params: ?status=actionable&context=work&priority=high
POST   /api/tasks               # Create new task
GET    /api/tasks/:id           # Get single task
PUT    /api/tasks/:id           # Update task
DELETE /api/tasks/:id           # Delete task
POST   /api/tasks/:id/complete  # Mark done (convenience endpoint)

# Utility
GET    /api/inbox               # Unprocessed tasks
GET    /api/today               # Actionable tasks
GET    /api/review              # Weekly review stats
GET    /api/search?q=keyword    # Search tasks

# System
GET    /health                  # Health check (unauthenticated)
```

**Authentication Flow:**

1. **Web UI Login:**
   - User submits email + password + TOTP code
   - Server validates, generates short-lived access token (15 min) and long-lived refresh token (30 days)
   - Tokens returned in HttpOnly, Secure, SameSite=Strict cookies
   - Access token in `access_token` cookie, refresh in `refresh_token` cookie

2. **CLI API Key:**
   - User generates API key via web UI or initial setup script
   - Key displayed once, user stores in `~/.gtd/credentials`
   - CLI reads key, sends in `Authorization: Bearer <api_key>` header
   - Server validates against bcrypt hash in `api_keys` table

**Middleware Chain (per request):**
```
Request → Logging → Rate Limiting → Auth Validation → Handler
```

**Rate Limiting:**
- `/api/auth/login`: 5 attempts per 15 minutes per IP
- All other authenticated endpoints: 100 requests per minute per session/API key
- Use in-memory token bucket, reset on server restart (fine for single user)

**Implementation Libraries:**
- `github.com/go-chi/chi/v5` - Router
- `github.com/golang-jwt/jwt/v5` - JWT handling
- `github.com/pquerna/otp` - TOTP
- `golang.org/x/crypto/bcrypt` - Password + API key hashing
- `github.com/go-chi/httprate` - Rate limiting middleware

**Config:**
Environment variables or `~/.gtd/server-config.json`:
```json
{
  "listen_addr": "127.0.0.1:8080",
  "jwt_secret": "<generated>",
  "totp_issuer": "GTD@eatbrainz.com",
  "db_path": "~/.gtd/gtd.db"
}
```

Generate JWT secret on first run: `openssl rand -base64 32`

---

## Phase 3: Web Frontend (UI)

**Goal:** Responsive single-page app that works on desktop, phone, and iPad.

**Tech Choice:** Vanilla JS + modern CSS (no framework). Keeps it simple, no build step, fast loading.

**File Structure:**
```
web/
├── index.html
├── css/
│   └── style.css
├── js/
│   ├── app.js       # Main application logic
│   ├── api.js       # API client wrapper
│   ├── auth.js      # Login/logout handling
│   └── tasks.js     # Task UI components
└── assets/
    └── (any icons/images)
```

**Served by Go:**
Embed the `web/` directory using `//go:embed` and serve at root path. API at `/api/*`, everything else serves the SPA.

```go
//go:embed web/*
var webFS embed.FS

func (s *Server) setupRoutes() {
    s.router.Get("/api/*", s.apiRoutes())
    s.router.Get("/*", s.serveWeb())
}
```

**UI Views:**
- **Login:** Email, password, TOTP code input
- **Task List:** Filter by status/context/priority, search bar, add button
- **Task Detail:** Inline editing, markdown notes, linked tasks
- **Inbox View:** Quick process workflow (delete, actionable, waiting, someday)
- **Today View:** Due today + actionable high-priority
- **Settings:** API key management, session management, MFA setup

**Mobile Considerations:**
- Viewport meta tag for proper scaling
- Touch-friendly tap targets (44px minimum)
- Swipe gestures for task actions (mark done, delete)
- Offline detection with message to user
- Add to home screen support (PWA manifest)

**PWA Manifest (optional but nice):**
```json
{
  "name": "GTD",
  "short_name": "GTD",
  "start_url": "/",
  "display": "standalone",
  "theme_color": "#000000",
  "icons": [...]
}
```

---

## Phase 4: CLI Client Migration

**Goal:** Make existing CLI tool talk to the server instead of local SQLite.

**New Package:** `internal/client/api_client.go`

**Configuration:**
`~/.gtd/config.json` gains new fields:
```json
{
  "mode": "server",  // or "local" for backward compat
  "server_url": "https://gtd.eatbrainz.com",
  "api_key_file": "~/.gtd/credentials",
  "contexts": [...],
  "local_db_path": "~/.gtd/gtd.db"  // used only in local mode
}
```

**API Key Storage:**
`~/.gtd/credentials` (chmod 600):
```
gtd_api_key=<generated_key>
```

**Refactor:**
- `internal/storage/store.go` becomes an interface
- `internal/storage/local.go` implements the interface using SQLite
- `internal/storage/remote.go` implements the interface using HTTP client
- CLI router checks config, instantiates appropriate storage backend

**Commands remain identical:**
```bash
gtd add "Task title" --context work --priority high
gtd list --status actionable
gtd done <id>
```

User experience unchanged, just backend shifts from local DB to API calls.

**Error Handling:**
Network errors should be clear: "Could not reach server at gtd.eatbrainz.com. Check network connection."

---

## Phase 5: Deployment & TLS

**Goal:** Get the server running on Linode with HTTPS.

**DNS Setup:**
Add A record: `gtd.eatbrainz.com` → your Linode IP

**Systemd Service:**
`/etc/systemd/system/gtd.service`:
```ini
[Unit]
Description=GTD API Server
After=network.target

[Service]
Type=simple
User=chris
WorkingDirectory=/home/chris/gtd-server
ExecStart=/home/chris/gtd-server/gtd-server
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable gtd
sudo systemctl start gtd
```

**Nginx Configuration:**
`/etc/nginx/conf.d/gtd.conf`:
```nginx
server {
    listen 80;
    server_name gtd.eatbrainz.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name gtd.eatbrainz.com;

    ssl_certificate /etc/letsencrypt/live/gtd.eatbrainz.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/gtd.eatbrainz.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Get Certificate:**
```bash
sudo certbot --nginx -d gtd.eatbrainz.com
```

Certbot auto-renews. Check with `sudo certbot renew --dry-run`.

**Nginx Reload:**
```bash
sudo systemctl enable nginx
sudo systemctl start nginx
```

**Firewall:**
```bash
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

---

## Phase 6: Git Backup Automation

**Goal:** Auto-commit database changes to private git repo.

**Setup:**
```bash
cd ~/.gtd
git init
git remote add origin git@github.com:your-private-repo.git
```

**Backup Script:**
`~/.gtd/backup.sh`:
```bash
#!/bin/bash
cd ~/.gtd
sqlite3 gtd.db ".backup gtd-backup.db"
git add gtd-backup.db
git commit -m "Backup $(date -Iseconds)" || true
git push origin main
```

**Cron Job:**
```bash
crontab -e
# Backup every 6 hours
0 */6 * * * /home/chris/.gtd/backup.sh >> /home/chris/.gtd/backup.log 2>&1
```

Alternatively, trigger from Go on writes using a debounced goroutine (commits max once per hour even if many changes).

---

## Phase 7: Initial User Setup

**Goal:** Onboarding flow for first-time setup.

**Setup Command:**
```bash
gtd-server setup
```

**Interactive Prompts:**
1. Create password (bcrypt hashed)
2. Generate TOTP secret, display QR code in terminal (using `github.com/skip2/go-qrcode`)
3. Verify TOTP code
4. Generate initial API key for CLI, save to `~/.gtd/credentials`
5. Save config to `~/.gtd/server-config.json`
6. Initialize database, run migrations

**User Table (add to schema):**
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    totp_secret TEXT,
    totp_enabled BOOLEAN DEFAULT 0,
    created_at INTEGER NOT NULL
);
```

Single-user system, but having the table makes adding users later trivial if needed.

---

## Testing Strategy

**Phase 1:** Unit tests for database operations, verify migration from JSON works.

**Phase 2:** Integration tests hitting API endpoints with test database.

**Phase 3:** Manual testing on desktop, phone (iOS Safari), iPad (Safari). Check responsive breakpoints.

**Phase 4:** CLI commands against local test server, verify network error handling.

**Phase 5:** Deployment smoke test: create task via web UI, list via CLI, verify appears in both.

**Phase 6:** Verify git commits happen, backup restore works.

---

## Timeline Estimate

- **Phase 1:** 4-6 hours (database + migration)
- **Phase 2:** 8-12 hours (API server + auth)
- **Phase 3:** 6-8 hours (web UI)
- **Phase 4:** 2-3 hours (CLI refactor)
- **Phase 5:** 2 hours (deployment)
- **Phase 6:** 1 hour (git automation)
- **Phase 7:** 2 hours (setup flow)

**Total:** ~30 hours spread across weekends or evenings. Phases 1-2 get you a working API. Phase 3 makes it usable on mobile. Everything else is polish.