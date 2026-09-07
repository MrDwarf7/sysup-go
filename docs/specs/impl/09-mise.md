---
title: "09 mise"
description: "Deactivate mise before the plan; defer reactivate; mise up only on a clean run."
keywords: [impl, mise, defer, wrap]
order: 29
---

# 09 mise

Mise shims can shadow system tools. Wrap the plan: deactivate, run,
reactivate. `defer` is the cleanup. Not a program.

## Depends on

05 (RunE exists). 01 (`Config.Mise.Wrap`). 07 (need to know if the
plan had failures before `mise up`).

## Files

- Create: `internal/mise/wrap.go`
- Create: `internal/mise/error.go`
- Create: `internal/mise/wrap_test.go`
- Modify: `cmd/root.go` around `RunAll`

## Types

```go
package mise

type LookPath func(string) (string, error)
type Runner func(ctx context.Context, name string, args ...string) error

type Wrap struct {
    LookPath LookPath
    Run      Runner
    Log      *slog.Logger
}

// Begin deactivates if mise exists and wrap is enabled.
// End reactivates. up is true when the plan had zero failures.
func (w Wrap) Begin(ctx context.Context, enabled bool) (end func(up bool), err error)
```

cmd:

```
end, err := miseWrap.Begin(ctx, cfg.Mise.Wrap)
if err != nil { return err }
defer func() { end(planClean) }()
```

`planClean` is a bool the RunE sets false on step errors. Because
`defer` sees the named result or a closed-over variable, use a
`clean := true` variable and set it false when `RunAll` returns a
step failure. Context cancel is not clean.

## Behaviour

`enabled == false`: `Begin` returns a no-op `end`.

`LookPath("mise")` fails: log info, no-op `end`. Not an error.

Otherwise:

1. `Run(ctx, "mise", "deactivate")`. Failure: log warn, continue
   (deactivate can fail if already off). Do not abort the plan.
2. `end(up)`:
   - `Run(ctx, "mise", "activate", "bash")` is **wrong** for us:
     Fish did `mise activate fish | source`. We are not a shell.
     Reactivate for the **user's later interactive shell** is not
     something a child process can persist into the parent fish.
     Lock: we only need the rest of **this process** to see system
     binaries, which deactivate already did by mutating this
     process environment if `mise deactivate` prints exports...

Fish: `mise deactivate` unsets shims in the current shell;
`mise activate fish | source` restores; `mise up` updates tools.

In Go we are a subprocess. `mise deactivate` as a child **cannot**
change our env unless we apply its stdout (env dump) ourselves.

Lock this:

- `Begin`: run `mise deactivate --quiet` **or** strip `MISE_*` /
  shim dir from `os.Environ` if documented. Prefer: execute
  `mise deactivate` with env dump if mise supports printing shell
  env; otherwise run the plan with `cmd.Env` inherited after
  removing the mise shims directory from `PATH`.

Practical v1 (do this):

1. `LookPath("mise")`.
2. Read `MISE_SHIMS_DIR` or default `~/.local/share/mise/shims`
   (and `mise bin-paths` if we want later).
3. For every `Exec`, set `cmd.Env` to current env with that shims
   dir removed from `PATH`. That is the wrap: children do not see
   shims.
4. After a clean plan: `Run(ctx, "mise", "up")`. Failure is a
   `StepError`-like error from cmd (exit 5) after programs ran.
5. Do not attempt `activate | source`. The parent fish still has
   mise; we only care that **our children** skip shims.

This matches the Fish *intent* (do not let mise shadow pacman)
without pretending a Go process can re-source the parent shell.

If `mise up` is skipped on dirty/crash: `up == false`.

Tests inject PATH lists rather than real mise.

## Tests

| Case | Expect |
| --- | --- |
| wrap false | no LookPath required, end no-op |
| mise missing | no-op, nil |
| PATH with shim dir, child `Exec` | child env PATH lacks shim dir |
| clean end | `mise up` called once |
| dirty end | `mise up` not called |

## Out of scope

Hooking Fish `source`. Windows.

## Done when

`go test ./internal/mise/` green. A program `command = ["which",
"ruby"]` (or similar) does not resolve a mise shim while wrap is on,
if you have mise locally for a manual check.
