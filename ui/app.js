const state = {
  sessions: [],
  filter: "all",
  selId: null,
  shownId: null,
  git: {},
  listSig: "",
  open: null,
  editingRepo: false,
  restoreAllowed: false,
  restoreReason: "",
  ctxPath: null,
};

const $ = (id) => document.getElementById(id);

const DANGER = new Set(["secrets", "tests-deleted", "blast-radius", "auth"]);
const RECENT_KEY = "aichange.recentRepos";
const RECENT_MAX = 8;

const CHIP_TIP = {
  "secrets": "Touched .env, keys, credentials, or other secret-looking files.",
  "auth": "Touched login, token, session, or permission code.",
  "blast-radius": "Large change: 15+ files or 800+ lines. Review the whole blast, not one file.",
  "tests-deleted": "Test files were removed. Coverage may have dropped.",
  "lockfile": "A package lockfile changed along with several other files.",
  "deletes": "Files were deleted in this change.",
  "live": "Uncommitted work in the working tree right now.",
  "commit": "From git history. Why is the commit body, if it has one.",
  "transcript": "From an agent session log (Cursor, Claude, …).",
  "cursor": "Looks like a Cursor agent (from commit trailers or a transcript).",
  "claude-code": "Looks like Claude Code (from commit trailers or a transcript).",
  "copilot": "Looks like GitHub Copilot (from commit trailers).",
  "aider": "Looks like Aider (from commit trailers).",
  "codex": "Looks like Codex (from commit trailers).",
  "windsurf": "Looks like Windsurf (from commit trailers).",
  "git": "A git commit. No agent trailers found.",
  "unknown-agent": "Commit trailers suggest an agent, but which one is unclear.",
};

function chip(label, cls) {
  const tip = CHIP_TIP[label] || "";
  const t = tip ? ` title="${esc(tip)}"` : "";
  return `<span class="chip ${cls || ""}"${t}>${esc(label)}</span>`;
}

async function load() {
  const [sess, git] = await Promise.all([
    fetch("/api/sessions").then((r) => r.json()),
    fetch("/api/git").then((r) => r.json()).catch(() => ({})),
  ]);
  state.sessions = sess || [];
  state.git = git || {};
  paintRepo(git);
  renderList();
}

function paintRepo(git) {
  if (state.editingRepo) return;
  const path = (git && git.repo) || "";
  $("repo-path").textContent = path || "scanning…";
  $("repo-branch").textContent = git && git.branch ? " · " + git.branch : "";
  if (path) rememberPath(path);
}

function recentPaths() {
  try {
    const raw = JSON.parse(localStorage.getItem(RECENT_KEY) || "[]");
    return Array.isArray(raw) ? raw.filter((p) => typeof p === "string" && p.trim()) : [];
  } catch (e) {
    return [];
  }
}

function rememberPath(path) {
  path = String(path || "").trim();
  if (!path) return;
  const next = [path, ...recentPaths().filter((p) => p !== path)].slice(0, RECENT_MAX);
  localStorage.setItem(RECENT_KEY, JSON.stringify(next));
}

function showRepoErr(msg) {
  const el = $("repo-err");
  if (!msg) {
    el.hidden = true;
    el.textContent = "";
    return;
  }
  el.hidden = false;
  el.textContent = msg;
}

function startRepoEdit() {
  const input = $("repo-input");
  const path = state.git.repo || "";
  state.editingRepo = true;
  $("repo-path").hidden = true;
  input.hidden = false;
  input.value = path;
  input.focus();
  input.select();
  renderRecent();
}

function endRepoEdit() {
  state.editingRepo = false;
  $("repo-input").hidden = true;
  $("repo-recent").hidden = true;
  $("repo-path").hidden = false;
  paintRepo(state.git);
}

function renderRecent() {
  const box = $("repo-recent");
  const all = recentPaths();
  if (!all.length) {
    box.hidden = true;
    box.innerHTML = "";
    return;
  }
  box.innerHTML = "";
  all.forEach((p) => {
    const b = document.createElement("button");
    b.type = "button";
    b.textContent = p;
    b.title = p;
    b.onmousedown = (e) => {
      e.preventDefault();
      submitRepo(p);
    };
    box.appendChild(b);
  });
  box.hidden = false;
}

async function submitRepo(path) {
  path = String(path || "").trim();
  if (!path) return;
  const r = await fetch("/api/repo", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });
  if (!r.ok) {
    const msg = (await r.text()).trim() || "not a git repo";
    showRepoErr(msg);
    return;
  }
  const git = await r.json();
  rememberPath(git.repo || path);
  showRepoErr("");
  state.git = git || {};
  state.selId = null;
  state.shownId = null;
  state.listSig = "";
  state.open = null;
  endRepoEdit();
  await load();
}

function isDanger(s) {
  return (s.risks || []).some((r) => DANGER.has(r));
}

function rowClass(s) {
  const bits = ["row"];
  if (s.id === state.selId) bits.push("sel");
  if (s.status === "accepted") bits.push("accepted");
  else if (s.status === "flagged") bits.push("flagged");
  else if (isDanger(s)) bits.push("danger");
  return bits.join(" ");
}

function filtered() {
  return state.sessions.filter((s) => {
    if (state.filter === "unreviewed") return s.status === "unseen";
    if (state.filter === "danger") return isDanger(s);
    return true;
  });
}

function sessionFingerprint(s) {
  return JSON.stringify({
    id: s.id,
    status: s.status,
    files: (s.files || []).length,
    added: s.added_lines || 0,
    deleted: s.deleted_lines || 0,
    intent: s.intent || "",
    risks: s.risks || [],
  });
}

function listFingerprint(list) {
  return list.map(sessionFingerprint).join("\n");
}

function titleOf(s) {
  const t = s && s.intent && String(s.intent).trim();
  return t || "Untitled change";
}

function whenParts(s) {
  const t = s.started_at || s.ended_at;
  if (!t) return { text: "—", title: "" };
  const d = new Date(t);
  if (Number.isNaN(d.getTime())) return { text: String(t), title: String(t) };
  return { text: relTime(d), title: d.toLocaleString() };
}

function relTime(d, now) {
  now = now || new Date();
  const sec = Math.round((now.getTime() - d.getTime()) / 1000);
  const min = Math.round(sec / 60);
  if (Math.abs(min) < 60) return Math.max(1, Math.abs(min)) + "m ago";
  const hr = Math.round(sec / 3600);
  if (Math.abs(hr) < 24) return Math.max(1, Math.abs(hr)) + "h ago";
  const start = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const days = Math.round((start(now) - start(d)) / 86400000);
  if (days === 1) return "yesterday";
  if (days >= 2 && days < 7) {
    return d.toLocaleDateString(undefined, { weekday: "short" });
  }
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function sourceChip(s) {
  const src = s.source || "";
  if (src === "live") return { label: "live", cls: "src-live" };
  if (src === "commit") {
    if (s.agent && s.agent !== "git") return { label: s.agent, cls: "src-agent" };
    return { label: "commit", cls: "src-commit" };
  }
  return { label: s.agent || "transcript", cls: "src-agent" };
}

function pm(n) {
  return (n > 0 ? n : 0);
}

function dangerChips(s) {
  return (s.risks || []).filter((r) => DANGER.has(r))
    .map((r) => chip(r, "danger"))
    .join("");
}

function renderList() {
  const list = filtered();
  const el = $("list");
  const sig = listFingerprint(list);

  if (list.length === 0) {
    el.innerHTML = "";
    state.listSig = sig;
    state.shownId = null;
    const msg = state.filter === "all"
      ? "Working tree is clean and no recent commits."
      : "No sessions match this filter.";
    $("detail").innerHTML = `<div class="empty">${msg}</div>`;
    return;
  }

  if (!list.some((s) => s.id === state.selId)) {
    state.selId = list[0].id;
  }

  if (sig === state.listSig && el.childElementCount === list.length) {
    [...el.children].forEach((c) => c.classList.toggle("sel", c.dataset.id === state.selId));
  } else {
    const scroll = el.scrollTop;
    el.innerHTML = "";
    list.forEach((s) => {
      const div = document.createElement("div");
      const danger = isDanger(s);
      div.className = rowClass(s);
      div.dataset.id = s.id;
      const src = sourceChip(s);
      const n = (s.files || []).length;
      const plus = pm(s.added_lines);
      const minus = pm(s.deleted_lines);
      const w = whenParts(s);
      div.innerHTML = `<div class="intent">${esc(titleOf(s))}</div>
        <div class="meta"><span>${chip(src.label, src.cls)} · ${n} files · +${plus}/−${minus} · <span title="${esc(w.title)}">${esc(w.text)}</span></span><span class="row-risks">${dangerChips(s)}</span></div>`;
      div.onclick = () => { state.selId = s.id; renderList(); };
      el.appendChild(div);
    });
    el.scrollTop = scroll;
    state.listSig = sig;
  }

  if (state.shownId !== state.selId) {
    const s = list.find((x) => x.id === state.selId);
    if (s) show(s);
  }
}

function esc(s) {
  return String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
}

function groupFolders(files) {
  const order = [];
  const by = {};
  for (const f of files || []) {
    const p = String(f.path || "").replace(/\\/g, "/").replace(/^\.\//, "");
    if (!p) continue;
    const slash = p.indexOf("/");
    const name = slash <= 0 ? "." : p.slice(0, slash);
    if (!by[name]) {
      by[name] = { name, count: 0, added: 0, deleted: 0 };
      order.push(name);
    }
    by[name].count++;
    by[name].added += f.added || 0;
    by[name].deleted += f.deleted || 0;
  }
  return order.map((n) => by[n]);
}

function whereBar(groups) {
  if (!groups.length) return "";
  let total = groups.reduce((s, g) => s + Math.abs(g.added) + Math.abs(g.deleted), 0);
  if (total === 0) total = groups.reduce((s, g) => s + g.count, 0) || 1;
  const segs = groups.map((g) => {
    const w = Math.abs(g.added) + Math.abs(g.deleted);
    const share = w > 0 ? w : g.count;
    const pct = Math.max(2, (share / total) * 100);
    return `<span style="width:${pct}%" title="${esc(g.name)}"></span>`;
  }).join("");
  return `<div class="where-bar">${segs}</div>`;
}

function riskSection(s) {
  const risks = s.risks || [];
  if (!risks.length) return "";
  return `<h3>Risk</h3><div class="chips">${risks.map((r) =>
    chip(r, DANGER.has(r) ? "danger" : "risk")
  ).join("")}</div>`;
}

function slide() {
  return `<div class="slide"><div class="slide-inner"><pre class="patch"></pre></div></div>`;
}

function show(s) {
  state.shownId = s.id;
  state.open = null;
  hideFileMenu();
  const live = (s.source === "live") || String(s.id || "").startsWith("live:");
  state.restoreAllowed = live;
  state.restoreReason = live ? "" : "restore is only for uncommitted or unpushed commit files";
  const shown = s.id;
  fetch("/api/session?id=" + encodeURIComponent(s.id)).then((r) => r.json()).then((full) => {
    if (state.shownId !== shown) return;
    state.restoreAllowed = !!full.restore_allowed;
    state.restoreReason = full.restore_reason || "";
  }).catch(() => {});
  const box = $("detail");
  const groups = groupFolders(s.files || []);
  const where = groups.map((g) =>
    `<div class="where-row" data-kind="folder" data-key="${esc(g.name)}"><div class="where-head"><b>${esc(g.name)}</b> · ${g.count} files · +${g.added}/−${g.deleted}</div>${slide()}</div>`
  ).join("");
  const why = (s.why && String(s.why).trim())
    ? `<h3>Why</h3><div class="why">${esc(s.why)}</div>`
    : "";
  const w = whenParts(s);
  const files = (s.files || []).map((f) => {
    const extra = `${f.delete ? " (deleted)" : ""}${f.added || f.deleted ? ` <span class="chip">+${f.added || 0}/−${f.deleted || 0}</span>` : ""}${f.prompt ? ` <span class="chip">${esc(f.prompt.slice(0, 80))}</span>` : ""}`;
    return `<li class="file-row" data-kind="file" data-key="${esc(f.path)}"><div class="file-head">${esc(f.path)}${extra}</div>${slide()}</li>`;
  }).join("");
  box.innerHTML = `<h2>${esc(titleOf(s))}</h2>
    <div class="kv">${esc(s.agent)} · ${esc(s.source || "")} · <span title="${esc(w.title)}">${esc(w.text)}</span> · ${esc(s.branch || "")} · <b>${esc(s.status)}</b></div>
    ${why}
    ${riskSection(s)}
    <h3>Where</h3>
    <div class="where">${where || '<span class="chip">no files</span>'}</div>
    ${whereBar(groups)}
    <h3>Files</h3><ul class="files">${files}</ul>
    <div class="act secondary">
      <button class="accept" onclick="review('${s.id}','accepted')">accept (a)</button>
      <button class="flag" onclick="review('${s.id}','flagged')">flag (f)</button>
      <button onclick="review('${s.id}','seen')">seen (s)</button>
    </div>`;
}

function colorDiff(text) {
  return esc(text).split("\n").map((ln) => {
    if (ln.startsWith("+") && !ln.startsWith("+++")) return '<span class="add">'+ln+"</span>";
    if (ln.startsWith("-") && !ln.startsWith("---")) return '<span class="sub">'+ln+"</span>";
    if (ln.startsWith("# prompt:")) return '<span class="chip warn">'+ln+"</span>";
    return ln;
  }).join("\n");
}

function closeOpenPatch() {
  const box = $("detail");
  if (!box) return;
  box.querySelectorAll(".file-row.open, .where-row.open").forEach((el) => el.classList.remove("open"));
}

async function togglePatch(row) {
  const kind = row.dataset.kind;
  const key = row.dataset.key;
  const sessId = state.selId;
  const same = state.open && state.open.kind === kind && state.open.key === key;
  closeOpenPatch();
  if (same) {
    state.open = null;
    return;
  }
  state.open = { kind, key };
  row.classList.add("open");
  const pre = row.querySelector("pre.patch");
  if (!pre) return;
  if (pre.dataset.loaded === "1") return;
  pre.textContent = "loading…";
  const q = "/api/diff?id=" + encodeURIComponent(sessId)
    + (kind === "file" ? "&file=" : "&folder=") + encodeURIComponent(key);
  const d = await fetch(q).then((r) => r.json()).catch(() => ({}));
  if (!(state.open && state.open.kind === kind && state.open.key === key && state.selId === sessId)) return;
  pre.innerHTML = colorDiff(d.diff || "(no git diff for these files)");
  pre.dataset.loaded = "1";
}

async function review(id, status) {
  await fetch("/api/review", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id, status }) });
  state.selId = id;
  state.shownId = null;
  state.listSig = "";
  await load();
}

document.getElementById("filters").onclick = (e) => {
  const b = e.target.closest("button");
  if (!b) return;
  state.filter = b.dataset.f;
  state.listSig = "";
  [...e.currentTarget.children].forEach((x) => x.classList.toggle("on", x === b));
  renderList();
};

$("detail").addEventListener("click", (e) => {
  if (e.target.closest("button")) return;
  const head = e.target.closest(".file-head, .where-head");
  if (!head) return;
  const row = head.closest("[data-kind][data-key]");
  if (!row || !$("detail").contains(row)) return;
  togglePatch(row);
});

$("repo-path").addEventListener("click", startRepoEdit);
$("repo-input").addEventListener("keydown", (e) => {
  if (e.key === "Enter") {
    e.preventDefault();
    e.stopPropagation();
    submitRepo($("repo-input").value);
  }
  if (e.key === "Escape") {
    e.preventDefault();
    e.stopPropagation();
    showRepoErr("");
    endRepoEdit();
  }
});
$("repo-input").addEventListener("focus", renderRecent);

document.addEventListener("keydown", (e) => {
  if (e.target.closest("input, textarea")) return;
  const list = filtered();
  if (!list.length) return;
  const i = Math.max(0, list.findIndex((s) => s.id === state.selId));
  if (e.key === "j") {
    state.selId = list[Math.min(list.length - 1, i + 1)].id;
    renderList();
  }
  if (e.key === "k") {
    state.selId = list[Math.max(0, i - 1)].id;
    renderList();
  }
  if (e.key === "Enter") {
    state.shownId = null;
    renderList();
  }
  const cur = list.find((s) => s.id === state.selId);
  if (!cur) return;
  if (e.key === "a") review(cur.id, "accepted");
  if (e.key === "f") review(cur.id, "flagged");
  if (e.key === "s") review(cur.id, "seen");
});

function toast(msg, err) {
  const el = $("toast");
  if (!el) return;
  el.hidden = !msg;
  el.textContent = msg || "";
  el.classList.toggle("err", !!err);
  clearTimeout(toast._t);
  if (msg) toast._t = setTimeout(() => { el.hidden = true; }, 4200);
}

function hideFileMenu() {
  const m = $("file-menu");
  if (!m) return;
  m.hidden = true;
  $("ctx-confirm").hidden = true;
  $("ctx-items").hidden = false;
}

function placeMenu(x, y) {
  const m = $("file-menu");
  m.hidden = false;
  const pad = 6;
  const w = m.offsetWidth || 188;
  const h = m.offsetHeight || 90;
  const nx = Math.min(x, window.innerWidth - w - pad);
  const ny = Math.min(y, window.innerHeight - h - pad);
  m.style.left = Math.max(pad, nx) + "px";
  m.style.top = Math.max(pad, ny) + "px";
}

function openFileMenu(path, x, y) {
  state.ctxPath = path;
  const restore = $("ctx-restore");
  restore.disabled = !state.restoreAllowed;
  restore.title = state.restoreAllowed ? "Discard this file's local change" : (state.restoreReason || "restore not allowed");
  $("ctx-confirm").hidden = true;
  $("ctx-items").hidden = false;
  placeMenu(x, y);
}

async function copyPath(path) {
  path = String(path || "");
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(path);
    } else {
      const ta = document.createElement("textarea");
      ta.value = path;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      ta.remove();
    }
    toast("copied " + path);
  } catch (e) {
    toast("copy failed", true);
  }
}

async function revealPath(path) {
  const r = await fetch("/api/reveal", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  }).catch(() => null);
  if (r && r.ok) {
    toast("revealed " + path);
    return;
  }
  await copyPath(path);
}

async function doRestore(path) {
  const id = state.selId;
  toast("restoring " + path + "…");
  const r = await fetch("/api/restore", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id, path }),
  }).catch(() => null);
  hideFileMenu();
  let data = {};
  try { data = r ? await r.json() : {}; } catch (e) { data = {}; }
  if (!r || !r.ok) {
    toast((data && data.error) || "restore failed", true);
    return;
  }
  toast("restored " + (data.path || path));
  state.shownId = null;
  state.listSig = "";
  await load();
}

$("detail").addEventListener("contextmenu", (e) => {
  const row = e.target.closest(".file-row");
  if (!row || !$("detail").contains(row)) return;
  e.preventDefault();
  const path = row.dataset.key;
  if (!path) return;
  openFileMenu(path, e.clientX, e.clientY);
});

$("file-menu").addEventListener("contextmenu", (e) => {
  e.preventDefault();
});

$("ctx-restore").addEventListener("click", (e) => {
  e.stopPropagation();
  if ($("ctx-restore").disabled) return;
  $("ctx-items").hidden = true;
  $("ctx-confirm").hidden = false;
});
$("ctx-copy").addEventListener("click", (e) => {
  e.stopPropagation();
  hideFileMenu();
  copyPath(state.ctxPath);
});
$("ctx-reveal").addEventListener("click", (e) => {
  e.stopPropagation();
  hideFileMenu();
  revealPath(state.ctxPath);
});
$("ctx-ok").addEventListener("click", (e) => {
  e.stopPropagation();
  doRestore(state.ctxPath);
});
$("ctx-cancel").addEventListener("click", (e) => {
  e.stopPropagation();
  hideFileMenu();
});

document.addEventListener("click", (e) => {
  const m = $("file-menu");
  if (!m || m.hidden) return;
  if (m.contains(e.target)) return;
  hideFileMenu();
});
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") hideFileMenu();
});

load();
setInterval(load, 2000);
