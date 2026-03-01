import { apiFetch, setCSRFToken } from "./api.js";

export async function login(email, password, totpCode) {
  const data = await apiFetch("/auth/login", {
    method: "POST",
    body: JSON.stringify({
      email,
      password,
      totp_code: totpCode,
    }),
  });

  if (data && data.csrf_token) {
    setCSRFToken(data.csrf_token);
  } else {
    const csrf = await apiFetch("/auth/csrf");
    setCSRFToken(csrf && csrf.csrf_token ? csrf.csrf_token : "");
  }
  return data;
}

export async function logout() {
  return apiFetch("/auth/logout", { method: "POST" });
}

