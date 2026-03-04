#!/usr/bin/env bash
set -euo pipefail

# Idempotent deploy script for GTD server on AlmaLinux/systemd.
# Performs backup -> pull -> build -> restart -> health checks with rollback.

APP_DIR="${APP_DIR:-/opt/gtd/app}"
BRANCH="${BRANCH:-main}"
GO_BIN="${GO_BIN:-go}"
BIN_PATH="${BIN_PATH:-/opt/gtd/bin/gtd}"
PREV_BIN_PATH="${PREV_BIN_PATH:-/opt/gtd/bin/gtd.prev}"
SERVICE_NAME="${SERVICE_NAME:-gtd}"
DATA_DIR="${DATA_DIR:-/home/gtd/.gtd}"
DB_PATH="${DB_PATH:-${DATA_DIR}/gtd.db}"
BACKUP_DIR="${BACKUP_DIR:-${DATA_DIR}/backups}"
LOCK_FILE="${LOCK_FILE:-/tmp/gtd-deploy.lock}"
LOCAL_HEALTH_URL="${LOCAL_HEALTH_URL:-http://127.0.0.1:8080/health}"
PUBLIC_HEALTH_URL="${PUBLIC_HEALTH_URL:-https://gtd.eatbrainz.com/health}"
HEALTH_RETRIES="${HEALTH_RETRIES:-20}"
HEALTH_DELAY_SECONDS="${HEALTH_DELAY_SECONDS:-2}"

timestamp() {
  date "+%Y-%m-%dT%H:%M:%S%z"
}

log() {
  printf "[%s] %s\n" "$(timestamp)" "$*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "ERROR: required command not found: $1"
    exit 1
  fi
}

run_systemctl() {
  if [[ "${EUID}" -eq 0 ]]; then
    systemctl "$@"
  else
    sudo systemctl "$@"
  fi
}

rollback() {
  if [[ -f "${PREV_BIN_PATH}" ]]; then
    log "Rollback: restoring previous binary"
    cp "${PREV_BIN_PATH}" "${BIN_PATH}"
    chmod 0755 "${BIN_PATH}"
    run_systemctl restart "${SERVICE_NAME}"
    run_systemctl --no-pager status "${SERVICE_NAME}" || true
    log "Rollback complete"
  else
    log "Rollback skipped: no previous binary at ${PREV_BIN_PATH}"
  fi
}

health_check() {
  local url="$1"
  local label="$2"
  local attempt=1
  while (( attempt <= HEALTH_RETRIES )); do
    if curl -fsS --max-time 5 "${url}" >/dev/null; then
      log "Health check passed (${label}): ${url}"
      return 0
    fi
    sleep "${HEALTH_DELAY_SECONDS}"
    ((attempt++))
  done
  log "ERROR: health check failed (${label}): ${url}"
  return 1
}

require_cmd git
require_cmd "${GO_BIN}"
require_cmd sqlite3
require_cmd curl
require_cmd flock
require_cmd systemctl

exec 9>"${LOCK_FILE}"
if ! flock -n 9; then
  log "Another deploy is in progress (lock: ${LOCK_FILE})"
  exit 1
fi

log "Starting deploy for branch ${BRANCH}"

if [[ ! -d "${APP_DIR}/.git" ]]; then
  log "ERROR: not a git checkout: ${APP_DIR}"
  exit 1
fi

mkdir -p "${BACKUP_DIR}"
if [[ -f "${DB_PATH}" ]]; then
  BACKUP_PATH="${BACKUP_DIR}/gtd-$(date +%F-%H%M%S).db"
  log "Creating SQLite backup: ${BACKUP_PATH}"
  sqlite3 "${DB_PATH}" ".backup ${BACKUP_PATH}"
fi

log "Updating source in ${APP_DIR}"
cd "${APP_DIR}"
git fetch --all --prune
git checkout "${BRANCH}"
git pull --ff-only origin "${BRANCH}"

if [[ -f "${BIN_PATH}" ]]; then
  cp "${BIN_PATH}" "${PREV_BIN_PATH}"
  chmod 0755 "${PREV_BIN_PATH}"
fi

TMP_BIN="$(mktemp "${BIN_PATH}.new.XXXXXX")"
trap 'rm -f "${TMP_BIN}"' EXIT

log "Downloading modules"
"${GO_BIN}" mod download

log "Building binary"
"${GO_BIN}" build -o "${TMP_BIN}" ./cmd/main.go
chmod 0755 "${TMP_BIN}"
mv "${TMP_BIN}" "${BIN_PATH}"
trap - EXIT

log "Restarting service: ${SERVICE_NAME}"
run_systemctl restart "${SERVICE_NAME}"
run_systemctl --no-pager status "${SERVICE_NAME}" || true

if ! health_check "${LOCAL_HEALTH_URL}" "local"; then
  rollback
  exit 1
fi

if [[ -n "${PUBLIC_HEALTH_URL}" ]]; then
  if ! health_check "${PUBLIC_HEALTH_URL}" "public"; then
    rollback
    exit 1
  fi
fi

log "Deploy successful"
