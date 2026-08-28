const state = { sessions: [], filter: "all", sel: -1, git: {} };

const $ = (id) => document.getElementById(id);

async function load() {
  const [sess, git] = await Promise.all([
    fetch("/api/sessions").then((r) => r.json()),
    fetch("/api/git").then((r) => r.json()).catch(() => ({})),
  ]);
  state.sessions = sess || [];
  state.git = git || {};
  $("repo").textContent = (git.repo || "") + (git.branch ? " · " + git.branch : "");
  renderList();
}

function filtered() {
  return state.sessions.filter((s) => {
    if (state.filter === "ai") return s.agent === "claude-code" || s.agent === "cursor";
    if (state.filter === "unreviewed") return s.status === "unseen";
    if (state.filter === "flagged") return s.status === "flagged";
    if (state.filter === "high") return (s.risks || []).some((r) => r === "blast-radius" || r === "secrets" || r === "tests-deleted");
    return true;
  });
}

function when(s) {
  const t = s.started_at || s.ended_at;
  if (!t) return "—";
  const d = new Date(t);
  if (Number.isNaN(d.getTime())) return t;
  return d.toLocaleString();
}

function renderList() {
  const list = filtered();
  const el = $("list");
  el.innerHTML = "";
  if (list.length === 0) {
    $("empty").style.display = "block";
    $("detail").innerHTML = '<div class="empty" id="empty">No agent sessions matched this repo.</div>';
    return;
  }
  list.forEach((s, i) => {
    const div = document.createElement("div");
    div.className = "row" + (i === state.sel ? " sel" : "");
    const risks = (s.risks || []).map((r) => `<span class="chip risk">${r}</span>`).join("");
    div.innerHTML = `<div class="meta"><span>${s.agent} · ${(s.files||[]).length} files</span><span>${s.status}</span></div>
      <div class="intent">${esc(s.intent || "(no prompt)")}</div>
      <div class="chips">${risks}<span class="chip">${when(s)}</span></div>`;
    div.onclick = () => { state.sel = i; renderList(); show(s); };
    el.appendChild(div);
  });
  if (state.sel < 0 || state.sel >= list.length) {
    state.sel = 0;
    show(list[0]);
    [...el.children].forEach((c, i) => c.classList.toggle("sel", i === 0));
  }
}

function esc(s) {
  return String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
}

async function show(s) {
  const box = $("detail");
  box.innerHTML = `<h2>${esc(s.intent || s.id)}</h2>
    <div class="kv">${esc(s.agent)} · ${esc(when(s))} · ${esc(s.branch || "")} · <b>${esc(s.status)}</b> · ${esc(s.id)}</div>
    <div class="act">
      <button class="accept" onclick="review('${s.id}','accepted')">accept (a)</button>
      <button class="flag" onclick="review('${s.id}','flagged')">flag (f)</button>
      <button onclick="review('${s.id}','seen')">seen (s)</button>
    </div>
    <h3>Why</h3><div class="why">${esc(s.why || "")}</div>
    <h3>Risk</h3><div class="chips">${(s.risks||[]).map(r=>`<span class="chip risk">${esc(r)}</span>`).join("") || '<span class="chip">none</span>'}</div>
    <h3>Files</h3><ul class="files">${(s.files||[]).map(f=>`<li>${esc(f.path)}${f.delete?" (deleted)":""}${f.prompt?` <span class="chip">${esc(f.prompt.slice(0,80))}</span>`:""}</li>`).join("")}</ul>
    <details class="diffbox" open><summary>Diff</summary><pre id="diff">loading…</pre></details>`;
  const d = await fetch("/api/diff?id=" + encodeURIComponent(s.id)).then((r) => r.json()).catch(() => ({}));
  const pre = document.getElementById("diff");
  if (pre) pre.innerHTML = colorDiff(d.diff || "(no git diff for these files)");
}

function colorDiff(text) {
  return esc(text).split("\n").map((ln) => {
    if (ln.startsWith("+") && !ln.startsWith("+++")) return '<span class="add">'+ln+"</span>";
    if (ln.startsWith("-") && !ln.startsWith("---")) return '<span class="sub">'+ln+"</span>";
    if (ln.startsWith("# prompt:")) return '<span class="chip warn">'+ln+"</span>";
    return ln;
  }).join("\n");
}

async function review(id, status) {
  await fetch("/api/review", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id, status }) });
  await load();
  const list = filtered();
  const i = list.findIndex((s) => s.id === id);
  if (i >= 0) { state.sel = i; show(list[i]); renderList(); }
}

document.getElementById("filters").onclick = (e) => {
  const b = e.target.closest("button");
  if (!b) return;
  state.filter = b.dataset.f;
  state.sel = 0;
  [...e.currentTarget.children].forEach((x) => x.classList.toggle("on", x === b));
  renderList();
};

document.addEventListener("keydown", (e) => {
  const list = filtered();
  if (!list.length) return;
  if (e.key === "j") { state.sel = Math.min(list.length - 1, state.sel + 1); renderList(); show(list[state.sel]); }
  if (e.key === "k") { state.sel = Math.max(0, state.sel - 1); renderList(); show(list[state.sel]); }
  if (e.key === "Enter") show(list[state.sel]);
  const cur = list[state.sel];
  if (!cur) return;
  if (e.key === "a") review(cur.id, "accepted");
  if (e.key === "f") review(cur.id, "flagged");
  if (e.key === "s") review(cur.id, "seen");
});

load();
setInterval(load, 2000);
