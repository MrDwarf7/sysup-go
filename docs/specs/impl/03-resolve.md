---
title: "03 resolve"
description: "PKG_MANAGER env -> paru/yay probe -> config name. Parallel LookPath."
keywords: [impl, resolve, pkg_manager, paru, yay, goroutine]
order: 23
---

# 03 resolve

Pick the AUR helper binary. Arch-family only. Failure is how Nix and
Debian die for v1.

## Depends on

01 (`Config.PkgManager.Name`). Independent of discovery.

## Files

- Create: `internal/resolve/resolve.go`
- Create: `internal/resolve/error.go`
- Create: `internal/resolve/resolve_test.go`

## Types

```go
package resolve

type LookPath func(file string) (string, error) // usually exec.LookPath

type Env func(key string) (string, bool) // usually os.LookupEnv

type Resolver struct {
    LookPath LookPath
    LookupEnv Env
}

type Error struct {
    Op  string
    Err error
}

func (r Resolver) PkgManager(ctx context.Context, cfgName string) (string, error)
```

Do not call `os.Getenv` / `exec.LookPath` directly in the logic;
always go through `Resolver` so tests inject maps and fake paths.
`cmd` constructs `Resolver{LookPath: exec.LookPath, LookupEnv: os.LookupEnv}`.

`LookupEnv` must distinguish unset vs empty: use `os.LookupEnv`
(ok=false vs ok=true with `""`).

## Behaviour

Honour `ctx`: if `ctx.Err() != nil` at entry or after probes, return
it.

1. Env `PKG_MANAGER` unset (`ok == false`): fall through.
2. Env set and `val == ""`: error immediately. Do not probe.
3. Env set and non-empty: `LookPath(val)`. Miss => error, no
   fallthrough. Hit => return that path (or the name; pick **path**
   from LookPath so later exec is unambiguous).
4. Else start two goroutines: `LookPath("paru")` and `LookPath("yay")`.
   Wait with a small `errgroup` or `sync.WaitGroup` + results on
   channels; both must finish or `ctx` cancel. Both found: **paru
   wins**. One found: that one. Neither: fall through.
5. `cfgName` non-empty: `LookPath(cfgName)`. Miss => error. Hit =>
   return it.
6. Else error: cannot resolve a package manager.

Do not treat "paru" as a reserved program name. This function only
returns a binary path for callers that need `$PKG_MANAGER` (aur
program argv can still be a literal `paru` in TOML). Milestone 05
does not rewrite user argv. Resolve exists so we can substitute or
check; v1 `cmd validate` prints the resolved path. Runtime rewrite of
`command[0] == "$PKG_MANAGER"` is **out of scope** unless a later
goal says otherwise. Document that programs should put the real
binary in `command`.

Wait: the Fish aur step uses `$PKG_MANAGER`. User programs in TOML
are argv arrays. If they write `command = ["paru", "-Syu", ...]` they
bypass resolve. Resolve is still required for validate/diagnostics
and for cache (10) if we exec the helper. Spec: `PkgManager` result
is stored on a small `Env` bag passed to the runner for logging and
for cache. Do not magically rewrite program commands in v1.

## Tests

Inject `LookPath` and `LookupEnv`. Never require real paru.

| Case | Expect |
| --- | --- |
| unset env, paru and yay both "found" | paru path |
| unset, only yay | yay path |
| unset, neither, cfg `name = "custom"` found | custom path |
| unset, neither, empty cfg | error |
| env `PKG_MANAGER=""` | error, no LookPath calls |
| env `PKG_MANAGER=foo` missing | error |
| env `PKG_MANAGER=foo` found | foo, skip probes |
| ctx already cancelled | error `ctx.Err()`, probes optional |

Assert paru-over-yay with a fake that records call order is **not**
required; asserting the winner is.

## Out of scope

pacman itself, Debian, rewriting program argv.

## Done when

`go test ./internal/resolve/` is green. No cobra wiring yet (04).
