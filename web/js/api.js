const API_BASE = "/api/v1";

let csrfToken = "";

export function setCSRFToken(token) {
  csrfToken = token || "";
}

export async function apiFetch(path, options = {}) {
  const csrfRetried = !!options.__csrfRetried;
  const method = (options.method || "GET").toUpperCase();
  const headers = new Headers(options.headers || {});
  const isWrite = !["GET", "HEAD", "OPTIONS"].includes(method);

  if (isWrite && !headers.has("X-CSRF-Token")) {
    await ensureCSRFToken();
  }

  if (options.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (isWrite && csrfToken && !headers.has("X-CSRF-Token")) {
    headers.set("X-CSRF-Token", csrfToken);
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    method,
    headers,
    credentials: "include",
  });

  const text = await res.text();
  const data = text ? safeParseJSON(text) : null;

  if (!res.ok) {
    if (isWrite && isCSRFError(res.status, data) && !csrfRetried) {
      // If the cookie/token pair drifted (e.g. after session bootstrap), refresh once.
      await forceRefreshCSRFToken();
      return apiFetch(path, { ...options, __csrfRetried: true });
    }
    const message = data && data.error ? data.error : `Request failed (${res.status})`;
    const err = new Error(message);
    err.status = res.status;
    err.data = data;
    throw err;
  }

  return data;
}

function safeParseJSON(text) {
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

async function ensureCSRFToken() {
  if (csrfToken) {
    return;
  }
  await forceRefreshCSRFToken();
}

async function forceRefreshCSRFToken() {
  const res = await fetch(`${API_BASE}/auth/csrf`, {
    method: "GET",
    credentials: "include",
  });
  if (!res.ok) {
    return;
  }
  const text = await res.text();
  const data = text ? safeParseJSON(text) : null;
  csrfToken = data && data.csrf_token ? data.csrf_token : "";
}

function isCSRFError(status, data) {
  if (status !== 403) {
    return false;
  }
  const msg = data && data.error ? String(data.error).toLowerCase() : "";
  return msg.includes("csrf");
}

export async function fetchStatus() {
  return apiFetch("/status");
}

export async function fetchHealth() {
  const res = await fetch("/health", { credentials: "include" });
  if (!res.ok) {
    throw new Error(`Health check failed (${res.status})`);
  }
  return res.json();
}

export async function fetchTasks(params = {}) {
  const search = new URLSearchParams(params);
  const suffix = search.toString() ? `?${search}` : "";
  return apiFetch(`/tasks${suffix}`);
}

export async function searchTasks(query) {
  const encoded = encodeURIComponent(query || "");
  return apiFetch(`/search?q=${encoded}`);
}

export async function createTask(input) {
  return apiFetch("/tasks", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function completeTask(taskID) {
  return apiFetch(`/tasks/${encodeURIComponent(taskID)}/complete`, {
    method: "POST",
  });
}

export async function updateTask(taskID, patch) {
  return apiFetch(`/tasks/${encodeURIComponent(taskID)}`, {
    method: "PUT",
    body: JSON.stringify(patch),
  });
}

export async function fetchToday() {
  return apiFetch("/today");
}

export async function fetchReview() {
  return apiFetch("/review");
}

export async function fetchProjects() {
  return apiFetch("/projects");
}

export async function createProject(input) {
  return apiFetch("/projects", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateProject(projectID, input) {
  return apiFetch(`/projects/${encodeURIComponent(projectID)}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteProject(projectID) {
  return apiFetch(`/projects/${encodeURIComponent(projectID)}`, {
    method: "DELETE",
  });
}

export async function fetchNotificationConfig() {
  return apiFetch("/notifications/config");
}

export async function updateNotificationConfig(input) {
  return apiFetch("/notifications/config", {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function runNotificationsNow() {
  return apiFetch("/notifications/run-now", {
    method: "POST",
  });
}

