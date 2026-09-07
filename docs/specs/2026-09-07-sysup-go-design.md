---
title: "sysup-go design"
description: "Port of the Fish sysup orchestrator: config vs programs, run loop, context, cache."
keywords: [design, spec, programs, config, runner, context, sudo, mise, cache, skip, continue]
order: 10
---

# sysup-go design

Port of the Fish `sysup` family to a Go CLI. Fish kept a registry plus
one file per step. Go cannot source `.go` at runtime, so the registry
becomes user TOML under XDG. Compiled code owns wraps (sudo keepalive,
mise, end-of-run cache) and the runner. Programs are exec calls only.

This spec is the source of truth for v1. Implementation plans and
`AGENTS.md` follow it.

## Goals

1. Run an ordered list of user-defined programs without a rebuild.
2. Fail fast on bad config before touching the system.
3. Keep sudo alive across long package updates.
4. Teach Go via real use: files, `context`, goroutines, interfaces,
   errors.

## Non-goals (v1)

- Distro support beyond Arch-family package managers (pacman + paru/yay).
  Unknown distros error after resolve fails.
- YAML or JSON config. Commands as shell strings.
- A `kind` field, reserved program names, or filename magic that
  changes behaviour. Files under `programs/` (or `programs.toml`) are
  execution calls. That is the only meaning they have.
- Parallel package updates. `parallel = true` is opt-in on programs
  that do not share state (neovim / ya / hermes), never mirror/pacman/aur.

## Layout

Root `main.go` stays the process entry. Do not move it under `cmd/`.
That layout is for multi-binary repos. This is one binary.

```
sysup-go/
  main.go                 # package main; calls cmd.Execute()
  cmd/                    # cobra: flags, wiring, exit codes
    root.go               # default action = run the plan
    list.go               # cobra-cli add list
    validate.go           # cobra-cli add validate
  internal/
    config/               # config.toml via viper (TOML)
    program/              # types, discovery, sort, decode
    resolve/              # PKG_MANAGER + LookPath probes
    runner/               # plan, waves, continue, wraps
    sudo/                 # keepalive goroutine
    mise/                 # deactivate; defer reactivate
    cache/                # end-of-run dir sweep
  docs/
    INDEX.md
    specs/
```

`cmd/` parses flags and prints. Logic lives in `internal/`. A
`*slog.Logger` is built in `cmd` and injected. Text handler on stderr.

Viper loads only `config.toml`. Program files are decoded with the TOML
library directly. Viper is a bad directory merger.

Add cobra commands with `cobra-cli add <name>`. Do not generate a cobra
command per program; that would require a rebuild.

## Two files, two jobs

`$XDG_CONFIG_HOME/sysup-go/` (fallback `~/.config/sysup-go/`).

| Path | Job |
| --- | --- |
| `config.toml` | Our behaviour: sudo, mise, cache, shutdown, pkg_manager last resort |
| `programs.toml` XOR `programs/*.toml` | What to run, and in what order |

If `programs.toml` exists, use it (array order). Else use `programs/*.toml`.
If neither exists, error: nothing to run.

Do not merge both. Do not invent a third registry.

## config.toml

```toml
[sudo]
keepalive = true
interval = "50s"

[mise]
wrap = true

[cache]
enabled = true
include_dirs = []
exclude_dirs = []

[pkg_manager]
# last resort only; see resolve

[shutdown]
force = false
wait = "1m"
```

`[cache] enabled` means: after every program in the plan has finished
(success, skip, or accumulated error), sweep cache directories. It is
not a program. Users who want a mid-chain cache command write a normal
program file at that index. We will not stop them. We also will not
treat that file as special.

Cache directories: start from a built-in set (pacman pkg cache, paru
clone/diff under `XDG_CACHE_HOME` / `~/.cache/paru`, and any others we
confirm in code). Merge `include_dirs`. Subtract `exclude_dirs`
(exclude can drop an internal path). Sweep what remains. Verify the
directory is gone or empty after the sweep; log and record a failure if
not.

v1 may start by exec-ing the packager's own clean with stdin fed in
process (not `yes | tr`). The dir sweep is the behaviour we actually
want. Piping confirmations is a known Fish bug; do not rebuild it as
the real path.

`[mise] wrap`: if `mise` is not on `PATH`, no-op. If it is, deactivate
before the plan and `defer` reactivate. Run `mise up` only when the
plan had zero failures.

## Program files

Directory order is **lexical filename sort**. Convention is systemd-style
tens with gaps: `10-mirror.toml`, `20-pacman.toml`, `30-aur.toml`.
Not `1-`, not `01-`. We do not parse the number. `9-` vs `10-` is the
user's problem; docs say use tens.

Single-file `programs.toml` uses `[[program]]` array order.

```toml
# programs/20-pacman.toml
name = "pacman"
alias = "p"
description = "Update official repository packages"
optional = false
parallel = false
command = ["sudo", "pacman", "-Syyu", "--needed", "--noconfirm"]
```

| Field | Rules |
| --- | --- |
| `name` | Required. Unique across the set. Collision errors cite both paths. |
| `alias` | Optional short skip token (`p`, `m`). Unique. Collision errors cite both paths. |
| `description` | Help / list text. |
| `optional` | If true, missing argv0 skips the program (not a failure). If false, missing argv0 is a validate/run error. |
| `parallel` | Default false. See waves. |
| `command` | Required non-empty argv. `command[0]` is the binary. No shell. |

There is no `kind`, no `skip` letter derived from the first character of
`name`, no reserved `name`. `--skip` matches `name` or `alias`.

## Skip vs continue

These are different flags. They are not in conflict.

```
sysup-go -c              # continue on program errors
sysup-go -s c            # skip the program whose alias is "c"
sysup-go -s m,c          # skip aliases m and c (StringSlice)
sysup-go --skip pacman   # skip by name
```

`-c` / `--continue` is behaviour. `-s` / `--skip` is a slice of program
identifiers. Cobra `StringSliceP`: repeated flags and comma-separated
values both work.

Unknown skip token: error, name the token (exit 3). Skipping our cache
phase is `[cache] enabled = false` or `--no-cache`, not `--skip cache`.
Cache is not a program.

## Interfaces

Go has no inheritance. Interface embedding is composition.

```go
type Runner interface {
    Run(ctx context.Context) error
}

type PreRunner interface {
    PreRun(ctx context.Context) error
}

type PostRunner interface {
    PostRun(ctx context.Context) error
}

type Program interface {
    Runner
    PreRunner
    PostRunner
}
```

The loop iterates `Runner`. Optional hooks are type assertions, not
no-op methods on exec specs:

```go
if p, ok := step.(PreRunner); ok {
    if err := p.PreRun(ctx); err != nil { /* ... */ }
}
if err := step.Run(ctx); err != nil { /* ... */ }
if p, ok := step.(PostRunner); ok {
    if err := p.PostRun(ctx); err != nil { /* ... */ }
}
```

A type that implements all three **is** a `Program`. An exec spec that
only has `Run` is a `Runner`. Do not invent dummy `PreRun`/`PostRun` so
everything can sit in a `[]Program`.

User programs are `Runner`s (`exec.CommandContext`). Mise/sudo/cache are
runner wraps around the whole plan, not programs.

## Resolve: PKG_MANAGER

1. Env `PKG_MANAGER` **unset**: fall through.
2. Env **set but empty** (`PKG_MANAGER=` or `""`): error immediately.
3. Env set and non-empty: that binary. If it is not on `PATH`, error.
   No fallthrough.
4. Else probe `paru` and `yay` in parallel (`LookPath` in goroutines).
   Both present: paru wins.
5. Else `config.pkg_manager` if set and present on `PATH`.
6. Else error (Nix, Debian, empty PATH, etc.).

## Run loop

1. Load + validate (unique names/aliases, non-empty commands, known
   fields, skip tokens resolve).
2. `signal.NotifyContext` for SIGINT/SIGTERM. That `ctx` is the root.
   Every child process is `exec.CommandContext(ctx, ...)`.
3. If sudo keepalive enabled: `sudo -v`, then a goroutine with a ticker
   (`sudo -n true`) that `select`s on `ctx.Done()`.
4. If mise wrap and `mise` exists: deactivate. `defer` reactivate
   (and `mise up` only on a clean plan). `defer` is function-scoped
   LIFO. It is not GC.
5. Build the plan (registry order minus `--skip`).
6. Group into waves (below). Run waves in order.
7. If `[cache] enabled` and not `--no-cache`: sweep dirs.
8. If `--force-shutdown` (or config `shutdown.force`): shut down after
   the wait, even with failures.
9. Else if `--shutdown` (`-d`) and **zero** failures: shut down after
   the wait.
10. Else do not shut down. Print accumulated errors if any.

Default is hard fail: first program error aborts remaining programs
(cache still runs if enabled; shutdown does not, unless force).

`-c` / `--continue`: keep going, collect errors, print them at the end.
Exit non-zero if any were collected. Shutdown still requires zero
failures unless `--force-shutdown`.

`--force-shutdown` is long-only (plus the config key). It is intentional.
`--force-shutdown` implies shutdown; it does not require `-d` as well.

## Waves (`parallel`)

Sort first (filename or array order). Then walk:

- `parallel = false` (default): its own wave of one. Barrier.
- Consecutive `parallel = true` entries form one wave and run together
  (`errgroup` + the wave's derived context).

Example:

```
10-mirror.toml      parallel = false
20-pacman.toml      parallel = false
30-aur.toml         parallel = false
50-rustup.toml      parallel = false
60-neovim.toml      parallel = true
70-ya.toml          parallel = true
80-hermes.toml      parallel = true
```

Waves: mirror, pacman, aur, rustup, then {neovim, ya, hermes}.

Hard fail in a parallel wave: cancel that wave's context so siblings
stop, then abort the plan. Continue: wait for the whole wave, collect
errors, proceed to the next wave.

Do not overlap rustup with pacman/aur. Leave rustup `parallel = false`
and ordered after them.

## CLI

Root command runs the plan (Fish parity: `sysup-go` with no subcommand).

| Flag | Meaning |
| --- | --- |
| `--config` | Override config path |
| `-s, --skip` | Program names and/or aliases (StringSlice) |
| `-c, --continue` | Accumulate program errors |
| `--no-cache` | Skip our end-of-run cache sweep |
| `-d, --shutdown` | Shutdown after a clean run |
| `--force-shutdown` | Shutdown even with failures |
| `--log-level` | slog level |

Subcommands (`cobra-cli add`):

- `list` -- discovered programs in run order
- `validate` -- parse and resolve, do not run

## Errors and exit codes

Each `internal/` package owns a typed error with `Unwrap()`
(`config.Error`, `program.Error`, `resolve.Error`, `runner.StepError`).
`cmd` maps them to process exit codes. Callers use `errors.Is` /
`errors.As`. Wrap with `fmt.Errorf("...: %w", err)`.

| Code | Meaning |
| --- | --- |
| 0 | ok |
| 2 | bad flags |
| 3 | unknown skip token |
| 5 | program failed (hard fail, or continue with failures) |
| 1 | everything else (missing programs dir, resolve, config) |

`log/slog` only. No zap. Inject the logger; do not rely on the global
except as a last-resort default in `main`.

## Context (normative)

See the implementation notes in `AGENTS.md` and the walkthrough in
conversation; the rules here are binding:

- Root `ctx` comes from `signal.NotifyContext`. `defer stop()`.
- Pass `ctx` as the first argument to Run/PreRun/PostRun, resolve,
  cache, mise, sudo.
- Derive per-wave contexts with `context.WithCancel` (hard fail) or
  inherit the parent (continue).
- Never store `Context` on a long-lived struct. Pass it in.
- `ctx.Done()` is how Ctrl-C kills children, the sudo ticker, and a
  hard-fail wave.

## Tests (v1)

Table tests, no network:

- Discovery: file XOR dir, lexical sort, missing both errors.
- Unique name/alias collisions cite paths.
- Skip by name and alias; unknown token errors.
- PKG_MANAGER: unset / empty / set-missing / paru-over-yay.
- Wave grouping from `parallel` flags.
- Continue vs hard fail vs shutdown vs force-shutdown.

Fake `Runner`s, not live pacman.

## Later

- Debian/apt as another `PkgManager` implementation.
- User-tunable sudo interval already exists; extra keepalive knobs.
- Cache dir list will grow once we confirm what pacman/paru actually
  delete.
