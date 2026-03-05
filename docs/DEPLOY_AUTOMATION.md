# GTD Deploy Automation (GitHub Actions + SSH)

This document wires automated deploys for the current production stack.

## Overview

- Trigger: push to `main`/`master` or manual GitHub workflow dispatch
- Transport: GitHub Actions SSH to server
- Deploy command: `/opt/gtd/app/scripts/deploy.sh`
- Safety: lock file, DB backup, health checks, rollback to previous binary

## 1) What Was Added

- Workflow: `.github/workflows/deploy.yml`
- Deploy script: `scripts/deploy.sh`

The script handles:

1. Exclusive lock (`flock`)
2. SQLite backup
3. Git pull (`--ff-only`)
4. Rebuild binary
5. Restart `gtd` service
6. Local/public health checks
7. Rollback on failure

## 2) Server-Side Prerequisites

Ensure these exist on server:

- Repo checkout at `/opt/gtd/app`
- Binary path `/opt/gtd/bin/gtd`
- Service name `gtd`
- Commands installed: `git`, `go`, `sqlite3`, `curl`, `flock`, `systemctl`

If your deployment user is not root, allow `systemctl` for just this service.

Example `/etc/sudoers.d/gtd-deploy` (use `visudo -f /etc/sudoers.d/gtd-deploy`):

```text
gtd ALL=(root) NOPASSWD: /usr/bin/systemctl restart gtd, /usr/bin/systemctl status gtd
```

Note: confirm actual `systemctl` path with `command -v systemctl`.

## 3) GitHub Secrets

In GitHub repo settings, add:

- `PROD_HOST` - server IP or hostname
- `PROD_PORT` - SSH port (usually `22`)
- `PROD_USER` - SSH user (for example `gtd`)
- `PROD_SSH_KEY` - private key content used by Actions to SSH
- `PROD_PUBLIC_HEALTH_URL` - e.g. `https://gtd.eatbrainz.com/health`

## 4) SSH Key Setup for Actions

Create a dedicated SSH keypair for GitHub Actions and add the public key to the deploy user on server:

```bash
# local machine (or secure admin machine)
ssh-keygen -t ed25519 -f gtd_actions_deploy -C "gtd-actions-deploy" -N ""
```

- Put `gtd_actions_deploy` (private key) in `PROD_SSH_KEY`.
- Append `gtd_actions_deploy.pub` to `/home/gtd/.ssh/authorized_keys` on server.

Permissions on server:

```bash
chmod 700 /home/gtd/.ssh
chmod 600 /home/gtd/.ssh/authorized_keys
chown -R gtd:gtd /home/gtd/.ssh
```

## 5) First Test Run

1. Push a commit to `main` (or manually trigger workflow in GitHub Actions).
2. In Actions logs, confirm `Deploy successful`.
3. Verify service and health:

```bash
sudo systemctl status gtd --no-pager
curl -fsS https://gtd.eatbrainz.com/health
```

## 6) Optional Script Overrides

`scripts/deploy.sh` supports environment overrides:

- `APP_DIR` (default `/opt/gtd/app`)
- `BRANCH` (default `main`)
- `GO_BIN` (default `go`)
- `BIN_PATH` (default `/opt/gtd/bin/gtd`)
- `SERVICE_NAME` (default `gtd`)
- `DATA_DIR` / `DB_PATH` / `BACKUP_DIR`
- `LOCAL_HEALTH_URL` (default `http://127.0.0.1:8080/health`)
- `PUBLIC_HEALTH_URL` (default `https://gtd.eatbrainz.com/health`)
- `HEALTH_RETRIES` / `HEALTH_DELAY_SECONDS`

## 7) Operational Notes

- Keep deploy key and repo key separate.
- Keep deploy backups under `/home/gtd/.gtd/backups`.
- Add retention cleanup later (for example keep last 30 backups).
