import { login, logout } from "./auth.js";
import { fetchHealth, fetchStatus } from "./api.js";
import { renderInbox, renderProjects, renderReview, renderTasks, renderToday } from "./tasks.js";
import { renderSettings } from "./settings.js";

const state = {
  authenticated: false,
  currentView: "tasks",
  rendering: false,
  mobileNavOpen: false,
};

const panels = Array.from(document.querySelectorAll("[data-view-panel]"));
const navButtons = Array.from(document.querySelectorAll(".nav-btn[data-view]"));
const loginView = document.getElementById("login-view");
const loginForm = document.getElementById("login-form");
const loginError = document.getElementById("login-error");
const statusBackend = document.getElementById("status-backend");
const statusAuth = document.getElementById("status-auth");
const statusAPI = document.getElementById("status-api");
const mobileNavToggle = document.getElementById("mobile-nav-toggle");
const mobileNavPanel = document.getElementById("mobile-nav-panel");

const roots = {
  tasks: document.getElementById("tasks-root"),
  inbox: document.getElementById("inbox-root"),
  today: document.getElementById("today-root"),
  review: document.getElementById("review-root"),
  projects: document.getElementById("projects-root"),
  settings: document.getElementById("settings-root"),
};

boot().catch((err) => {
  console.error(err);
});

async function boot() {
  bindNav();
  bindMobileNav();
  bindLogin();
  await trySessionBootstrap();
  render();
  setInterval(() => {
    renderStatusFooter().catch((err) => console.error(err));
  }, 30000);
}

function bindNav() {
  for (const btn of navButtons) {
    btn.addEventListener("click", async () => {
      state.currentView = btn.dataset.view;
      closeMobileNav();
      await render();
    });
  }
}

function bindMobileNav() {
  if (!mobileNavToggle || !mobileNavPanel) {
    return;
  }
  mobileNavToggle.addEventListener("click", () => {
    state.mobileNavOpen = !state.mobileNavOpen;
    syncMobileNav();
  });
  document.addEventListener("click", (event) => {
    if (!state.mobileNavOpen) {
      return;
    }
    if (mobileNavPanel.contains(event.target) || mobileNavToggle.contains(event.target)) {
      return;
    }
    closeMobileNav();
  });
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && state.mobileNavOpen) {
      closeMobileNav();
    }
  });
}

function bindLogin() {
  loginForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    loginError.textContent = "";
    const email = document.getElementById("login-email").value.trim();
    const password = document.getElementById("login-password").value;
    const totpCode = document.getElementById("login-totp").value.trim();

    try {
      await login(email, password, totpCode);
      state.authenticated = true;
      state.currentView = "tasks";
      await render();
    } catch (err) {
      loginError.textContent = err.message || "Login failed";
    }
  });
}

async function trySessionBootstrap() {
  try {
    await fetchStatus();
    state.authenticated = true;
  } catch {
    state.authenticated = false;
  }
}

async function render() {
  if (state.rendering) {
    return;
  }
  state.rendering = true;
  setActiveNav();
  await renderStatusFooter();

  if (!state.authenticated) {
    showPanel("login");
    state.rendering = false;
    return;
  }

  showPanel(state.currentView);
  try {
    switch (state.currentView) {
      case "tasks":
        await renderTasks(roots.tasks);
        break;
      case "inbox":
        await renderInbox(roots.inbox);
        break;
      case "today":
        await renderToday(roots.today);
        break;
      case "review":
        await renderReview(roots.review);
        break;
      case "projects":
        await renderProjects(roots.projects);
        break;
      case "settings":
        await renderSettings(roots.settings);
        addLogoutButton(roots.settings);
        break;
      default:
        await renderTasks(roots.tasks);
        break;
    }
  } finally {
    state.rendering = false;
  }
}

function showPanel(viewName) {
  for (const panel of panels) {
    const active = panel.dataset.viewPanel === viewName;
    panel.classList.toggle("is-visible", active);
  }
}

function setActiveNav() {
  for (const btn of navButtons) {
    const active = btn.dataset.view === state.currentView && state.authenticated;
    btn.classList.toggle("is-active", active);
    btn.disabled = !state.authenticated;
  }
  syncMobileNav();
}

function closeMobileNav() {
  state.mobileNavOpen = false;
  syncMobileNav();
}

function syncMobileNav() {
  if (!mobileNavToggle || !mobileNavPanel) {
    return;
  }
  const visible = state.mobileNavOpen && state.authenticated;
  mobileNavPanel.hidden = !visible;
  mobileNavToggle.setAttribute("aria-expanded", visible ? "true" : "false");
  mobileNavToggle.disabled = !state.authenticated;
}

function addLogoutButton(root) {
  const wrap = document.createElement("div");
  wrap.style.marginTop = "12px";

  const btn = document.createElement("button");
  btn.className = "btn";
  btn.textContent = "Log Out";
  btn.addEventListener("click", async () => {
    try {
      await logout();
    } finally {
      state.authenticated = false;
      state.currentView = "tasks";
      await render();
    }
  });

  wrap.appendChild(btn);
  root.appendChild(wrap);
}

async function renderStatusFooter() {
  try {
    await fetchHealth();
    statusBackend.textContent = "Backend: online";
  } catch {
    statusBackend.textContent = "Backend: offline";
    statusAuth.textContent = "Auth: unavailable";
    statusAPI.textContent = "API: -";
    return;
  }

  if (!state.authenticated) {
    statusAuth.textContent = "Auth: signed out";
    statusAPI.textContent = "API: v1";
    return;
  }

  try {
    const status = await fetchStatus();
    const apiVersion = status && status.api ? status.api.version : "v1";
    const uptime = status && status.runtime ? status.runtime.uptime_seconds : "?";
    statusAuth.textContent = "Auth: signed in";
    statusAPI.textContent = `API: ${apiVersion} • Uptime: ${uptime}s`;
  } catch {
    statusAuth.textContent = "Auth: session issue";
    statusAPI.textContent = "API: unavailable";
  }
}

