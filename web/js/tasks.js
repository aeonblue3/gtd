import {
  completeTask,
  createProject,
  createTask,
  deleteProject,
  fetchProjects,
  fetchReview,
  fetchTasks,
  fetchToday,
  updateProject,
  updateTask,
} from "./api.js";

const modal = document.getElementById("task-modal");
const modalForm = document.getElementById("task-modal-form");
const modalCloseBtn = document.getElementById("task-modal-close");
const modalCancelBtn = document.getElementById("task-modal-cancel");
const modalFeedback = document.getElementById("task-modal-feedback");
const modalTaskID = document.getElementById("task-modal-id");
const modalStatus = document.getElementById("task-modal-status");
const modalPriority = document.getElementById("task-modal-priority");
const modalDueDate = document.getElementById("task-modal-due");
const modalDescription = document.getElementById("task-modal-description");
const modalNotes = document.getElementById("task-modal-notes");
const modalContexts = document.getElementById("task-modal-contexts");
const modalTags = document.getElementById("task-modal-tags");
const modalProject = document.getElementById("task-modal-project");
const modalLocation = document.getElementById("task-modal-location");
const toastRegion = document.getElementById("toast-region");

let modalOnSaved = null;
let modalBound = false;
let tasksRenderBusy = false;
const PENDING_LABEL = "Saving...";
let projectsCache = [];
const defaultFilters = {
  q: "",
  status: "",
  priority: "",
  context: "",
  projectId: "",
  includeDone: false,
};

export async function renderTasks(root, filters = {}) {
  if (tasksRenderBusy) {
    return;
  }
  tasksRenderBusy = true;
  await refreshProjects();
  root.innerHTML = "";
  root.appendChild(sectionTitle("Tasks"));

  const loading = loadingIndicator("Loading tasks...");
  root.appendChild(loading);

  try {
    const tasks = await fetchTasksForView("tasks", filters);
    loading.remove();
    root.appendChild(renderTaskList(tasks, async () => {
      await renderTasks(root, filters);
    }, "No tasks match current filters."));
  } catch (err) {
    loading.remove();
    root.appendChild(placeholder("Could not load tasks."));
    showToast(err.message || "Could not load tasks", true);
  } finally {
    tasksRenderBusy = false;
  }
}

export async function renderInbox(root, filters = {}) {
  await refreshProjects();
  const tasks = await fetchTasksForView("inbox", filters);
  root.innerHTML = "";
  root.appendChild(sectionTitle("Inbox"));
  root.appendChild(renderTaskList(tasks, async () => {
    await renderInbox(root, filters);
  }, "Inbox is clear."));
}

export async function renderToday(root, filters = {}) {
  await refreshProjects();
  const tasks = await fetchTasksForView("today", filters);
  root.innerHTML = "";
  root.appendChild(sectionTitle("Today"));
  root.appendChild(renderTaskList(tasks, async () => {
    await renderToday(root, filters);
  }, "No tasks due today."));
}

export async function renderCompleted(root, filters = {}) {
  await refreshProjects();
  const tasks = await fetchTasksForView("completed", filters);
  root.innerHTML = "";
  root.appendChild(sectionTitle("Completed"));
  root.appendChild(renderTaskList(tasks, async () => {
    await renderCompleted(root, filters);
  }, "No completed tasks match current filters."));
}

export async function renderReview(root) {
  await refreshProjects();
  const data = await fetchReview();
  root.innerHTML = "";
  root.appendChild(sectionTitle("Review"));
  root.appendChild(renderReviewSummary(data.summary || {}));
  const sections = data.sections || {};
  root.appendChild(renderReviewSection("Overdue", sections.overdue, root, "No overdue tasks."));
  root.appendChild(renderReviewSection("Due Today", sections.due_today, root, "No tasks due today."));
  root.appendChild(renderReviewSection("Stale Waiting", sections.stale_waiting, root, "No stale waiting tasks."));
  root.appendChild(renderReviewSection("Done Recently", sections.done_recent, root, "No recently completed tasks."));
}

export async function renderProjects(root) {
  await refreshProjects();
  root.innerHTML = "";
  root.appendChild(sectionTitle("Projects"));
  root.appendChild(projectCreateForm(root));
  root.appendChild(projectList(root));
}

export async function renderProjectDetail(root, projectID, filters = {}, onNavigateProjects) {
  await refreshProjects();
  root.innerHTML = "";
  const project = projectsCache.find((item) => item.id === projectID);
  const title = sectionTitle(project ? project.name : "Project");
  root.appendChild(title);
  const meta = document.createElement("p");
  meta.className = "muted";
  meta.textContent = project ? (project.description || "No description") : "Project not found.";
  root.appendChild(meta);

  const backBtn = document.createElement("button");
  backBtn.className = "btn";
  backBtn.type = "button";
  backBtn.textContent = "Back to Projects";
  backBtn.addEventListener("click", () => {
    if (onNavigateProjects) {
      onNavigateProjects();
    }
  });
  root.appendChild(backBtn);

  if (!project) {
    root.appendChild(placeholder("Project does not exist."));
    return;
  }

  const loading = loadingIndicator("Loading project tasks...");
  root.appendChild(loading);
  try {
    const tasks = await fetchTasks({ project_id: projectID });
    loading.remove();
    const filtered = applyGlobalFilters(tasks, { ...filters, projectId: projectID }, "project");
    root.appendChild(renderTaskList(filtered, async () => {
      await renderProjectDetail(root, projectID, filters, onNavigateProjects);
    }, "No tasks in this project match current filters."));
  } catch (err) {
    loading.remove();
    root.appendChild(placeholder("Could not load project tasks."));
    showToast(err.message || "Could not load project tasks", true);
  }
}

function taskCard(task, onChanged) {
  const card = document.createElement("article");
  card.className = "card";
  card.style.padding = "16px";
  card.style.display = "grid";
  card.style.gap = "8px";

  const title = document.createElement("h3");
  title.className = "task-title";
  title.tabIndex = 0;
  title.title = "Click to edit title";
  const titleText = document.createElement("span");
  titleText.textContent = task.title || "(untitled)";
  const titleHint = document.createElement("span");
  titleHint.className = "task-title-hint";
  titleHint.setAttribute("aria-hidden", "true");
  titleHint.textContent = "edit";
  title.appendChild(titleText);
  title.appendChild(titleHint);
  title.addEventListener("click", () => {
    openInlineTitleEditor(title, task, onChanged);
  });
  title.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      openInlineTitleEditor(title, task, onChanged);
    }
  });

  const meta = document.createElement("p");
  meta.className = "muted";
  meta.style.margin = "0";
  const bits = [task.status || "unknown", task.priority || "none"];
  const projectName = projectNameForTask(task);
  if (projectName) {
    bits.push(`project ${projectName}`);
  }
  if (task.location) {
    bits.push(`@ ${task.location}`);
  }
  if (task.dueDate) {
    bits.push(`due ${new Date(task.dueDate).toLocaleDateString()}`);
  }
  meta.textContent = bits.join(" • ");

  const actions = document.createElement("div");
  actions.className = "task-actions";
  const subtasks = normalizeTaskSubtasks(task);
  const hasOpenSubtasks = subtasks.some((subtask) => !isSubtaskDone(subtask));

  const statusSelect = document.createElement("select");
  statusSelect.innerHTML = `
    <option value="inbox">inbox</option>
    <option value="actionable">actionable</option>
    <option value="waiting">waiting</option>
    <option value="someday">someday</option>
    <option value="done">done</option>
  `;
  statusSelect.value = task.status || "inbox";
  const statusSaving = inlineSavingHint();
  statusSelect.addEventListener("change", async () => {
    const next = statusSelect.value;
    statusSelect.disabled = true;
    statusSaving.hidden = false;
    try {
      await updateTask(task.id, { status: next });
      if (onChanged) {
        await onChanged();
      }
    } catch (err) {
      showToast(err.message || "Could not update status", true);
      statusSelect.value = task.status || "inbox";
    } finally {
      statusSaving.hidden = true;
      statusSelect.disabled = false;
    }
  });

  const prioritySelect = document.createElement("select");
  prioritySelect.innerHTML = `
    <option value="none">priority: none</option>
    <option value="low">priority: low</option>
    <option value="medium">priority: medium</option>
    <option value="high">priority: high</option>
  `;
  prioritySelect.value = task.priority || "none";
  const prioritySaving = inlineSavingHint();
  prioritySelect.addEventListener("change", async () => {
    const next = prioritySelect.value;
    prioritySelect.disabled = true;
    prioritySaving.hidden = false;
    try {
      await updateTask(task.id, { priority: next });
      if (onChanged) {
        await onChanged();
      }
    } catch (err) {
      showToast(err.message || "Could not update priority", true);
      prioritySelect.value = task.priority || "none";
    } finally {
      prioritySaving.hidden = true;
      prioritySelect.disabled = false;
    }
  });

  const completeBtn = document.createElement("button");
  completeBtn.className = "btn btn-primary";
  completeBtn.textContent = task.status === "done" ? "Completed" : (hasOpenSubtasks ? "Subtasks Open" : "Mark Done");
  completeBtn.disabled = task.status === "done" || hasOpenSubtasks;
  completeBtn.addEventListener("click", async () => {
    try {
      startButtonPending(completeBtn);
      await completeTask(task.id);
      if (onChanged) {
        await onChanged();
      }
    } catch (err) {
      showToast(err.message || "Could not complete task", true);
    } finally {
      stopButtonPending(completeBtn);
    }
  });

  const dueTodayBtn = document.createElement("button");
  dueTodayBtn.className = "btn";
  dueTodayBtn.textContent = "Due Today";
  dueTodayBtn.addEventListener("click", async () => {
    await quickSetDueDate(task, 0, dueTodayBtn, onChanged);
  });

  const dueTomorrowBtn = document.createElement("button");
  dueTomorrowBtn.className = "btn";
  dueTomorrowBtn.textContent = "Due Tomorrow";
  dueTomorrowBtn.addEventListener("click", async () => {
    await quickSetDueDate(task, 1, dueTomorrowBtn, onChanged);
  });

  const clearDueBtn = document.createElement("button");
  clearDueBtn.className = "btn";
  clearDueBtn.textContent = "Clear Due";
  clearDueBtn.addEventListener("click", async () => {
    try {
      startButtonPending(clearDueBtn);
      await updateTask(task.id, { clearDueDate: true });
      if (onChanged) {
        await onChanged();
      }
    } catch (err) {
      showToast(err.message || "Could not clear due date", true);
    } finally {
      stopButtonPending(clearDueBtn);
    }
  });

  const editBtn = document.createElement("button");
  editBtn.className = "btn";
  editBtn.textContent = "Edit Details";
  editBtn.addEventListener("click", () => {
    openTaskModal(task, onChanged);
  });

  actions.appendChild(statusSelect);
  actions.appendChild(statusSaving);
  actions.appendChild(prioritySelect);
  actions.appendChild(prioritySaving);
  actions.appendChild(dueTodayBtn);
  actions.appendChild(dueTomorrowBtn);
  actions.appendChild(clearDueBtn);
  actions.appendChild(editBtn);
  actions.appendChild(completeBtn);

  const subtaskSection = renderSubtasks(task, subtasks, onChanged);

  card.appendChild(title);
  card.appendChild(meta);
  card.appendChild(actions);
  if (hasOpenSubtasks && task.status !== "done") {
    const blockHint = document.createElement("p");
    blockHint.className = "muted";
    blockHint.style.margin = "0";
    blockHint.textContent = "Complete all subtasks before marking this task done.";
    card.appendChild(blockHint);
  }
  card.appendChild(subtaskSection);
  return card;
}

function renderSubtasks(task, subtasks, onChanged) {
  const section = document.createElement("section");
  section.className = "subtasks-section";

  const doneCount = subtasks.filter((subtask) => isSubtaskDone(subtask)).length;
  const heading = document.createElement("h4");
  heading.className = "subtasks-heading";
  heading.textContent = `Subtasks (${doneCount}/${subtasks.length})`;
  section.appendChild(heading);

  const list = document.createElement("div");
  list.className = "subtasks-list";
  for (let index = 0; index < subtasks.length; index += 1) {
    const subtask = subtasks[index];
    list.appendChild(renderSubtaskRow(task, subtasks, subtask, index, onChanged));
  }
  section.appendChild(list);
  section.appendChild(renderSubtaskCreate(task, subtasks, onChanged));
  return section;
}

function renderSubtaskRow(task, subtasks, subtask, index, onChanged) {
  const item = document.createElement("article");
  item.className = "subtask-item";

  const row = document.createElement("div");
  row.className = "subtask-row";

  const doneLabel = document.createElement("label");
  doneLabel.className = "subtask-done-toggle";
  const doneToggle = document.createElement("input");
  doneToggle.type = "checkbox";
  doneToggle.checked = isSubtaskDone(subtask);
  doneLabel.appendChild(doneToggle);
  doneLabel.append("Done");

  const title = document.createElement("input");
  title.type = "text";
  title.value = subtask.title || "";
  title.placeholder = "Subtask title";

  const priority = document.createElement("select");
  priority.innerHTML = `
    <option value="none">none</option>
    <option value="low">low</option>
    <option value="medium">medium</option>
    <option value="high">high</option>
  `;
  priority.value = subtask.priority || "none";

  const dueDate = document.createElement("input");
  dueDate.type = "date";
  dueDate.value = toDateInputValue(subtask.dueDate || "");

  const location = document.createElement("input");
  location.type = "text";
  location.placeholder = "Location";
  location.value = subtask.location || "";

  const detailsToggle = document.createElement("button");
  detailsToggle.type = "button";
  detailsToggle.className = "btn";
  detailsToggle.textContent = "Details";

  const save = document.createElement("button");
  save.type = "button";
  save.className = "btn";
  save.textContent = "Save";

  const remove = document.createElement("button");
  remove.type = "button";
  remove.className = "btn";
  remove.textContent = "Remove";

  row.appendChild(doneLabel);
  row.appendChild(title);
  row.appendChild(priority);
  row.appendChild(dueDate);
  row.appendChild(location);
  row.appendChild(detailsToggle);
  row.appendChild(save);
  row.appendChild(remove);

  const details = document.createElement("div");
  details.className = "subtask-details";
  details.hidden = true;

  const description = document.createElement("textarea");
  description.rows = 2;
  description.placeholder = "Description";
  description.value = subtask.description || "";

  const notes = document.createElement("textarea");
  notes.rows = 2;
  notes.placeholder = "Notes";
  notes.value = subtask.notes || "";

  details.appendChild(description);
  details.appendChild(notes);

  detailsToggle.addEventListener("click", () => {
    details.hidden = !details.hidden;
  });

  const buildNextSubtasks = () => subtasks.map((current, currentIndex) => {
    if (currentIndex !== index) {
      return current;
    }
    return {
      ...current,
      id: current.id || createClientID(),
      title: title.value.trim(),
      description: description.value.trim(),
      notes: notes.value.trim(),
      status: doneToggle.checked ? "done" : "open",
      priority: priority.value,
      dueDate: dueDate.value ? dateInputToNoonISOString(dueDate.value) : null,
      location: location.value.trim(),
      createdAt: current.createdAt || new Date().toISOString(),
    };
  });

  save.addEventListener("click", async () => {
    const nextTitle = title.value.trim();
    if (!nextTitle) {
      showToast("Subtask title is required.", true);
      title.focus();
      return;
    }
    startButtonPending(save);
    try {
      const nextSubtasks = buildNextSubtasks();
      nextSubtasks[index].title = nextTitle;
      await updateTask(task.id, { subtasks: nextSubtasks });
      if (onChanged) {
        await onChanged();
      }
    } catch (err) {
      showToast(err.message || "Could not update subtask", true);
    } finally {
      stopButtonPending(save);
    }
  });

  doneToggle.addEventListener("change", async () => {
    doneToggle.disabled = true;
    try {
      const nextSubtasks = buildNextSubtasks();
      await updateTask(task.id, { subtasks: nextSubtasks });
      if (onChanged) {
        await onChanged();
      }
    } catch (err) {
      doneToggle.checked = !doneToggle.checked;
      showToast(err.message || "Could not update subtask status", true);
    } finally {
      doneToggle.disabled = false;
    }
  });

  remove.addEventListener("click", async () => {
    startButtonPending(remove);
    try {
      const nextSubtasks = subtasks.filter((_, currentIndex) => currentIndex !== index);
      await updateTask(task.id, { subtasks: nextSubtasks });
      if (onChanged) {
        await onChanged();
      }
    } catch (err) {
      showToast(err.message || "Could not remove subtask", true);
    } finally {
      stopButtonPending(remove);
    }
  });

  item.appendChild(row);
  item.appendChild(details);
  return item;
}

function renderSubtaskCreate(task, subtasks, onChanged) {
  const form = document.createElement("form");
  form.className = "subtask-create";

  const title = document.createElement("input");
  title.type = "text";
  title.placeholder = "Add subtask";
  title.required = true;

  const add = document.createElement("button");
  add.type = "submit";
  add.className = "btn btn-primary";
  add.textContent = "Add";

  form.appendChild(title);
  form.appendChild(add);

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const nextTitle = title.value.trim();
    if (!nextTitle) {
      return;
    }
    startButtonPending(add);
    try {
      const nextSubtasks = [
        ...subtasks,
        {
          id: createClientID(),
          title: nextTitle,
          description: "",
          notes: "",
          status: "open",
          priority: "none",
          dueDate: null,
          location: "",
          createdAt: new Date().toISOString(),
        },
      ];
      await updateTask(task.id, { subtasks: nextSubtasks });
      title.value = "";
      if (onChanged) {
        await onChanged();
      }
    } catch (err) {
      showToast(err.message || "Could not add subtask", true);
    } finally {
      stopButtonPending(add);
    }
  });

  return form;
}

function renderTaskList(tasks, onChanged, emptyMessage) {
  if (!tasks || tasks.length === 0) {
    return placeholder(emptyMessage);
  }
  const list = document.createElement("div");
  list.className = "stack";
  for (const task of tasks.slice(0, 40)) {
    list.appendChild(taskCard(task, onChanged));
  }
  return list;
}

async function fetchTasksForView(view, filters = {}) {
  const f = { ...defaultFilters, ...filters };
  let base = [];
  switch (view) {
    case "inbox":
      base = await fetchTasks({ status: "inbox" });
      break;
    case "today":
      base = await fetchToday();
      break;
    case "completed":
      base = await fetchTasks({ status: "done" });
      break;
    default:
      base = await fetchTasks(compact({
        status: f.status,
        priority: f.priority,
        context: f.context,
        project_id: f.projectId,
      }));
      break;
  }
  return applyGlobalFilters(base, f, view);
}

function compact(obj) {
  const out = {};
  for (const [k, v] of Object.entries(obj)) {
    if (v !== undefined && v !== null && `${v}`.trim() !== "") {
      out[k] = v;
    }
  }
  return out;
}

function normalizeTaskSubtasks(task) {
  const subtasks = Array.isArray(task.subtasks) ? task.subtasks : [];
  return subtasks.map((subtask) => ({
    id: subtask.id || createClientID(),
    title: subtask.title || "",
    description: subtask.description || "",
    notes: subtask.notes || "",
    status: isSubtaskDone(subtask) ? "done" : "open",
    priority: subtask.priority || "none",
    dueDate: subtask.dueDate || null,
    location: subtask.location || "",
    createdAt: subtask.createdAt || new Date().toISOString(),
    completedAt: subtask.completedAt || null,
  }));
}

function isSubtaskDone(subtask) {
  return subtask && (subtask.status === "done" || !!subtask.completedAt);
}

function createClientID() {
  if (window.crypto && typeof window.crypto.randomUUID === "function") {
    return window.crypto.randomUUID();
  }
  return `sub-${Date.now()}-${Math.floor(Math.random() * 1000000)}`;
}

function applyGlobalFilters(tasks, filters, view) {
  const q = (filters.q || "").trim().toLowerCase();
  return (tasks || []).filter((task) => {
    if (view === "completed") {
      if (task.status !== "done") {
        return false;
      }
    } else if (!filters.includeDone && task.status === "done") {
      return false;
    }

    if (filters.status && task.status !== filters.status) {
      return false;
    }
    if (filters.priority && task.priority !== filters.priority) {
      return false;
    }
    if (filters.projectId && (task.projectId || "") !== filters.projectId) {
      return false;
    }
    if (filters.context) {
      const contexts = Array.isArray(task.contexts || task.context) ? (task.contexts || task.context) : [];
      if (!contexts.includes(filters.context)) {
        return false;
      }
    }
    if (q && !matchesTaskQuery(task, q)) {
      return false;
    }
    return true;
  });
}

function matchesTaskQuery(task, q) {
  const baseText = `${task.title || ""}\n${task.description || ""}\n${task.notes || ""}`.toLowerCase();
  if (baseText.includes(q)) {
    return true;
  }
  const subtasks = Array.isArray(task.subtasks) ? task.subtasks : [];
  return subtasks.some((subtask) => {
    const subText = `${subtask.title || ""}\n${subtask.description || ""}\n${subtask.notes || ""}`.toLowerCase();
    return subText.includes(q);
  });
}

function addTaskForm(onAdded, options = {}) {
  const asModal = !!options.asModal;
  const form = document.createElement("form");
  form.className = asModal ? "stack" : "stack card";
  form.style.padding = asModal ? "0" : "16px";
  form.style.marginBottom = asModal ? "0" : "12px";

  const title = document.createElement("input");
  title.type = "text";
  title.placeholder = "Task title";
  title.required = true;

  const context = document.createElement("input");
  context.type = "text";
  context.placeholder = "Context (optional)";

  const location = document.createElement("input");
  location.type = "text";
  location.placeholder = "Location (optional)";

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
  for (const p of projectsCache) {
    const option = document.createElement("option");
    option.value = p.id;
    option.textContent = p.name;
    project.appendChild(option);
  }

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

  form.appendChild(title);
  form.appendChild(context);
  form.appendChild(location);
  form.appendChild(row);
  form.appendChild(submit);
  form.appendChild(feedback);

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    feedback.textContent = "";
    startButtonPending(submit);
    try {
      const contexts = context.value.trim() ? [context.value.trim()] : [];
      await createTask({
        title: title.value.trim(),
        context: contexts,
        location: location.value.trim(),
        projectId: project.value,
        priority: priority.value,
        status: status.value,
      });
      title.value = "";
      context.value = "";
      location.value = "";
      priority.value = "none";
      status.value = "inbox";
      project.value = "";
      feedback.textContent = "Task added.";
      showToast("Task added.");
      if (onAdded) {
        await onAdded();
      }
    } catch (err) {
      feedback.textContent = err.message || "Could not add task";
    } finally {
      stopButtonPending(submit);
    }
  });

  return form;
}

function sectionTitle(text) {
  const h = document.createElement("h2");
  h.className = "section-title";
  h.textContent = text;
  return h;
}

function placeholder(text) {
  const p = document.createElement("p");
  p.className = "placeholder";
  p.textContent = text;
  return p;
}

async function refreshProjects() {
  try {
    const projects = await fetchProjects();
    projectsCache = Array.isArray(projects) ? projects : [];
  } catch {
    projectsCache = [];
  }
}

function fillProjectSelect(selectEl, includeNoneOption) {
  if (!selectEl) {
    return;
  }
  const currentValue = selectEl.value;
  selectEl.innerHTML = "";
  if (includeNoneOption) {
    const none = document.createElement("option");
    none.value = "";
    none.textContent = "none";
    selectEl.appendChild(none);
  }
  for (const project of projectsCache) {
    const option = document.createElement("option");
    option.value = project.id;
    option.textContent = project.name;
    selectEl.appendChild(option);
  }
  if (Array.from(selectEl.options).some((opt) => opt.value === currentValue)) {
    selectEl.value = currentValue;
  }
}

function projectNameForTask(task) {
  if (!task || !task.projectId) {
    return "";
  }
  const project = projectsCache.find((p) => p.id === task.projectId);
  return project ? project.name : "";
}

function renderReviewSummary(summary) {
  const card = document.createElement("section");
  card.className = "card summary-grid";
  const items = [
    ["Inbox", summary.inbox || 0],
    ["Actionable", summary.actionable || 0],
    ["Waiting", summary.waiting || 0],
    ["Someday", summary.someday || 0],
    ["Done", summary.done || 0],
    ["Done This Week", summary.completed_this_week || 0],
  ];
  for (const [label, value] of items) {
    const tile = document.createElement("div");
    tile.className = "summary-tile";
    const k = document.createElement("p");
    k.className = "muted";
    k.style.margin = "0";
    k.textContent = label;
    const v = document.createElement("p");
    v.className = "summary-value";
    v.textContent = `${value}`;
    tile.appendChild(k);
    tile.appendChild(v);
    card.appendChild(tile);
  }
  return card;
}

function renderReviewSection(title, section, root, empty) {
  const card = document.createElement("section");
  card.className = "card";
  card.style.padding = "16px";
  card.style.marginTop = "12px";
  const h = document.createElement("h3");
  h.className = "section-title";
  h.style.marginBottom = "8px";
  const count = section && typeof section.count === "number" ? section.count : 0;
  h.textContent = `${title} (${count})`;
  card.appendChild(h);
  const tasks = section && Array.isArray(section.tasks) ? section.tasks : [];
  card.appendChild(renderTaskList(tasks, async () => {
    await renderReview(root);
  }, empty));
  return card;
}

function projectCreateForm(root) {
  const form = document.createElement("form");
  form.className = "stack card";
  form.style.padding = "16px";
  form.style.marginBottom = "12px";

  const name = document.createElement("input");
  name.type = "text";
  name.placeholder = "Project name";
  name.required = true;

  const desc = document.createElement("input");
  desc.type = "text";
  desc.placeholder = "Description (optional)";

  const submit = document.createElement("button");
  submit.type = "submit";
  submit.className = "btn btn-primary";
  submit.textContent = "Create Project";

  form.appendChild(name);
  form.appendChild(desc);
  form.appendChild(submit);

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    startButtonPending(submit);
    try {
      await createProject({
        name: name.value.trim(),
        description: desc.value.trim(),
      });
      name.value = "";
      desc.value = "";
      showToast("Project created.");
      await renderProjects(root);
    } catch (err) {
      showToast(err.message || "Could not create project", true);
    } finally {
      stopButtonPending(submit);
    }
  });
  return form;
}

function projectList(root) {
  if (!projectsCache.length) {
    return placeholder("No projects yet.");
  }
  const list = document.createElement("div");
  list.className = "stack";
  for (const project of projectsCache) {
    const card = document.createElement("article");
    card.className = "card";
    card.style.padding = "16px";
    const title = document.createElement("h3");
    title.textContent = project.name;
    title.style.margin = "0 0 6px 0";
    const desc = document.createElement("p");
    desc.className = "muted";
    desc.textContent = project.description || "No description";
    desc.style.margin = "0 0 10px 0";

    const actions = document.createElement("div");
    actions.className = "task-actions";

    const open = document.createElement("button");
    open.className = "btn btn-primary";
    open.textContent = "Open";
    open.addEventListener("click", () => {
      window.location.hash = `project/${project.id}`;
    });

    const renameInput = document.createElement("input");
    renameInput.type = "text";
    renameInput.value = project.name;
    renameInput.hidden = true;
    renameInput.ariaLabel = "Project name";

    const edit = document.createElement("button");
    edit.className = "btn";
    edit.textContent = "Rename";
    const saveRename = document.createElement("button");
    saveRename.className = "btn";
    saveRename.textContent = "Save Name";
    saveRename.hidden = true;

    edit.addEventListener("click", async () => {
      const enteringEdit = renameInput.hidden;
      renameInput.hidden = !renameInput.hidden;
      saveRename.hidden = !saveRename.hidden;
      edit.textContent = enteringEdit ? "Cancel" : "Rename";
      if (enteringEdit) {
        renameInput.focus();
        renameInput.select();
      }
    });

    saveRename.addEventListener("click", async () => {
      const nextName = renameInput.value.trim();
      if (!nextName) {
        showToast("Project name is required.", true);
        renameInput.focus();
        return;
      }
      startButtonPending(saveRename);
      startButtonPending(edit);
      try {
        await updateProject(project.id, {
          name: nextName,
          description: project.description || "",
        });
        showToast("Project updated.");
        await renderProjects(root);
      } catch (err) {
        showToast(err.message || "Could not update project", true);
      } finally {
        stopButtonPending(edit);
        stopButtonPending(saveRename);
      }
    });

    const del = document.createElement("button");
    del.className = "btn";
    del.textContent = "Delete";
    let confirmDelete = false;
    del.addEventListener("click", async () => {
      if (!confirmDelete) {
        confirmDelete = true;
        del.textContent = "Confirm Delete";
        setTimeout(() => {
          confirmDelete = false;
          del.textContent = "Delete";
        }, 3000);
        return;
      }
      startButtonPending(del);
      try {
        await deleteProject(project.id);
        showToast("Project deleted.");
        await renderProjects(root);
      } catch (err) {
        showToast(err.message || "Could not delete project", true);
      } finally {
        stopButtonPending(del);
      }
    });

    actions.appendChild(open);
    actions.appendChild(edit);
    actions.appendChild(saveRename);
    actions.appendChild(del);
    card.appendChild(title);
    card.appendChild(desc);
    card.appendChild(renameInput);
    card.appendChild(actions);
    list.appendChild(card);
  }
  return list;
}

async function quickSetDueDate(task, dayOffset, button, onChanged) {
  try {
    startButtonPending(button);
    const dueDate = localNoonISOString(dayOffset);
    await updateTask(task.id, { dueDate });
    if (onChanged) {
      await onChanged();
    }
  } catch (err) {
    showToast(err.message || "Could not update due date", true);
  } finally {
    stopButtonPending(button);
  }
}

function localNoonISOString(dayOffset) {
  const d = new Date();
  d.setDate(d.getDate() + dayOffset);
  d.setHours(12, 0, 0, 0);
  return d.toISOString();
}

function openTaskModal(task, onSaved) {
  bindModalEvents();
  modalOnSaved = onSaved || null;
  modalFeedback.textContent = "";
  fillProjectSelect(modalProject, true);
  modalTaskID.value = task.id || "";
  modalStatus.value = task.status || "inbox";
  modalPriority.value = task.priority || "none";
  modalProject.value = task.projectId || "";
  modalLocation.value = task.location || "";
  modalDueDate.value = toDateInputValue(task.dueDate || task.due_date || "");
  modalDescription.value = task.description || "";
  modalNotes.value = task.notes || "";
  modalContexts.value = Array.isArray(task.contexts || task.context)
    ? (task.contexts || task.context).join(", ")
    : "";
  modalTags.value = Array.isArray(task.tags) ? task.tags.join(", ") : "";
  modal.hidden = false;
}

function closeTaskModal() {
  modal.hidden = true;
}

function bindModalEvents() {
  if (modalBound) {
    return;
  }
  modalBound = true;

  modalCloseBtn.addEventListener("click", closeTaskModal);
  modalCancelBtn.addEventListener("click", closeTaskModal);

  modal.addEventListener("click", (event) => {
    if (event.target === modal) {
      closeTaskModal();
    }
  });

  document.addEventListener("keydown", (event) => {
    if (modal.hidden) {
      return;
    }
    if (event.key === "Escape") {
      closeTaskModal();
      return;
    }
    const saveCombo = (event.metaKey || event.ctrlKey) && event.key === "Enter";
    if (saveCombo) {
      event.preventDefault();
      modalForm.requestSubmit();
    }
  });

  modalForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    modalFeedback.textContent = "";
    const id = modalTaskID.value.trim();
    if (!id) {
      modalFeedback.textContent = "Missing task id.";
      return;
    }

    const contexts = splitCommaList(modalContexts.value);
    const tags = splitCommaList(modalTags.value);

    const payload = {
      status: modalStatus.value,
      priority: modalPriority.value,
      projectId: modalProject.value,
      location: modalLocation.value,
      description: modalDescription.value,
      notes: modalNotes.value,
      context: contexts,
      tags,
    };
    if (modalDueDate.value) {
      payload.dueDate = dateInputToNoonISOString(modalDueDate.value);
    } else {
      payload.clearDueDate = true;
    }

    const saveBtn = document.getElementById("task-modal-save");
    startButtonPending(saveBtn);
    try {
      await updateTask(id, payload);
      showToast("Task updated.");
      closeTaskModal();
      if (modalOnSaved) {
        await modalOnSaved();
      }
    } catch (err) {
      modalFeedback.textContent = err.message || "Could not save task details";
    } finally {
      stopButtonPending(saveBtn);
    }
  });
}

function splitCommaList(raw) {
  return raw
    .split(",")
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

function toDateInputValue(raw) {
  if (!raw) {
    return "";
  }
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) {
    return "";
  }
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function dateInputToNoonISOString(value) {
  const [y, m, d] = value.split("-").map(Number);
  const date = new Date();
  date.setFullYear(y, m - 1, d);
  date.setHours(12, 0, 0, 0);
  return date.toISOString();
}

function showToast(message, isError = false) {
  if (!toastRegion) {
    return;
  }
  const el = document.createElement("div");
  el.className = `toast${isError ? " is-error" : ""}`;
  el.textContent = message;
  toastRegion.appendChild(el);
  setTimeout(() => {
    el.remove();
  }, 2800);
}

function loadingIndicator(text) {
  const wrap = document.createElement("div");
  wrap.className = "loading-row";
  const spinner = document.createElement("span");
  spinner.className = "spinner";
  spinner.setAttribute("aria-hidden", "true");
  const label = document.createElement("span");
  label.textContent = text;
  wrap.appendChild(spinner);
  wrap.appendChild(label);
  return wrap;
}

function inlineSavingHint() {
  const hint = document.createElement("span");
  hint.className = "inline-saving";
  hint.textContent = "Saving...";
  hint.hidden = true;
  return hint;
}

function openInlineTitleEditor(titleEl, task, onChanged) {
  const original = task.title || "";
  const input = document.createElement("input");
  input.type = "text";
  input.value = original;
  input.style.margin = "0 0 8px 0";

  const commit = async () => {
    const nextTitle = input.value.trim();
    if (!nextTitle) {
      showToast("Title cannot be empty", true);
      input.focus();
      return;
    }
    if (nextTitle === original) {
      titleEl.textContent = original || "(untitled)";
      input.replaceWith(titleEl);
      return;
    }
    input.disabled = true;
    input.classList.add("is-pending");
    const priorPlaceholder = input.placeholder;
    input.placeholder = "Saving...";
    try {
      await updateTask(task.id, { title: nextTitle });
      showToast("Title updated.");
      if (onChanged) {
        await onChanged();
      } else {
        titleEl.textContent = nextTitle;
        input.replaceWith(titleEl);
      }
    } catch (err) {
      showToast(err.message || "Could not update title", true);
      input.disabled = false;
      input.classList.remove("is-pending");
      input.placeholder = priorPlaceholder;
      input.focus();
      return;
    }
    input.classList.remove("is-pending");
    input.placeholder = priorPlaceholder;
  };

  const cancel = () => {
    titleEl.textContent = original || "(untitled)";
    input.replaceWith(titleEl);
  };

  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      commit();
    } else if (event.key === "Escape") {
      event.preventDefault();
      cancel();
    }
  });

  input.addEventListener("blur", () => {
    commit();
  });

  titleEl.replaceWith(input);
  input.focus();
  input.select();
}

function startButtonPending(button) {
  if (!button) {
    return;
  }
  if (!button.dataset.idleLabel) {
    button.dataset.idleLabel = button.textContent || "";
  }
  button.disabled = true;
  button.classList.add("is-pending");
  button.textContent = PENDING_LABEL;
}

function stopButtonPending(button) {
  if (!button) {
    return;
  }
  button.disabled = false;
  button.classList.remove("is-pending");
  if (button.dataset.idleLabel) {
    button.textContent = button.dataset.idleLabel;
  }
}

