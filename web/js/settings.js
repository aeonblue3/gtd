import { apiFetch } from "./api.js";

export async function renderSettings(root) {
  root.innerHTML = "";
  const title = document.createElement("h2");
  title.className = "section-title";
  title.textContent = "Settings";
  root.appendChild(title);

  const status = await apiFetch("/status");
  const notifyConfig = await apiFetch("/notifications/config");

  root.appendChild(block("Server Status", status));
  root.appendChild(block("Notification Config", notifyConfig));

  const runNowBtn = document.createElement("button");
  runNowBtn.className = "btn btn-primary";
  runNowBtn.textContent = "Run Notifications Now";
  runNowBtn.addEventListener("click", async () => {
    runNowBtn.disabled = true;
    try {
      const res = await apiFetch("/notifications/run-now", { method: "POST" });
      alert(`Run complete: ${JSON.stringify(res.stats)}`);
    } catch (err) {
      alert(err.message);
    } finally {
      runNowBtn.disabled = false;
    }
  });
  root.appendChild(runNowBtn);
}

function block(title, data) {
  const wrap = document.createElement("section");
  wrap.className = "card";
  wrap.style.padding = "16px";
  wrap.style.marginBottom = "12px";

  const h = document.createElement("h3");
  h.textContent = title;
  h.style.margin = "0 0 8px 0";

  const pre = document.createElement("pre");
  pre.className = "muted";
  pre.style.margin = "0";
  pre.textContent = JSON.stringify(data, null, 2);

  wrap.appendChild(h);
  wrap.appendChild(pre);
  return wrap;
}

