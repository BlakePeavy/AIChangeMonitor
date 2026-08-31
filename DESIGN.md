# Design

## Thesis

Git is the ledger. The review unit is an agent *session* joined to the dirty working tree.

Seniors reviewing AI-touched code need what changed, why the agent did it, when the session ran, and who the actor was — *now*, while the tree is still dirty. Waiting for a commit loses the join: the prompt, the thinking, the file list, and the working-tree diff scatter across tools.

`aichange` occupies that hole. git-ai and AgentNote need agent hooks for line authorship. vibe-replay browses sessions and never looks at git. git-why writes `.why/` at commit (Cursor adapter unshipped). We join session logs to the working tree to why, locally, with no repo pollution.

Git stays the source of truth for history. This tool is a reviewer overlay. The default feed is git log (recent commits) plus the dirty working tree as a live session — first run in any repo is never empty. Transcripts, when present, upgrade why (prompt/thinking only; never invented). Show `git show` for a commit, worktree+cached diff for live, and keep accept/flag/seen in *our* index.

## How a session is built

1. Upsert recent `git log --numstat` commits as `git:<sha>` and one `live:<fingerprint>` session from porcelain + staged/unstaged numstat. Then discover Claude Code and Cursor JSONL next to the user home (never inside the repo).
2. Match workspace by encoded path == repo root (or session cwd under that root).
3. First human user prompt becomes intent (Cursor user_query unwrapped). Redacted.
4. Assistant text and thinking before the first Edit/Write becomes why (3-8 lines). Redacted.
5. File list from tool_use Edit / Write / NotebookEdit (Cursor aliases included). Bash rm / redirects when the path is obvious.
6. Risk chips: secrets/env/auth paths, lockfile + many files, deletes, blast radius (15+ files or 800+ lines), tests deleted.
7. Related git: git status, git diff -- files, git log --since=start -- files.

Status (unseen|seen|accepted|flagged) lives only in the local index. Re-parsing a transcript updates content and preserves status.

## Non-goals

Do not replace git. Do not write notes, hooks, or why-folders into the monitored repo. No whole-tree watcher. No vendor DB decode. No extra agent vendors. No LLM, network, telemetry, or line blame.

## Stack

Pure Go SQLite, embedded HTML UI, shell out to git, offline.
