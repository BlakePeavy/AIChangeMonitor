# AI Change Monitor

Git is the ledger. This is a local inbox for **reviewing what an AI agent just changed** — not a git GUI, and not a chat log browser.

A *session* is the unit of review: a dirty working tree, a recent commit, or (when present) a Claude Code / Cursor transcript joined to those files. The point is a 60-second keep / dig in / revert call: **what** moved, **why** if we actually have it, **when**, and whether it looks dangerous.

It never writes the repo you are watching. No Git Notes, no hooks, no `.why/` files. Status (unseen / seen / accepted / flagged) lives only in a local index. Nothing is uploaded. There is no LLM in this tool.

## Install

Needs **Go 1.22+**. SQLite is pure Go (`modernc.org/sqlite`); leave `CGO_ENABLED=0`.

```bash
git clone https://github.com/BlakePeavy/AIChangeMonitor.git
cd AIChangeMonitor
go build -o aichange .
```

Windows PowerShell:

```powershell
go build -o aichange.exe .
```

Or, without a clone:

```bash
go install github.com/BlakePeavy/AIChangeMonitor@latest
```

`make dist` cross-compiles Windows / macOS / Linux binaries into `dist/` (gitignored).

## Usage

From any git checkout:

```bash
./aichange                  # Linux / macOS — scans, then serves the UI
.\aichange.exe              # Windows (PowerShell needs the .\ )
```

```
aichange listening on http://127.0.0.1:7380
```

Open that URL, or skip the browser and use the [desktop window](#desktop-optional).

| Command | What it does |
| --- | --- |
| `aichange` / `aichange serve` | Scan + local web UI on `:7380` |
| `aichange serve --repo PATH` | Watch a repo you are not currently in |
| `aichange serve --addr :9000` | Pick a port |
| `aichange sessions` | List sessions |
| `aichange sessions --json` | Same, JSON |
| `aichange show <id>` | One session |
| `aichange diff <id>` | Patch |
| `aichange why path/to/file` | Why for a path, if a transcript has it |
| `aichange review <id> accept` | `accept` / `flag` / `seen` |

The UI header path is clickable: paste another folder, Enter, it rescan. **File → Open repo** does the same in the desktop app.

### Inbox

- Left: sessions. Live dirty tree, recent commits, transcripts when they match this repo.
- Click a file for an inline diff. Right-click a file to restore (uncommitted or unpushed only), copy the path, or reveal it.
- Filters: All / Unreviewed / Danger.
- Keys: `j` `k` move, `enter` open, `a` accept (green), `f` flag (amber), `s` seen.
- Danger chips (`secrets`, `auth`, `blast-radius`, `tests-deleted`) have hover explanations. We never invent a *why* — if there is no transcript, why is empty.
- Theme follows the OS (warm paper in light, a dimmer room in dark).

Empty inbox means the working tree is clean and there are no recent commits. That is a successful first run only if git itself is empty.

## Desktop (optional)

`desktop/` is a Tauri 2 window around the **same** Go engine. It does not rewrite the UI. The CLI still works without it.

Needs Go 1.22+, Rust (rustup), Node.js (for the Tauri CLI), and a webview (WebView2 on Windows — already on current Win10/11).

```bash
cd desktop
npm install
# builds the Go sidecar, then the window
npm run tauri dev      # development
npm run tauri build    # installer + exe / app / deb
```

Windows PowerShell, from `desktop\`:

```powershell
npm install
npm run tauri dev
```

First launch: **File → Open repo** and pick a git folder. Later launches reuse it.

Build the native window **on the OS you want to ship**. The Go engine cross-compiles; Tauri does not (no Linux to Windows window).

### Windows linker notes

The default Rust host is MSVC. That needs the Visual Studio **Desktop C++** workload so `msvcrt.lib` exists. If `npm run tauri dev` dies with `LNK1104 cannot open file 'msvcrt.lib'`, either install that workload, or build with the GNU/MinGW target (`x86_64-pc-windows-gnu`) and put Git + MinGW on `PATH` ahead of stray `link.exe` copies (Cygwin / VS Insiders). Do not pin a GNU `rust-toolchain.toml` in this repo — it breaks everyone else.

## Where data lives

| What | Linux / macOS | Windows |
| --- | --- | --- |
| Index | `$XDG_STATE_HOME/aichange/index.db` or `~/.local/state/aichange/index.db` | `%LOCALAPPDATA%\aichange\index.db` |
| Override | `AICHANGE_INDEX` | `AICHANGE_INDEX` |
| Claude Code logs (read-only) | `~/.claude/projects/<encoded-cwd>/` | `%USERPROFILE%\.claude\projects\` |
| Cursor logs (read-only) | `~/.cursor/projects/<slug>/agent-transcripts/` | `%USERPROFILE%\.cursor\projects\` |

Encoded cwd = absolute path with every non-alphanumeric character turned into `-`.

Zero writes into the monitored repo. Transcripts are treated as secret-bearing: AWS keys, PEM blocks, `.env` assignments, and high-entropy tokens are redacted on ingest.

## Non-goals

Git Notes, hooks, `.why/` commits, whole-tree daemons, vendor DB decode (`state.vscdb`, Windsurf `.pb`), Copilot `workspaceStorage`, an LLM summarizer, line-level blame.

See [DESIGN.md](DESIGN.md) for how a session is built. Desktop wiring is in [DESKTOP.md](DESKTOP.md).

## License

MIT
