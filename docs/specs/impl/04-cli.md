---
title: "04 cli"
description: "Root flags, list, validate, exit-code mapping. cobra-cli add."
keywords: [impl, cobra, flags, list, validate, skip, continue]
order: 24
---

# 04 cli

Wire cobra to 01-03. Root still does not run programs (that is 05).
`list` and `validate` become usable.

## Depends on

01, 02, 03.

## Files

- Modify: `cmd/root.go` (flags, short/long text, `initConfig`, logger,
  exit mapping)
- Create via `cobra-cli add list`: `cmd/list.go`
- Create via `cobra-cli add validate`: `cmd/validate.go`
- Create: `cmd/exit.go` (map error -> code)
- Create: `cmd/exit_test.go`

Run, from repo root:

```
cobra-cli add list
cobra-cli add validate
```

Then edit the generated files. Do not hand-roll the cobra command
registration boilerplate if `cobra-cli` is available; if it is not,
copy the same `init()` + `rootCmd.AddCommand` pattern cobra-cli
emits.

## Flags (root, persistent where listed)

| Flag | Cobra | Default |
| --- | --- | --- |
| `--config` | `String` (already exists) | "" |
| `-s, --skip` | `StringSliceP` | nil |
| `-c, --continue` | `BoolP` | false |
| `--no-cache` | `Bool` | false |
| `-d, --shutdown` | `BoolP` | false |
| `--force-shutdown` | `Bool` (long only) | false |
| `--log-level` | `String` (`info` / `debug` / `warn` / `error`) | `info` |

`--force-shutdown` is **not** mutually exclusive with `-d`. Force
implies shutdown. Do not `MarkFlagsMutuallyExclusive` those two.

`-c` is continue. It is not cache. Help text must say so.

Root `Use`: `sysup-go`. Short: system update orchestrator. Long:
points at programs dir, mentions `-s` vs `-c`.

Root `RunE` in this milestone: if someone runs bare `sysup-go`,
return a clear "run not implemented" error **or** leave `RunE` unset
so cobra shows help. Prefer **help on no-op** (no `Run`) until 05
attaches `RunE`. Do not fake a successful update.

## list

Load config dir (from `--config` or default Dir). `program.Load`.
Print one program per line, in order, to stdout. Format:

```
p  pacman  Update official repository packages
-  rustup  Update rustup toolchains
```

Column 1 is alias or `-` if none. Stable, parseable, no decoration.
`--skip` is allowed and filters the list (unknown skip => exit 3).

## validate

Load config, load programs, resolve PKG_MANAGER, apply skip.
Print to stdout:

```
config:  <path or "defaults">
pkg_manager: <path>
programs: N
```

Then the same list as `list`. No exec. Resolve failure => exit 1.
Unknown skip => exit 3.

## Exit mapping

```go
func exitCode(err error) int
```

| err | code |
| --- | --- |
| nil | 0 |
| `program.SkipError` | 3 |
| cobra `flag` errors (already handled by cobra before RunE) | 2 |
| anything else | 1 |

`Execute` uses `os.Exit(exitCode(err))`. StepError (code 5) lands in
05; add the `errors.As` branch then, not a placeholder now.

## Tests

`exitCode` table test in `cmd/exit_test.go`.

Discovery/skip behaviour is already tested in `internal/program`.
CLI tests: optional `cobra` `SetArgs` + `Execute` against a temp XDG
if cheap; not required if `list`/`validate` stay thin wrappers.
Minimum: `exitCode` unit tests + `go build`.

## Out of scope

Running programs, sudo, cache sweep.

## Done when

`go build .` and `go test ./...` pass.
`sysup-go list` and `sysup-go validate` work against a temp config
tree you can document in the PR/commit message. Help shows `-c` as
continue and `-s` as skip.
