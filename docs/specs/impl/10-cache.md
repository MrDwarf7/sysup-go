---
title: "10 cache"
description: "After the plan, sweep known cache dirs plus include_dirs minus exclude_dirs."
keywords: [impl, cache, directories, include, exclude]
order: 30
---

# 10 cache

End-of-run only. Not a program. `--skip` cannot target it.
`--no-cache` or `[cache] enabled = false` skips the sweep.

Runs after `RunAll` returns, including on hard fail and on
continue-with-errors. Does not run if ctx is already cancelled
(Ctrl-C): skip the sweep, log info.

## Depends on

01 (`Config.Cache`), 05 (RunE). Resolve (03) is optional; dir sweep
does not need paru if we know the paths.

## Files

- Create: `internal/cache/dirs.go` (built-in list, merge, exclude)
- Create: `internal/cache/sweep.go`
- Create: `internal/cache/error.go`
- Create: `internal/cache/sweep_test.go`
- Modify: `cmd/root.go` after `RunAll`

## Built-in dirs (v1)

Compute at sweep time, do not hardcode a single user's home:

| ID | Path |
| --- | --- |
| pacman | `/var/cache/pacman/pkg` |
| paru-clone | `$XDG_CACHE_HOME/paru/clone` or `~/.cache/paru/clone` |
| paru-diff | `$XDG_CACHE_HOME/paru/diff` or `~/.cache/paru/diff` |

If paru cache roots do not exist, skip them (not an error). Missing
pacman pkg dir: skip, log debug (might be running unprivileged in
tests).

`include_dirs`: append, cleaned with `filepath.Abs` / `Clean`.
Duplicates dropped.

`exclude_dirs`: remove matching cleaned paths from the set (can drop
a built-in).

Empty set after merge: log info, return nil.

## Sweep

For each remaining path:

1. If not exist: continue.
2. Remove **contents** of package caches, not necessarily the
   directory inode if it is a system dir we should keep.
   Lock: delete children of the directory (`os.RemoveAll` on each
   entry) so the dir itself remains. Do not `RemoveAll` on
   `/var/cache/pacman/pkg` in a way that removes the pkg dir if
   pacman expects it; removing contents is enough.
3. After: directory exists and is empty, **or** directory is gone
   (include_dirs user paths may be fully removed). Else error that
   path.

Privilege: pacman cache likely needs sudo. Inject a `Remove` func:

```go
type Remover func(ctx context.Context, path string) error
```

Default: `os.RemoveAll` on children. If EACCES on pacman pkg, retry
via `sudo rm -rf -- <child>` **or** skip with warn. v1 lock: try
unprivileged remove; on EACCES log warn and record a cache error
(does not invent `yes | pacman -Scc`).

Do not pipe confirmations to pacman. Dir sweep is the real path.

A cache sweep error does **not** un-fail a clean program plan: it is
an additional failure (exit 5). It also does not block 11's
force-shutdown decision except that it counts as a failure (so
`-d` without force will not shut down).

## Tests

Temp dirs only. Never touch `/var/cache`.

| Case | Expect |
| --- | --- |
| include one temp dir with files | files gone, dir empty |
| exclude that same dir | left untouched |
| exclude of a built-in by abs path | not in the work list |
| merge include + builtin, drop dups | unique list |
| missing path | skipped |
| `--no-cache` / enabled false | Remover not called |

`dirs()` / `plan(cfg, env)` is unit-tested with fake home/xdg.

## Out of scope

`yes | tr`. Treating a program named `cache` as this sweep.

## Done when

`go test ./internal/cache/` green. Manual: include_dirs pointing at
a throwaway directory actually empties it after `sysup-go`.
