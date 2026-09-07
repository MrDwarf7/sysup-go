# AGENTS.md

Local working notes for this repo. Tracked. Keep it short; the design
spec is the behaviour source of truth.

## Project

`sysup-go` is a Go CLI that ports the Fish `sysup` orchestrator.
User programs live in XDG TOML and run without a rebuild. Compiled
code owns sudo keepalive, mise wrap, and end-of-run cache.

## Docs

Never dump `docs/` into context.

1. Read `docs/INDEX.md`.
2. Open only the file the task needs (usually the design spec).
3. When you add a doc, add a row to `docs/INDEX.md`.

Design: `docs/specs/2026-09-07-sysup-go-design.md`.
Impl: `docs/specs/impl/00-overview.md` then the numbered milestone
spec the goal call names. Do not implement a later ID first.

## VCS

This is a colocated `jj` repo (`.jj/` present). Use `jj`, not raw `git`.
Raw `git commit` / `checkout` / `reset` corrupts the jj graph.

- No staging area. Edits are in `@`.
- `jj commit -m "type(scope): summary"` to snapshot and start a new change.
- Conventional Commits, lowercase, no emoji, no AI trailer.
- Do not commit or push unless asked (except when the user asked).

## Layout

- `main.go` at repo root. Do not move it under `cmd/`.
- `cmd/` is cobra only (flags, help, exit codes). Add commands with
  `cobra-cli add <name>`.
- `internal/` is the program. Other modules cannot import it.
- One package per directory. Package name = directory name, short,
  lowercase, no underscores, no stutter (`config.Load` not
  `config.LoadConfig` unless the extra word is required).

## Hard rules (this project)

- Programs on disk are execution calls. No `kind` field, no reserved
  names, no filename magic that changes behaviour.
- Order: lexical filenames (`10-`, `20-`, `30-`) or `[[program]]` array
  order. We do not parse the number.
- Viper loads `config.toml` only. Decode program files with TOML
  directly.
- Config format is TOML. Not YAML. Commands are argv arrays, not shell
  strings.
- `-c` is continue. `-s c` skips the program aliased `c`. Different.
- Cache runs after the whole plan. `[cache] include_dirs` /
  `exclude_dirs` merge with / subtract from our built-in list.
- Inject `*slog.Logger`. `log/slog` only.
- Typed errors per `internal/` package, wrapped with `%w`. `cmd` maps
  to exit codes.
- ASCII in source and docs. No em-dash, no box-drawing, no unicode
  arrows. `->` and `--` are fine.
- No no-op `PreRun`/`PostRun` on exec specs. Type-assert optional
  interfaces.

## Go (official practice, not a rewrite of the site)

Follow [Effective Go](https://go.dev/doc/effective_go) and
[Code Review Comments](https://go.dev/wiki/CodeReviewComments).
Highlights we actually care about here:

- `gofmt` is not optional. `go test ./...` is the unit gate.
- MixedCaps. No `snake_case` identifiers.
- Errors are values. Check them. Do not `_ = err`. Wrap:
  `fmt.Errorf("cache: sweep %s: %w", path, err)`.
- Error strings: lowercase, no trailing punctuation.
- `context.Context` is the first parameter. Do not store it on a
  long-lived struct. Do not pass `nil`. Use `context.Background()` or
  `TODO()` only at the top of `main` / tests, then derive.
- Accept interfaces, return concrete types.
- Small interfaces (one or two methods). Embedding is composition.
- Comments are sentences and start with the name of the thing.
- Table-driven tests. Testdata in `testdata/`.
- `init()` is tolerated in `cmd/` because cobra-cli generates it.
  Do not add `init()` in `internal/` without a hard reason.
- Receiver names: one or two letters from the type (`func (r *Runner)`).
- Generics are for containers and shared algorithms, not for
  "Arch vs Debian". That is an interface plus a `Resolve` function.

Package docs: `go doc` / [godoc comment rules](https://go.dev/doc/comment).

## Context in this program

Root context is `signal.NotifyContext` in `cmd`. Everything that can
block (exec, sleep, ticker, errgroup) takes that `ctx` or a child.

```
signal.NotifyContext
  +-- sudo keepalive: select { case <-ctx.Done(); case <-ticker.C }
  +-- each CommandContext(ctx, argv0, argv[1:]...)
  +-- wave errgroup.WithContext(ctx)   # hard fail cancels siblings
```

Ctrl-C cancels the root. Children receive SIGKILL/wait via
`CommandContext`. Keepalive exits because `Done()` fires. `defer`
still runs (mise reactivate, stop keepalive). See the spec section
"Context (normative)".

## Logging and errors

```
internal/config.Error
internal/program.Error
internal/resolve.Error
internal/runner.StepError
        \-> cmd.Execute maps to os.Exit
```

slog text on stderr. `--log-level` controls it. Tests pass a logger
that records to a `bytes.Buffer` or `slog.NewJSONHandler`.

## What not to do

- Do not add YAML because cobra-cli's scaffold defaulted to it.
- Do not parallelise pacman/aur/mirror because goroutines exist.
- Do not treat a program named `cache` as our cache sweep.
- Do not rewrite Effective Go into this file.
