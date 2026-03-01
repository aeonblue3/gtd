const API_BASE = "/api/v1";

let csrfToken = "";

export function setCSRFToken(token) {
  csrfToken = token || "";
}

export async function apiFetch(path, options = {}) {
  const method = (options.method || "GET").toUpperCase();
  const headers = new Headers(options.headers || {});
  const isWrite = !["GET", "HEAD", "OPTIONS"].includes(method);

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

