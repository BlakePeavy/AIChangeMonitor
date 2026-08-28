# AI Change Monitor

Git is the ledger. The review unit is an agent session joined to the dirty working tree
(what / why / when / who) **now**, not after commit.

```
go build -o aichange .
# or: go install github.com/BlakePeavy/AIChangeMonitor@latest
```

Requires **Go 1.22+**. Windows Go 1.19 cannot build this module; upgrade the toolchain
and build from source. A prebuilt `.exe` is not the install method.

Same source builds on Windows, macOS, and Linux with `CGO_ENABLED=0` (pure Go SQLite).

Cross-compile release artifacts if you want them:

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/aichange.exe .
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -o dist/aichange-darwin-arm64 .
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o dist/aichange-linux-amd64 .
```

`GOOS` may be `windows|darwin|linux`. `GOARCH` may be `amd64|arm64`.

## 30-second usage

From any git checkout (or pass `--repo`):

```
./aichange
# same as: ./aichange serve
```

Opens the inbox at http://127.0.0.1:7380. Then:

```
aichange sessions
aichange sessions --json
aichange show [id]
aichange diff [id]
aichange why internal/store/store.go
aichange review <id> accept
aichange review <id> flag
aichange review <id> seen
```

Inbox filters: All / AI / Unreviewed / Flagged / High risk.
Keys: `j`/`k` move, `enter` open, `a` accept, `f` flag, `s` seen.

Empty state: **No agent sessions matched this repo.**

Status lives only in the local index. Zero writes to the monitored repo.

## Where things live

| What | Path |
| --- | --- |
| Index (Linux/mac) | `$XDG_STATE_HOME/aichange/index.db` or `~/.local/state/aichange/index.db` |
| Index (Windows) | `%LOCALAPPDATA%\aichange\index.db` |
| Override | `AICHANGE_INDEX` |
| Claude Code | `~/.claude/projects/<encoded-cwd>/*.jsonl` (nested `sessions/` ok). Windows: `%USERPROFILE%\.claude\projects\`. Honors `CLAUDE_CONFIG_DIR`. |
| Cursor | `~/.cursor/projects/<slug>/agent-transcripts/*.jsonl` (nested ok). Windows: `%USERPROFILE%\.cursor\projects\`. Best-effort JSONL only. |

Encoded cwd = every non-alphanumeric rune becomes `-`.
Example: `/Users/you/code/my-app` → `-Users-you-code-my-app`.

Claude/Cursor project folders match when the encoded repo root equals the folder name.

## Privacy

Transcripts contain source and secrets. Treat them as secret-bearing.

- Never upload transcripts or the index.
- Never write the monitored repo (no Git Notes, no hooks, no `.why/`).
- AWS keys, PEM blocks, `.env` assignments, and high-entropy tokens are redacted before store or print.
- Offline. No network. No LLM. No telemetry.
- SQLite via `modernc.org/sqlite` only (pure Go, `CGO_ENABLED=0`).
- Shells out to `git`. No libgit2.

If a provider directory is missing, that provider is skipped.

## Non-goals

Git Notes, hooks, `.why/` commits, daemons, `state.vscdb`, protobuf, Windsurf, Copilot `workspaceStorage`, Aider, an LLM summarizer, line-level blame.

See [DESIGN.md](DESIGN.md).

## License

MIT
