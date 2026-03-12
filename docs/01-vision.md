# 01 — Vision

## Elevator Pitch

**Loom** is a versioning system built for 2026 and beyond. It replaces the manual commit-branch-merge ceremony of Git with continuous, intelligent, multi-space versioning. Code, docs, design, data — everything gets versioned in one unified timeline. Auto-snapshots like Google Docs. No merge conflicts. LLM-native from day one. Client-server architecture for remote collaboration.

Built with Go. Open source. MIT license.

## Why Not Git

Git was designed in 2005 for the Linux kernel. It solved version control for text-based source code managed by humans through manual commits. Twenty years later, the world looks different:

- Projects are more than code. Docs, design files, data schemas, configurations, notes, AI conversations — all need versioning.
- AI agents are writing and modifying code. They need a versioning API, not a CLI designed for humans typing in terminals.
- Merge conflicts are the number one productivity killer in collaborative development. They're a 2005 solution to a 2005 problem.
- Manual commits create gaps in history. The time between commits is a black hole — work is lost, context disappears.
- Branch management is ceremony. Most branching workflows exist to work around Git's limitations, not because they add value.

Loom is a ground-up reinvention of version control. It doesn't borrow Git's model, vocabulary, or limitations. Git didn't clone SVN — Loom doesn't clone Git.

## Design Pillars

1. **Continuous** — Versioning happens automatically, like Google Docs. No manual commits required. Every meaningful change is captured.
2. **Intelligent** — No merge conflicts. Concurrent changes converge automatically via CRDTs (v2) or are resolved by LLM agents. Conflicts become suggestions, not blockers.
3. **Universal** — One system for code, docs, design, data, configs, notes, and anything else. Space adapters normalize different content types into a shared model.
4. **Agent-First** — LLM agents are first-class citizens. Structured API for versioning, diffing, rolling back, and explaining changes. Agents can version their own work.
5. **Collaborative** — Client-server architecture. Send and receive streams to remotes. Real-time sync in the future.
6. **Local-First** — All data lives locally. Remotes are optional. Works offline. Your data is yours.

## How It's Different

| Concept | Git | Loom |
|---------|-----|------|
| Unit of change | Commit (manual) | Operation (automatic) |
| History model | Snapshot DAG | Append-only operation log |
| Branching | Branches with merge ceremony | Streams that auto-converge |
| Bookmarks | Commits are the only waypoints | Checkpoints (auto + manual, named) |
| Merge | Three-way with conflict markers | CRDT convergence + LLM resolution |
| Content types | Text files only | Any content via space adapters |
| Scope | Single repository | Multi-space project (code + docs + design + ...) |
| Audience | Humans typing in terminals | Humans + AI agents + automation |
| Diff | Line-based text diff | Semantic diff per content type |
| Remote | git push/pull/fetch | Stream sync (send/receive operations) |
| Auto-save | None | Continuous auto-checkpointing |

## Core Concepts

### Operations

An operation is the atomic unit of change. Not a snapshot of the whole tree — just what changed.

```
Operation: modify
  Space: code
  Entity: src/auth/login.go
  Delta: [line 42: changed "password" to "passphrase"]
  Author: flakerimi
  Timestamp: 2026-03-11T10:15:30.000Z
```

Operations are append-only. Once written, they never change. State is derived by replaying operations.

### Streams

A stream is a live, auto-versioning timeline. Think of it as a branch that versions itself continuously.

- `main` — the primary stream
- `feature/auth` — a working stream
- Streams can be forked and merged
- Multiple streams can auto-converge (no conflicts)

### Checkpoints

A checkpoint is a named point on a stream. Like a Git commit, but optional — the history exists with or without checkpoints.

- **Auto checkpoints** — created automatically (every N operations, on significant changes, before risky actions)
- **Manual checkpoints** — created by the user (`loom checkpoint "before refactor"`)
- **Agent checkpoints** — created by AI agents before/after their work

### Spaces

A space is a content domain with its own adapter. Each space knows how to track, diff, and restore its content type.

Built-in spaces:
- `code` — source code (backed by Git when available)
- `docs` — documentation and markdown files
- `design` — design files and structured UI data
- `data` — schemas, migrations, datasets
- `config` — configuration files
- `notes` — freeform notes and journals

Custom spaces can be registered via adapters.

### Hubs

A hub is a remote Loom server that stores and syncs streams. Add a hub and send/receive operations.

```
loom hub add origin https://loomhub.dev/flakerimi/my-app
loom send
loom receive
```

## Product Shape

From the user's perspective:

1. **Initialize** — `loom init` in any project directory. Loom detects existing spaces (Git repos, doc folders, design files) and starts tracking.
2. **Work** — Just work. Loom auto-versions in the background. No commit ceremony.
3. **Checkpoint** — Optionally name a point: `loom checkpoint "auth system complete"`.
4. **Browse** — `loom log` shows the timeline. `loom diff` shows what changed. Filter by space, author, time.
5. **Restore** — `loom restore <checkpoint>` restores any point. By space, by entity, or the whole project.
6. **Collaborate** — `loom send` / `loom receive` to sync with hubs. No merge conflicts.
7. **Agent** — AI agents call the Loom API directly. `loom.checkpoint("before refactor")`, `loom.rollback(id)`.

## What Loom Is

- **A versioning system** — replaces Git with operation-based, multi-space versioning
- **A sync engine** — replaces Google Drive-style sync with automatic, continuous collaboration
- **A timeline system** — every change is recorded, undo/redo at any granularity
- **An embeddable SDK** — integrate Loom inside any app for built-in versioning and history

## What Loom Is Not

- Not a real-time collaboration tool (v1 — that's v2+ with CRDTs)
- Not a backup system (though it can serve as one)
- Not a deployment tool
- Not a CI/CD system

## Target Users

1. **Developers** who want continuous versioning beyond just code
2. **AI agents** that need structured version control APIs
3. **Design-dev teams** who want unified history across code and design
4. **Solo creators** who want Google Docs-style auto-save for their entire project
5. **Teams** who are tired of merge conflicts

## Open Source Strategy

- MIT license
- Go binary — single binary, no runtime dependencies
- Cross-platform (macOS, Linux, Windows)
- Embeddable as a library (Go package) or standalone CLI
- Server component for remote collaboration
- Construct integration as a bundled space (but Loom works independently)
