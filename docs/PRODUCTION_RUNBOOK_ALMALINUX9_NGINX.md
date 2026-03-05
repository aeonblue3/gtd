# GTD Production Runbook (AlmaLinux 9 + Nginx)

This runbook covers:

- First production deployment on a fresh AlmaLinux 9 host
- Ongoing (successive) deployments
- Service management, Nginx reverse proxy, TLS, and rollback basics

It assumes Nginx is already installed and running.

## 1) Deployment Topology

- `gtd` Go binary runs as a systemd service on `127.0.0.1:8080`
- Nginx terminates TLS and proxies to `http://127.0.0.1:8080`
- App data/config live under the service user's home:
  - `/home/gtd/.gtd/server-config.json`
  - `/home/gtd/.gtd/gtd.db`
  - `/home/gtd/.gtd/credentials`

## 2) One-Time Server Preparation

Run as root (or with `sudo`).

### 2.1 Create a dedicated service user

```bash
sudo useradd --system --create-home --home-dir /home/gtd --shell /bin/bash gtd
sudo mkdir -p /opt/gtd/{app,bin}
sudo chown -R gtd:gtd /opt/gtd /home/gtd
```

### 2.2 Install build/runtime dependencies

```bash
sudo dnf -y update
sudo dnf -y install git gcc tar sqlite
```

`github.com/mattn/go-sqlite3` uses CGO, so `gcc` is required.

### 2.3 Install Go (recommended: official tarball)

Pick the current stable Go version from `https://go.dev/dl/`.

```bash
cd /tmp
GO_VERSION=1.25.1   # set to current stable from https://go.dev/dl/
curl -LO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
source /etc/profile.d/go.sh
go version
```

## 3) First Deployment

### 3.1 Get source code and build binary

```bash
sudo -u gtd bash -lc 'git clone <YOUR_GIT_REMOTE_URL> /opt/gtd/app'
sudo -u gtd bash -lc 'cd /opt/gtd/app && /usr/local/go/bin/go mod download'
sudo -u gtd bash -lc 'cd /opt/gtd/app && /usr/local/go/bin/go build -o /opt/gtd/bin/gtd ./cmd/main.go'
```

### 3.2 Run interactive server bootstrap once

This creates the initial user, MFA setup, first API key, config, and DB artifacts.

```bash
sudo -u gtd HOME=/home/gtd /opt/gtd/bin/gtd server setup
```

Important:

- Save the generated API key from setup output.
- Keep `/home/gtd/.gtd/credentials` secure (`0600`).
- The QR PNG is written to `/home/gtd/.gtd/totp-setup.png` during setup.

### 3.3 Review and tune server config

Edit:

- `/home/gtd/.gtd/server-config.json`

Recommended baseline:

- `"listen_addr": "127.0.0.1:8080"`
- `"cookie_secure": true` (required for secure cookie behavior behind HTTPS)
- notification settings as needed (`notifications_enabled`, SMTP fields)

Ensure permissions:

```bash
sudo chmod 600 /home/gtd/.gtd/server-config.json /home/gtd/.gtd/credentials
sudo chown gtd:gtd /home/gtd/.gtd/server-config.json /home/gtd/.gtd/credentials
```

### 3.4 Create systemd service

Create `/etc/systemd/system/gtd.service`:

```ini
[Unit]
Description=GTD API/Web Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=gtd
Group=gtd
WorkingDirectory=/opt/gtd/app
Environment=HOME=/home/gtd
ExecStart=/opt/gtd/bin/gtd server
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gtd
sudo systemctl status gtd --no-pager
```

### 3.5 Configure Nginx reverse proxy

Use a dedicated server block (recommended: subdomain like `gtd.example.com`).

`/etc/nginx/conf.d/gtd.conf`:

```nginx
server {
    listen 80;
    server_name gtd.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name gtd.example.com;

    ssl_certificate /etc/letsencrypt/live/gtd.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/gtd.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_connect_timeout 5s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

Validate and reload:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

### 3.6 TLS certificate (if needed)

```bash
sudo dnf -y install certbot python3-certbot-nginx
sudo certbot --nginx -d gtd.example.com
sudo certbot renew --dry-run
```

### 3.7 SELinux and firewall checks (AlmaLinux defaults)

If Nginx cannot proxy to localhost app port and SELinux is enforcing:

```bash
sudo setsebool -P httpd_can_network_connect 1
```

Firewall (if not already open):

```bash
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

### 3.8 Smoke test checklist

```bash
curl -i http://127.0.0.1:8080/health
curl -i https://gtd.example.com/health
curl -i https://gtd.example.com/
```

Also verify login from browser UI.

## 4) Successive Deployments (Standard Path)

Run this sequence for each release.

### 4.1 Pre-deploy backup

```bash
sudo -u gtd bash -lc 'mkdir -p /home/gtd/.gtd/backups'
sudo -u gtd bash -lc 'sqlite3 /home/gtd/.gtd/gtd.db ".backup /home/gtd/.gtd/backups/gtd-$(date +%F-%H%M%S).db"'
```

### 4.2 Pull, build, and restart

```bash
sudo -u gtd bash -lc 'cd /opt/gtd/app && git fetch --all && git pull --ff-only'
sudo -u gtd bash -lc 'cd /opt/gtd/app && /usr/local/go/bin/go mod download'
sudo -u gtd bash -lc 'cp /opt/gtd/bin/gtd /opt/gtd/bin/gtd.prev || true'
sudo -u gtd bash -lc 'cd /opt/gtd/app && /usr/local/go/bin/go build -o /opt/gtd/bin/gtd ./cmd/main.go'
sudo systemctl restart gtd
sudo systemctl status gtd --no-pager
```

Notes:

- DB migrations run automatically on server startup.
- If restart fails after a schema change, check logs before rollback.

### 4.3 Post-deploy verification

```bash
curl -fsS https://gtd.example.com/health
sudo journalctl -u gtd -n 100 --no-pager
```

Verify:

- UI loads
- login works
- create/update task works
- project CRUD works

## 5) Rollback Procedure

If a deploy is bad:

```bash
sudo -u gtd bash -lc 'cp /opt/gtd/bin/gtd.prev /opt/gtd/bin/gtd'
sudo systemctl restart gtd
sudo systemctl status gtd --no-pager
```

If data needs restoration, stop service first and restore latest backup:

```bash
sudo systemctl stop gtd
sudo -u gtd cp /home/gtd/.gtd/backups/<backup-file>.db /home/gtd/.gtd/gtd.db
sudo systemctl start gtd
```

## 6) Operations Reference

### Service control

```bash
sudo systemctl start gtd
sudo systemctl stop gtd
sudo systemctl restart gtd
sudo systemctl status gtd --no-pager
```

### Logs

```bash
sudo journalctl -u gtd -f
sudo journalctl -u gtd -n 200 --no-pager
```

### Nginx

```bash
sudo nginx -t
sudo systemctl reload nginx
sudo systemctl status nginx --no-pager
```

## 7) Known Notes for This Project

- Use `gtd server` (not `gtd serve`) for the current API + web runtime.
- Static UI is served by the Go app at `/`.
- API routes are under both `/api/*` and `/api/v1/*`.
- App auth middleware uses direct socket source (`RemoteAddr`) by design.
  - Expect app-level client IP to appear as the proxy hop unless architecture changes.

## 8) Suggested Next Hardening Steps

- Add automated nightly DB backup + retention policy.
- Add monitoring/alerts (service down, 5xx spikes, cert expiration).
- Add deployment script/CI pipeline to reduce manual steps.
- Add a staging environment before production upgrades.
