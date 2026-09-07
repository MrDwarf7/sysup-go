---
title: "09 mise"
description: "Strip mise shims from child PATH once; restore only if did_strip; mise up on a clean run."
keywords: [impl, mise, did_strip, PATH, defer]
order: 29
---

# 09 mise

Mise shims can shadow system tools. Fish used `deactivate` / `source
activate` because it was a shell. We are not a shell. Strip the shims
directory from the env we hand to **child programs**, remember whether
we did, and only restore if we did.

Not a program.

## Depends on

05 (RunE exists). 01 (`Config.Mise.Wrap`). 07 (`planClean` for
`mise up`).

## Files

- Create: `internal/mise/wrap.go`
- Create: `internal/mise/error.go`
- Create: `internal/mise/wrap_test.go`
- Modify: `cmd/root.go` around `RunAll` / `Exec` env
- Modify: `internal/program/exec.go` if `Exec` needs an `Env []string`

## Types

```go
package mise

type LookPath func(string) (string, error)

type State struct {
    DidStrip bool
    PathOrig string // set only when DidStrip
    ShimDir  string // the entry we removed, for tests/logs
}

func (s State) ChildEnv(base []string) []string
func (s State) RestoreProcessPath()

type Wrap struct {
    LookPath LookPath
    Log      *slog.Logger
}

func (w Wrap) Begin(enabled bool) State
```

`Begin` does **not** re-read PATH at End. End uses `State` only.

cmd:

```
st := miseWrap.Begin(cfg.Mise.Wrap)
defer st.RestoreProcessPath()
// pass st.ChildEnv(os.Environ()) into each Exec
// after RunAll, if st.DidStrip && planClean { mise up }
```

## Behaviour

`enabled == false`: `State{DidStrip: false}`. No PATH walk.

`LookPath("mise")` fails: same, `DidStrip: false`. Log info. Not an
error.

Otherwise, **once** at Begin:

1. Resolve shim dir: `MISE_SHIMS_DIR` if set, else
   `$XDG_DATA_HOME/mise/shims` / `~/.local/share/mise/shims`.
2. Walk `PATH` **once**. If that dir is an entry, set
   `DidStrip: true`, `PathOrig: os.Getenv("PATH")`, `ShimDir: dir`,
   and `os.Setenv("PATH", stripped)` for this process so we do not
   rebuild env per child from a stale parent PATH. Children inherit
   the stripped PATH unless `Exec` overrides `Env`.
3. If the shim dir is not on PATH: `DidStrip: false`. We did not
   touch anything.

`RestoreProcessPath`: if `!DidStrip`, return immediately. If
`DidStrip`, `os.Setenv("PATH", PathOrig)`. Do not parse PATH again.

`mise up`: only if `DidStrip && planClean`. We only update tools when
we actually took mise out of the way at startup. Dirty/cancel: skip
`mise up`. `mise up` failure is exit 5 after the plan (log + return
error from RunE).

Do not run `mise deactivate` / `mise activate` / `source`. Do not
expand env vars inside program `command` arrays.

## Tests

Inject PATH via `t.Setenv`. Do not require real mise; inject LookPath.

| Case | Expect |
| --- | --- |
| wrap false | DidStrip false, PATH unchanged |
| mise missing | DidStrip false, PATH unchanged |
| shim dir on PATH | DidStrip true, PATHOrig saved, process PATH lacks shim |
| RestoreProcessPath after strip | PATH equals PathOrig |
| RestoreProcessPath when !DidStrip | PATH unchanged (no second parse) |
| DidStrip && clean | `mise up` called once |
| DidStrip && dirty | `mise up` not called |
| !DidStrip && clean | `mise up` not called |

## Out of scope

Fish `source`. Windows. Env expansion in recipe `command`.

## Done when

`go test ./internal/mise/` green. Restore is a stored-string write,
not a second PATH split.
