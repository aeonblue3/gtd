# API Error Contract (v1)

This document defines expected non-2xx responses for `/api/v1/*` (and current alias `/api/*`).

## Standard Error Body

All handled errors should follow:

```json
{
  "error": "human-readable message"
}
```

Notes:

- Message text may evolve; frontend should primarily branch on status code.
- Unknown paths may return framework default `404`.

## Auth + Session Endpoints

### `POST /api/v1/auth/login`

- `400` invalid JSON body
- `401` invalid credentials / invalid MFA / MFA not enabled
- `429` login rate limit exceeded
- `500` unexpected server failure

### `POST /api/v1/auth/setup-mfa`

- `400` invalid JSON body / invalid setup input
- `500` unexpected server failure

### `POST /api/v1/auth/verify-mfa`

- `400` invalid JSON body / invalid code / user not found
- `500` unexpected server failure

### `POST /api/v1/auth/refresh`

- Session refresh mode (`refresh_token` body/cookie):
  - `403` CSRF failure when cookie refresh path is used
  - `401` refresh token invalid/expired
- API-key rotation mode (bearer path):
  - `401` missing/invalid bearer token
- Shared:
  - `400` malformed JSON body (best effort parse; malformed payload may still resolve to `401`)
  - `500` unexpected server failure

### `POST /api/v1/auth/logout` (auth required)

- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

### `POST /api/v1/auth/apikey/create` (auth required)

- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `429` authenticated rate limit exceeded
- `500` key creation/storage error

### `DELETE /api/v1/auth/apikey/{id}` (auth required)

- `400` missing key id path value
- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `404` key id not found
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

### `GET /api/v1/auth/sessions` (auth required)

- `401` missing/invalid auth token
- `429` authenticated rate limit exceeded
- `500` list failure

### `DELETE /api/v1/auth/sessions/{id}` (auth required)

- `400` missing session id
- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `404` session id not found
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

### `DELETE /api/v1/auth/sessions/all` (auth required)

- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

### `GET /api/v1/auth/csrf` (auth required)

- `401` missing/invalid auth token
- `429` authenticated rate limit exceeded
- `500` CSRF token generation failure

## Task Endpoints (all auth required)

### `GET /api/v1/tasks`

- `400` invalid query values (`status`, `priority`)
- `401` missing/invalid auth token
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

### `POST /api/v1/tasks`

- `400` invalid JSON body / missing title / invalid enum values
- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `429` authenticated rate limit exceeded
- `500` storage failure

### `GET /api/v1/tasks/{id}`

- `401` missing/invalid auth token
- `404` task not found
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

### `PUT /api/v1/tasks/{id}`

- `400` invalid JSON body / invalid enum values / empty title update
- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `404` task not found
- `429` authenticated rate limit exceeded
- `500` storage failure

### `DELETE /api/v1/tasks/{id}`

- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `404` task not found
- `429` authenticated rate limit exceeded
- `500` storage failure

### `POST /api/v1/tasks/{id}/complete`

- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `404` task not found
- `429` authenticated rate limit exceeded
- `500` storage failure

## Utility Endpoints (all auth required)

### `GET /api/v1/inbox`
### `GET /api/v1/today`
### `GET /api/v1/review`

- `401` missing/invalid auth token
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

### `GET /api/v1/search?q=...`

- `400` missing `q` query parameter
- `401` missing/invalid auth token
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

### `GET /api/v1/status`

- `401` missing/invalid auth token
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

## System Endpoint

### `GET /health`

- `200` expected healthy response
- `500` only for severe runtime failure

## Notification Endpoints (all auth required)

### `POST /api/v1/notifications/run-now`

- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `429` authenticated rate limit exceeded
- `500` reminder run failure

### `GET /api/v1/notifications/config`

- `401` missing/invalid auth token
- `429` authenticated rate limit exceeded
- `500` unexpected server failure

### `PUT /api/v1/notifications/config`

- `400` invalid JSON body / invalid config values
- `401` missing/invalid auth token
- `403` CSRF failure for cookie-authenticated requests
- `429` authenticated rate limit exceeded
- `500` persistence/runtime update failure

