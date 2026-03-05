import { login, logout } from "./auth.js";
import { createTask, fetchHealth, fetchProjects, fetchStatus } from "./api.js";
import { renderCompleted, renderInbox, renderProjectDetail, renderProjects, renderReview, renderTasks, renderToday } from "./tasks.js";
import { renderSettings } from "./settings.js";

const state = {
  authenticated: false,
  currentView: "tasks",
  rendering: false,
  mobileNavOpen: false,
  currentProjectID: "",
  globalFilters: {
    q: "",
    status: "",
    priority: "",
    context: "",
    projectId: "",
    includeDone: false,
    filtersOpen: false,
  },
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
const globalFiltersRoot = document.getElementById("global-filters");
const activeViewTitle = document.getElementById("active-view-title");
const sidebar = document.getElementById("app-sidebar");
const appShell = document.getElementById("app");
const globalToolbar = document.querySelector(".global-toolbar");

const roots = {
  tasks: document.getElementById("tasks-root"),
  inbox: document.getElementById("inbox-root"),
  today: document.getElementById("today-root"),
  completed: document.getElementById("completed-root"),
  review: document.getElementById("review-root"),
  projects: document.getElementById("projects-root"),
  project: document.getElementById("project-root"),
  settings: document.getElementById("settings-root"),
};

let globalSearchDebounce = null;
let fabMounted = false;
let quickAddProjects = [];

boot().catch((err) => {
  console.error(err);
});

async function boot() {
  bindNav();
  bindMobileNav();
  bindLogin();
  bindRouting();
  applyRouteFromHash();
  await trySessionBootstrap();
  mountGlobalFAB();
  render();
  setInterval(() => {
    renderStatusFooter().catch((err) => console.error(err));
  }, 30000);
}

function bindNav() {
  for (const btn of navButtons) {
    btn.addEventListener("click", () => {
      navigateToView(btn.dataset.view);
      closeMobileNav();
    });
  }
}

function bindRouting() {
  window.addEventListener("hashchange", () => {
    applyRouteFromHash();
    render().catch((err) => console.error(err));
  });
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
  updateViewTitle();
  await renderStatusFooter();

  if (!state.authenticated) {
    showPanel("login");
    hideAuthChrome();
    state.rendering = false;
    return;
  }

  showAuthChrome();
  await renderGlobalFilters();
  showPanel(state.currentView);
  try {
    switch (state.currentView) {
      case "tasks":
        await renderTasks(roots.tasks, currentTaskFilters());
        break;
      case "inbox":
        await renderInbox(roots.inbox, currentTaskFilters());
        break;
      case "today":
        await renderToday(roots.today, currentTaskFilters());
        break;
      case "completed":
        await renderCompleted(roots.completed, currentTaskFilters());
        break;
      case "review":
        await renderReview(roots.review);
        break;
      case "projects":
        await renderProjects(roots.projects);
        break;
      case "project":
        await renderProjectDetail(roots.project, state.currentProjectID, currentTaskFilters(), () => {
          navigateToView("projects");
        });
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
    const activeView = state.currentView === "project" ? "projects" : state.currentView;
    const active = btn.dataset.view === activeView && state.authenticated;
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

function currentTaskFilters() {
  return {
    q: state.globalFilters.q,
    status: state.globalFilters.status,
    priority: state.globalFilters.priority,
    context: state.globalFilters.context,
    projectId: state.globalFilters.projectId,
    includeDone: state.globalFilters.includeDone,
  };
}

function hideAuthChrome() {
  if (appShell) {
    appShell.classList.add("is-unauthenticated");
  }
  if (sidebar) {
    sidebar.style.display = "none";
  }
  if (globalToolbar) {
    globalToolbar.hidden = true;
  }
  if (globalFiltersRoot) {
    globalFiltersRoot.hidden = true;
  }
  if (mobileNavPanel) {
    mobileNavPanel.hidden = true;
  }
  const fab = document.getElementById("global-fab");
  const overlay = document.getElementById("global-fab-overlay");
  if (fab) {
    fab.hidden = true;
  }
  if (overlay) {
    overlay.hidden = true;
  }
}

function showAuthChrome() {
  if (appShell) {
    appShell.classList.remove("is-unauthenticated");
  }
  if (sidebar) {
    sidebar.style.display = "";
  }
  if (globalToolbar) {
    globalToolbar.hidden = false;
  }
  const fab = document.getElementById("global-fab");
  if (fab) {
    fab.hidden = false;
  }
}

function viewLabel(view) {
  switch (view) {
    case "inbox":
      return "Inbox";
    case "today":
      return "Today";
    case "completed":
      return "Completed";
    case "review":
      return "Review";
    case "projects":
      return "Projects";
    case "project":
      return "Project";
    case "settings":
      return "Settings";
    default:
      return "Tasks";
  }
}

function updateViewTitle() {
  if (!activeViewTitle) {
    return;
  }
  activeViewTitle.textContent = state.authenticated ? viewLabel(state.currentView) : "Sign In";
}

function shouldShowGlobalFilters() {
  if (!state.authenticated) {
    return false;
  }
  return !["review", "settings"].includes(state.currentView);
}

function navigateToView(view) {
  state.currentProjectID = "";
  const hash = view === "tasks" ? "" : `#${view}`;
  if (window.location.hash !== hash) {
    window.location.hash = hash;
    return;
  }
  state.currentView = view;
  render().catch((err) => console.error(err));
}

function navigateToProject(projectID) {
  const hash = `#project/${projectID}`;
  if (window.location.hash !== hash) {
    window.location.hash = hash;
    return;
  }
  state.currentView = "project";
  state.currentProjectID = projectID;
  render().catch((err) => console.error(err));
}

function applyRouteFromHash() {
  const hash = (window.location.hash || "").replace(/^#/, "").trim();
  if (!hash) {
    state.currentView = "tasks";
    state.currentProjectID = "";
    return;
  }
  if (hash.startsWith("project/")) {
    const projectID = hash.slice("project/".length).trim();
    if (projectID) {
      state.currentView = "project";
      state.currentProjectID = projectID;
      return;
    }
  }
  const allowed = new Set(["tasks", "inbox", "today", "completed", "review", "projects", "settings"]);
  if (allowed.has(hash)) {
    state.currentView = hash;
    state.currentProjectID = "";
    return;
  }
  state.currentView = "tasks";
  state.currentProjectID = "";
}

async function renderGlobalFilters() {
  if (!globalFiltersRoot) {
    return;
  }
  if (!shouldShowGlobalFilters()) {
    globalFiltersRoot.hidden = true;
    return;
  }
  globalFiltersRoot.hidden = false;
  const filters = state.globalFilters;
  quickAddProjects = await safeFetchProjects();

  const row = document.createElement("div");
  row.className = "global-filter-row";

  const search = document.createElement("input");
  search.type = "search";
  search.placeholder = "Search tasks and subtasks";
  search.value = filters.q;
  search.addEventListener("input", () => {
    filters.q = search.value.trim();
    if (globalSearchDebounce) {
      clearTimeout(globalSearchDebounce);
    }
    globalSearchDebounce = setTimeout(() => {
      render().catch((err) => console.error(err));
    }, 220);
  });

  const toggle = document.createElement("button");
  toggle.type = "button";
  toggle.className = "btn";
  const count = activeGlobalFilterCount();
  toggle.textContent = count > 0 ? `Filters (${count})` : "Filters";
  toggle.addEventListener("click", () => {
    filters.filtersOpen = !filters.filtersOpen;
    render().catch((err) => console.error(err));
  });

  row.appendChild(search);
  row.appendChild(toggle);
  globalFiltersRoot.innerHTML = "";
  globalFiltersRoot.appendChild(row);

  if (!filters.filtersOpen) {
    return;
  }

  const grid = document.createElement("div");
  grid.className = "global-filter-grid";

  const status = document.createElement("select");
  status.innerHTML = `
    <option value="">Status: all</option>
    <option value="inbox">inbox</option>
    <option value="actionable">actionable</option>
    <option value="waiting">waiting</option>
    <option value="someday">someday</option>
    <option value="done">done</option>
  `;
  status.value = filters.status;
  status.addEventListener("change", () => {
    filters.status = status.value;
    render().catch((err) => console.error(err));
  });

  const priority = document.createElement("select");
  priority.innerHTML = `
    <option value="">Priority: all</option>
    <option value="none">none</option>
    <option value="low">low</option>
    <option value="medium">medium</option>
    <option value="high">high</option>
  `;
  priority.value = filters.priority;
  priority.addEventListener("change", () => {
    filters.priority = priority.value;
    render().catch((err) => console.error(err));
  });

  const context = document.createElement("input");
  context.type = "text";
  context.placeholder = "Context";
  context.value = filters.context;
  context.addEventListener("change", () => {
    filters.context = context.value.trim();
    render().catch((err) => console.error(err));
  });

  const project = document.createElement("select");
  project.innerHTML = `<option value="">Project: all</option>`;
  for (const p of quickAddProjects) {
    const option = document.createElement("option");
    option.value = p.id;
    option.textContent = p.name;
    project.appendChild(option);
  }
  project.value = filters.projectId;
  project.addEventListener("change", () => {
    filters.projectId = project.value;
    render().catch((err) => console.error(err));
  });

  const includeDone = document.createElement("label");
  includeDone.className = "global-filter-toggle";
  const includeDoneInput = document.createElement("input");
  includeDoneInput.type = "checkbox";
  includeDoneInput.checked = !!filters.includeDone;
  includeDoneInput.addEventListener("change", () => {
    filters.includeDone = includeDoneInput.checked;
    render().catch((err) => console.error(err));
  });
  includeDone.appendChild(includeDoneInput);
  includeDone.append("Include done");

  const clearBtn = document.createElement("button");
  clearBtn.type = "button";
  clearBtn.className = "btn";
  clearBtn.textContent = "Clear";
  clearBtn.addEventListener("click", () => {
    filters.q = "";
    filters.status = "";
    filters.priority = "";
    filters.context = "";
    filters.projectId = "";
    filters.includeDone = false;
    filters.filtersOpen = false;
    render().catch((err) => console.error(err));
  });

  grid.appendChild(status);
  grid.appendChild(priority);
  grid.appendChild(context);
  grid.appendChild(project);
  grid.appendChild(includeDone);
  grid.appendChild(clearBtn);
  globalFiltersRoot.appendChild(grid);
}

function activeGlobalFilterCount() {
  let count = 0;
  if (state.globalFilters.status) {
    count += 1;
  }
  if (state.globalFilters.priority) {
    count += 1;
  }
  if (state.globalFilters.context) {
    count += 1;
  }
  if (state.globalFilters.projectId) {
    count += 1;
  }
  if (state.globalFilters.includeDone) {
    count += 1;
  }
  return count;
}

function mountGlobalFAB() {
  if (fabMounted) {
    return;
  }
  fabMounted = true;

  const fab = document.createElement("button");
  fab.id = "global-fab";
  fab.type = "button";
  fab.className = "fab-add";
  fab.hidden = true;
  fab.title = "Quick add task";
  fab.setAttribute("aria-label", "Quick add task");
  fab.textContent = "+";

  const overlay = document.createElement("div");
  overlay.id = "global-fab-overlay";
  overlay.className = "quick-add-overlay";
  overlay.hidden = true;

  const modal = document.createElement("section");
  modal.className = "quick-add-modal card stack";

  const header = document.createElement("div");
  header.className = "modal-header";
  const title = document.createElement("h3");
  title.textContent = "Quick Add";
  title.style.margin = "0";
  const close = document.createElement("button");
  close.type = "button";
  close.className = "btn";
  close.textContent = "Close";
  close.addEventListener("click", () => {
    overlay.hidden = true;
  });
  header.appendChild(title);
  header.appendChild(close);

  const form = document.createElement("form");
  form.className = "stack";
  const titleInput = document.createElement("input");
  titleInput.type = "text";
  titleInput.placeholder = "Task title";
  titleInput.required = true;
  const contextInput = document.createElement("input");
  contextInput.type = "text";
  contextInput.placeholder = "Context (optional)";
  const locationInput = document.createElement("input");
  locationInput.type = "text";
  locationInput.placeholder = "Location (optional)";
  const row = document.createElement("div");
  row.style.display = "flex";
  row.style.gap = "8px";
  row.style.flexWrap = "wrap";
  const priority = document.createElement("select");
  priority.innerHTML = `
    <option value="none">Priority: none</option>
    <option value="low">Priority: low</option>
    <option value="medium">Priority: medium</option>
    <option value="high">Priority: high</option>
  `;
  const status = document.createElement("select");
  status.innerHTML = `
    <option value="inbox">Status: inbox</option>
    <option value="actionable">Status: actionable</option>
    <option value="waiting">Status: waiting</option>
    <option value="someday">Status: someday</option>
  `;
  const project = document.createElement("select");
  project.innerHTML = `<option value="">Project: none</option>`;
  row.appendChild(priority);
  row.appendChild(status);
  row.appendChild(project);
  const submit = document.createElement("button");
  submit.type = "submit";
  submit.className = "btn btn-primary";
  submit.textContent = "Add Task";
  const feedback = document.createElement("p");
  feedback.className = "muted";
  feedback.style.margin = "0";
  feedback.style.minHeight = "1rem";
  form.appendChild(titleInput);
  form.appendChild(contextInput);
  form.appendChild(locationInput);
  form.appendChild(row);
  form.appendChild(submit);
  form.appendChild(feedback);

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    feedback.textContent = "";
    submit.disabled = true;
    const idleLabel = submit.textContent;
    submit.textContent = "Saving...";
    try {
      await createTask({
        title: titleInput.value.trim(),
        context: contextInput.value.trim() ? [contextInput.value.trim()] : [],
        location: locationInput.value.trim(),
        projectId: project.value,
        priority: priority.value,
        status: status.value,
      });
      titleInput.value = "";
      contextInput.value = "";
      locationInput.value = "";
      status.value = "inbox";
      priority.value = "none";
      project.value = "";
      feedback.textContent = "Task added.";
      overlay.hidden = true;
      await render();
    } catch (err) {
      feedback.textContent = err.message || "Could not add task";
    } finally {
      submit.disabled = false;
      submit.textContent = idleLabel;
    }
  });

  fab.addEventListener("click", async () => {
    if (!state.authenticated) {
      return;
    }
    quickAddProjects = await safeFetchProjects();
    project.innerHTML = `<option value="">Project: none</option>`;
    for (const p of quickAddProjects) {
      const option = document.createElement("option");
      option.value = p.id;
      option.textContent = p.name;
      project.appendChild(option);
    }
    overlay.hidden = false;
    titleInput.focus();
  });

  overlay.addEventListener("click", (event) => {
    if (event.target === overlay) {
      overlay.hidden = true;
    }
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && !overlay.hidden) {
      overlay.hidden = true;
    }
  });

  modal.appendChild(header);
  modal.appendChild(form);
  overlay.appendChild(modal);
  document.body.appendChild(fab);
  document.body.appendChild(overlay);
}

async function safeFetchProjects() {
  try {
    const projects = await fetchProjects();
    return Array.isArray(projects) ? projects : [];
  } catch {
    return [];
  }
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

