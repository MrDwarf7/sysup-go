---
title: "07 continue"
description: "-c accumulates program errors, prints them at the end, still exits 5."
keywords: [impl, continue, errors, skip, hard-fail]
order: 27
---

# 07 continue

Default remains hard fail. `-c` / `--continue` runs the rest of the
plan after a program error and reports every failure at the end.

## Depends on

05. Shutdown gating is specified here but **not implemented** until
11; do not shut down in this milestone.

## Files

- Modify: `internal/runner/run.go` (`RunAll` grows `continueOnErr bool`)
- Modify: `internal/runner/error.go` (`Errors` multi-error)
- Modify: `internal/runner/run_test.go`
- Modify: `cmd/root.go` pass `Options.Continue`
- Modify: `cmd/exit.go` map `runner.Errors` / `StepError` to 5

## Types

```go
package runner

type Errors struct {
    Steps []StepError
}

func (e *Errors) Error() string // one step per line: "name: err"
func (e *Errors) Unwrap() []error // Go 1.20 multi-unwrap
```

`RunAll(ctx, log, steps, continueOnErr bool) error`

- `continueOnErr == false`: first `StepError` returned (current 05).
- `continueOnErr == true`: collect `StepError`s, always finish the
  slice (unless `ctx` cancelled). If len==0, return nil. Else return
  `*Errors`.

Ctrl-C (`ctx.Done()`) during continue: stop starting new steps,
return `ctx.Err()` **and** any collected step errors if you can wrap
both; prefer `Errors` plus a final step named `"context"` **or**
return `ctx.Err()` only if no steps failed. Lock: if any steps
failed **and** ctx cancelled, return `*Errors` (failures are what
the user cares about) after the loop breaks. If none failed and ctx
cancelled, return `ctx.Err()`.

PreRun/Run/PostRun: a PreRun failure counts as that step failed;
do not call Run/PostRun for that step. Continue to the next step
when `-c`.

## Output

cmd prints `err.Error()` on stderr when `RunAll` returns. slog
already logged each failure at error as it happened (`step failed`
attr name). Do not double-print a novel format; `Errors.Error()` is
the summary.

Exit code 5 if any step failed, including under `-c`.

## Tests

| Case | Expect |
| --- | --- |
| hard fail, step 2 errors | step 3 not run |
| continue, step 2 errors, step 3 ok | all run, `*Errors` len 1, name of 2 |
| continue, two failures | len 2, both names |
| continue, all ok | nil |
| continue, ctx cancel after step 1 | step 2+ not started |

## Out of scope

Parallel waves (08). Shutdown (11). Cache still not wired.

## Done when

`go test ./internal/runner/` covers both modes.
`sysup-go -c` with a failing `false` program and a later `true`
runs both and exits 5.
