import {
  fetchNotificationConfig,
  fetchStatus,
  runNotificationsNow,
  updateNotificationConfig,
} from "./api.js";

export async function renderSettings(root) {
  root.innerHTML = "";
  root.appendChild(sectionTitle("Settings"));

  const [status, notifyConfig] = await Promise.all([
    fetchStatus(),
    fetchNotificationConfig(),
  ]);

  root.appendChild(renderStatusCard(status));
  root.appendChild(renderNotificationForm(notifyConfig, root));
}

function renderStatusCard(status) {
  const wrap = document.createElement("section");
  wrap.className = "card";
  wrap.style.padding = "16px";

  const h = document.createElement("h3");
  h.textContent = "Runtime Status";
  h.style.margin = "0 0 8px 0";

  const grid = document.createElement("div");
  grid.className = "summary-grid";

  const runtime = status.runtime || {};
  const auth = status.auth || {};
  const notifications = status.notifications || {};
  const tiles = [
    ["Service", status.service || "gtd-api"],
    ["API", status.api && status.api.version ? status.api.version : "v1"],
    ["Uptime (s)", runtime.uptime_seconds ?? "?"],
    ["API Key Auth", auth.api_key ? "enabled" : "disabled"],
    ["Password+TOTP", auth.password_totp ? "enabled" : "disabled"],
    ["Scheduler", notifications.scheduler_active ? "active" : "inactive"],
  ];

  for (const [label, value] of tiles) {
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
    grid.appendChild(tile);
  }

  wrap.appendChild(h);
  wrap.appendChild(grid);
  return wrap;
}

function renderNotificationForm(cfg, root) {
  const card = document.createElement("section");
  card.className = "card";
  card.style.padding = "16px";
  card.style.marginTop = "12px";

  const title = document.createElement("h3");
  title.textContent = "Notification Config";
  title.style.margin = "0 0 8px 0";

  const form = document.createElement("form");
  form.className = "stack";

  const enabled = checkboxInput("Enabled", cfg.enabled);
  const dryRun = checkboxInput("Dry Run", cfg.dry_run);
  const checkSeconds = textInput("Check Seconds", `${cfg.check_seconds || 300}`, "number");
  const emailTo = textInput("Email To", cfg.email_to || "");
  const emailFrom = textInput("Email From", cfg.email_from || "");
  const smtpHost = textInput("SMTP Host", cfg.smtp_host || "");
  const smtpPort = textInput("SMTP Port", `${cfg.smtp_port || 587}`, "number");
  const smtpUser = textInput("SMTP Username", cfg.smtp_username || "");
  const smtpPass = textInput("SMTP Password", "", "password");
  smtpPass.input.placeholder = cfg.has_smtp_password ? "stored (leave blank to keep)" : "optional";

  form.appendChild(enabled.wrap);
  form.appendChild(dryRun.wrap);
  form.appendChild(checkSeconds.wrap);
  form.appendChild(emailTo.wrap);
  form.appendChild(emailFrom.wrap);
  form.appendChild(smtpHost.wrap);
  form.appendChild(smtpPort.wrap);
  form.appendChild(smtpUser.wrap);
  form.appendChild(smtpPass.wrap);

  const actions = document.createElement("div");
  actions.className = "task-actions";
  const save = document.createElement("button");
  save.type = "submit";
  save.className = "btn btn-primary";
  save.textContent = "Save Config";
  const runNow = document.createElement("button");
  runNow.type = "button";
  runNow.className = "btn";
  runNow.textContent = "Run Notifications Now";
  actions.appendChild(save);
  actions.appendChild(runNow);
  form.appendChild(actions);

  const feedback = document.createElement("p");
  feedback.className = "muted";
  feedback.style.margin = "0";
  feedback.textContent = cfg.restart_required ? "Restart required for scheduler changes." : "";
  form.appendChild(feedback);

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    save.disabled = true;
    save.textContent = "Saving...";
    feedback.textContent = "";
    try {
      const updated = await updateNotificationConfig({
        enabled: enabled.input.checked,
        dry_run: dryRun.input.checked,
        check_seconds: Number.parseInt(checkSeconds.input.value, 10) || 300,
        email_to: emailTo.input.value.trim(),
        email_from: emailFrom.input.value.trim(),
        smtp_host: smtpHost.input.value.trim(),
        smtp_port: Number.parseInt(smtpPort.input.value, 10) || 587,
        smtp_username: smtpUser.input.value.trim(),
        smtp_password: smtpPass.input.value,
      });
      feedback.textContent = updated.restart_required
        ? "Saved. Restart required for scheduler changes."
        : "Saved.";
      await renderSettings(root);
    } catch (err) {
      feedback.textContent = err.message || "Could not save notification config";
    } finally {
      save.disabled = false;
      save.textContent = "Save Config";
    }
  });

  runNow.addEventListener("click", async () => {
    runNow.disabled = true;
    runNow.textContent = "Saving...";
    feedback.textContent = "";
    try {
      const result = await runNotificationsNow();
      const stats = result && result.stats ? result.stats : {};
      feedback.textContent = `Run complete. Scanned ${stats.scanned || 0}, sent ${stats.sent || 0}, failed ${stats.failed || 0}.`;
    } catch (err) {
      feedback.textContent = err.message || "Could not run notifications";
    } finally {
      runNow.disabled = false;
      runNow.textContent = "Run Notifications Now";
    }
  });

  card.appendChild(title);
  card.appendChild(form);
  return card;
}

function sectionTitle(text) {
  const h = document.createElement("h2");
  h.className = "section-title";
  h.textContent = text;
  return h;
}

function checkboxInput(label, checked) {
  const wrap = document.createElement("label");
  const input = document.createElement("input");
  input.type = "checkbox";
  input.checked = !!checked;
  wrap.appendChild(document.createTextNode(label));
  wrap.appendChild(input);
  return { wrap, input };
}

function textInput(label, value, type = "text") {
  const wrap = document.createElement("label");
  wrap.textContent = label;
  const input = document.createElement("input");
  input.type = type;
  input.value = value;
  wrap.appendChild(input);
  return { wrap, input };
}

