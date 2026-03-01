import { completeTask, createTask, fetchReview, fetchTasks, fetchToday, searchTasks, updateTask } from "./api.js";

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
const toastRegion = document.getElementById("toast-region");

let modalOnSaved = null;
let modalBound = false;
let taskSearchDebounce = null;
let tasksRenderBusy = false;
const PENDING_LABEL = "Saving...";
const tasksViewState = {
  q: "",
  status: "",
  priority: "",
  context: "",
};

export async function renderTasks(root) {
  if (tasksRenderBusy) {
    return;
  }
  tasksRenderBusy = true;
  root.innerHTML = "";
  root.appendChild(sectionTitle("Tasks"));
  root.appendChild(tasksControlPanel(root));
  root.appendChild(addTaskForm(async () => {
    await renderTasks(root);
  }));

  const loading = loadingIndicator("Loading tasks...");
  root.appendChild(loading);

  try {
    const tasks = await fetchTasksForView();
    loading.remove();
    root.appendChild(renderTaskList(tasks, async () => {
      await renderTasks(root);
    }, "No tasks match current filters."));
  } catch (err) {
    loading.remove();
    root.appendChild(placeholder("Could not load tasks."));
    showToast(err.message || "Could not load tasks", true);
  } finally {
    tasksRenderBusy = false;
  }
}

export async function renderInbox(root) {
  const tasks = await fetchTasks({ status: "inbox" });
  root.innerHTML = "";
  root.appendChild(sectionTitle("Inbox"));
  root.appendChild(renderTaskList(tasks, async () => {
    await renderInbox(root);
  }, "Inbox is clear."));
}

export async function renderToday(root) {
  const tasks = await fetchToday();
  root.innerHTML = "";
  root.appendChild(sectionTitle("Today"));
  root.appendChild(renderTaskList(tasks, async () => {
    await renderToday(root);
  }, "No tasks due today."));
}

export async function renderReview(root) {
  const data = await fetchReview();
  root.innerHTML = "";
  root.appendChild(sectionTitle("Review"));
  const pre = document.createElement("pre");
  pre.textContent = JSON.stringify(data, null, 2);
  pre.className = "muted";
  root.appendChild(pre);
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
  meta.textContent = `${task.status || "unknown"} • ${task.priority || "none"}${task.dueDate ? ` • due ${task.dueDate}` : ""}`;

  const actions = document.createElement("div");
  actions.className = "task-actions";

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
  completeBtn.textContent = task.status === "done" ? "Completed" : "Mark Done";
  completeBtn.disabled = task.status === "done";
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

  card.appendChild(title);
  card.appendChild(meta);
  card.appendChild(actions);
  return card;
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

function tasksControlPanel(root) {
  const panel = document.createElement("section");
  panel.className = "card task-controls";

  const grid = document.createElement("div");
  grid.className = "task-controls-grid";

  const search = document.createElement("input");
  search.type = "search";
  search.placeholder = "Search title/description/notes";
  search.value = tasksViewState.q;
  search.addEventListener("input", () => {
    tasksViewState.q = search.value.trim();
    if (taskSearchDebounce) {
      clearTimeout(taskSearchDebounce);
    }
    taskSearchDebounce = setTimeout(() => {
      renderTasks(root).catch((err) => showToast(err.message || "Search failed", true));
    }, 220);
  });

  const status = document.createElement("select");
  status.innerHTML = `
    <option value="">Status: all</option>
    <option value="inbox">inbox</option>
    <option value="actionable">actionable</option>
    <option value="waiting">waiting</option>
    <option value="someday">someday</option>
    <option value="done">done</option>
  `;
  status.value = tasksViewState.status;
  status.addEventListener("change", () => {
    tasksViewState.status = status.value;
    renderTasks(root).catch((err) => showToast(err.message || "Filter failed", true));
  });

  const priority = document.createElement("select");
  priority.innerHTML = `
    <option value="">Priority: all</option>
    <option value="none">none</option>
    <option value="low">low</option>
    <option value="medium">medium</option>
    <option value="high">high</option>
  `;
  priority.value = tasksViewState.priority;
  priority.addEventListener("change", () => {
    tasksViewState.priority = priority.value;
    renderTasks(root).catch((err) => showToast(err.message || "Filter failed", true));
  });

  const context = document.createElement("input");
  context.type = "text";
  context.placeholder = "Context filter";
  context.value = tasksViewState.context;
  context.addEventListener("change", () => {
    tasksViewState.context = context.value.trim();
    renderTasks(root).catch((err) => showToast(err.message || "Filter failed", true));
  });

  const clearBtn = document.createElement("button");
  clearBtn.className = "btn";
  clearBtn.type = "button";
  clearBtn.textContent = "Clear Filters";
  clearBtn.addEventListener("click", () => {
    tasksViewState.q = "";
    tasksViewState.status = "";
    tasksViewState.priority = "";
    tasksViewState.context = "";
    renderTasks(root).catch((err) => showToast(err.message || "Could not reset filters", true));
  });

  grid.appendChild(search);
  grid.appendChild(status);
  grid.appendChild(priority);
  grid.appendChild(context);
  panel.appendChild(grid);
  panel.appendChild(clearBtn);
  return panel;
}

async function fetchTasksForView() {
  const { q, status, priority, context } = tasksViewState;
  if (!q) {
    return fetchTasks(compact({
      status,
      priority,
      context,
    }));
  }

  const searched = await searchTasks(q);
  return searched.filter((task) => {
    if (status && task.status !== status) {
      return false;
    }
    if (priority && task.priority !== priority) {
      return false;
    }
    if (context) {
      const contexts = Array.isArray(task.contexts || task.context) ? (task.contexts || task.context) : [];
      if (!contexts.includes(context)) {
        return false;
      }
    }
    return true;
  });
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

function addTaskForm(onAdded) {
  const form = document.createElement("form");
  form.className = "stack card";
  form.style.padding = "16px";
  form.style.marginBottom = "12px";

  const heading = document.createElement("h3");
  heading.textContent = "Quick Add";
  heading.style.margin = "0";

  const title = document.createElement("input");
  title.type = "text";
  title.placeholder = "Task title";
  title.required = true;

  const context = document.createElement("input");
  context.type = "text";
  context.placeholder = "Context (optional)";

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

  row.appendChild(priority);
  row.appendChild(status);

  const submit = document.createElement("button");
  submit.type = "submit";
  submit.className = "btn btn-primary";
  submit.textContent = "Add Task";

  const feedback = document.createElement("p");
  feedback.className = "muted";
  feedback.style.margin = "0";
  feedback.style.minHeight = "1rem";

  form.appendChild(heading);
  form.appendChild(title);
  form.appendChild(context);
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
        priority: priority.value,
        status: status.value,
      });
      title.value = "";
      context.value = "";
      priority.value = "none";
      status.value = "inbox";
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
  modalTaskID.value = task.id || "";
  modalStatus.value = task.status || "inbox";
  modalPriority.value = task.priority || "none";
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

