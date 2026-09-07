---
title: "Goal prompt"
description: "Paste-ready work order for a milestone. Agents read this plus the named spec IDs only."
keywords: [goal, prompt, subagents, parallel]
order: 19
---

# Goal prompt

Copy the block that matches the slice. Do not paste chat history.

## Standing rules (every goal)

- Colocated `jj` repo: `jj` only, never raw `git`.
- Read `docs/INDEX.md`, then `AGENTS.md`, then **only** the spec
  files named in the goal. Do not load the Fish sources or older
  design debate.
- Design doc wins on behaviour conflicts:
  `docs/specs/2026-09-07-sysup-go-design.md`.
- Types are locked in `docs/specs/impl/00-overview.md`. Do not rename.
- Programs on disk are exec calls. No `kind`, no reserved names, no
  argv env expansion.
- Tests: table-driven, `t.TempDir` / injected `LookPath`. No live
  pacman/sudo/shutdown.
- ASCII in source and docs.
- `gofmt` + `go test ./...` for packages you touch.
- Commit each milestone with `jj commit` conventional commits when
  that milestone is green, unless the user said otherwise.

## Parallelism

Go packages under `internal/<name>/` are isolated. Independent
packages in the same slice run as **parallel sub-agents**, one
package per agent, `isolation: none` (same worktree, disjoint
paths). The parent agent owns `cmd/` wiring after children return.

Do **not** parallelise a package with its `Needs` column still
unimplemented. `cmd/` is never a child's job if two children would
edit `cmd/root.go`.

| Slice | Parallel children | Then parent |
| --- | --- | --- |
| A (01-05) | 01 `internal/config`, 02 `internal/program` (load/spec only), 03 `internal/resolve` | 04 `cmd/` then 05 `internal/runner` + `program/exec.go` + `cmd` RunE |
| B (06) | none | `internal/sudo` + `cmd` hook |
| C (07-08) | none (08 needs 07's `RunAll` shape) | runner continue, then waves |
| D (09-11) | 09 `internal/mise`, 10 `internal/cache`, 11 `internal/shutdown` (packages only) | `cmd` compose wraps |

## Slice A -- paste this

```
Goal: implement sysup-go milestones 01-05.

Read, in order:
  docs/INDEX.md
  AGENTS.md
  docs/specs/impl/00-overview.md
  docs/specs/impl/01-config.md
  docs/specs/impl/02-discovery.md
  docs/specs/impl/03-resolve.md
  docs/specs/impl/04-cli.md
  docs/specs/impl/05-runner.md

Do not implement 06-11.

Work:
1. Spawn three parallel sub-agents (isolation none, disjoint paths):
   - 01: internal/config only
   - 02: internal/program spec+load+error+tests only (no exec.go)
   - 03: internal/resolve only
2. When all three are green (`go test` on each package), YOU implement
   04 (cobra-cli add list/validate, flags, exitCode) then 05
   (Exec + RunAll + signal.NotifyContext + root RunE).
3. go test ./... and go build .
4. jj commit per milestone if green (01, 02, 03 may be one commit
   if they land together; 04 and 05 separate commits).

Locks:
- config.toml optional (defaults); programs required.
- TOML, not YAML. Drop cobra-cli yaml default.
- --skip is names/aliases; -c is continue (unused until 07, still add the flag).
- No $PKG_MANAGER expansion in command arrays.
- Exec implements Runner only. No no-op Pre/Post.
- Root main.go stays at repo root.
```

## Later slices -- paste and fill ID

```
Goal: implement sysup-go milestone NN.

Read:
  docs/INDEX.md
  AGENTS.md
  docs/specs/impl/00-overview.md
  docs/specs/impl/NN-*.md

Do not implement other IDs.

Follow the spec's Files / Tests / Done when.
Use a sub-agent only if the spec's package is disjoint from cmd;
you keep cmd wiring.

Locks from 00-overview still apply.
Milestone 09: State.DidStrip + PathOrig. Restore is a stored-string
write, not a second PATH parse. mise up only if DidStrip && clean.
```
