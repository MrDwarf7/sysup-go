---
title: "05 runner"
description: "Sequential exec via CommandContext, signal.NotifyContext, hard fail."
keywords: [impl, runner, context, exec, CommandContext, signal]
order: 25
---

# 05 runner

Run the plan one program at a time. Root context is cancelled on
SIGINT/SIGTERM. Each child is bound to that context. First error
stops the rest (continue is 07).

## Depends on

02, 04. Uses `program.Spec` and `Options.Skip` (already filtered in
cmd, or filter here; pick **cmd filters then passes `[]Spec`**).

## Files

- Create: `internal/program/exec.go` (`Exec` Runner)
- Create: `internal/program/exec_test.go`
- Create: `internal/runner/run.go`
- Create: `internal/runner/error.go`
- Create: `internal/runner/run_test.go`
- Modify: `cmd/root.go` attach `RunE`

## Types

```go
package program

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

type Exec struct {
    Spec Spec
    // stdout/stderr: default os.Stdout/os.Stderr; tests inject buffers
    Stdout io.Writer
    Stderr io.Writer
    LookPath func(string) (string, error) // optional; default exec.LookPath
}

func (e Exec) Run(ctx context.Context) error
func (e Exec) Meta() Spec

package runner

type StepError struct {
    Name string
    Err  error
}

func RunAll(ctx context.Context, log *slog.Logger, steps []program.Runner) error
```

`Exec` does **not** implement `PreRunner`/`PostRunner`. No no-op
methods.

`RunAll` type-asserts Pre/Post around `Run`. User execs only hit
`Run`.

## Exec behaviour

1. If `ctx.Err() != nil`, return it without starting a process.
2. `optional` and argv0 missing (`LookPath` fails): log at info,
   return **nil** (skip, not a failure).
3. `optional == false` and argv0 missing: error, becomes StepError.
4. `cmd := exec.CommandContext(ctx, argv0, argv[1:]...)`.
   `cmd.Stdout` / `Stderr` as injected. `cmd.Stdin = os.Stdin` (TTY
   sudo prompts still work until 06).
5. `cmd.Run()`. Non-zero exit: error wrapping `*exec.ExitError`.
6. No shell. No `bash -c`.

## RunAll behaviour (this milestone)

For each step, in slice order:

1. PreRun if asserted, abort on error.
2. Run, abort on error.
3. PostRun if asserted, abort on error.

Wrap step failures as `StepError{Name, Err}`. Name comes from
`Meta()` if the concrete type has it; for `Exec` use `Spec.Name`.
For fakes in tests, a small `named` helper is fine.

Do not implement continue, waves, sudo, mise, cache, shutdown here.

## cmd RunE

```
ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

`cmd.Context()` is cobra's context. Derive NotifyContext from it.

Load config, load programs, filter skip, build `[]Exec`, `RunAll`.
Map `StepError` to exit 5 in `exitCode`.

## Tests

| Case | Expect |
| --- | --- |
| two fakes, both nil | both ran, order preserved |
| second fake errors | third not called, `StepError` name of second |
| ctx cancelled before loop | no steps run, `ctx.Err()` |
| Exec with `optional`, missing binary | nil error |
| Exec required, missing binary | error |
| Exec `CommandContext` + cancel while `sleep 30` | returns quickly, not 30s |

The last test: `exec.CommandContext` with `sleep` (or a tiny Go
`os.Args` helper). Cancel after 50ms. Assert elapsed << 30s.

No pacman.

## Out of scope

`-c`, `parallel`, sudo ticker, mise, cache, shutdown.

## Done when

`go test ./internal/runner/ ./internal/program/` green.
`sysup-go` with a programs dir of `command = ["true"]` (or `echo`)
exits 0. Ctrl-C during `sleep 10` exits promptly.
