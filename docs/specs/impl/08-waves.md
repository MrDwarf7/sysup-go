---
title: "08 waves"
description: "Consecutive parallel=true programs share an errgroup wave."
keywords: [impl, parallel, waves, errgroup, context]
order: 28
---

# 08 waves

`parallel` does not reorder. After the existing sort/filter, group
consecutive `parallel == true` specs into one wave. `false` is always
a wave of one (barrier).

## Depends on

05, 07. Each `Exec` must expose `Spec.Parallel` to the grouper.
Pass `[]Exec` or a small `Step` struct `{ Runner, Parallel, Name }`
so fakes can set Parallel without being `Exec`.

## Files

- Create: `internal/runner/waves.go` (`groupWaves`)
- Create: `internal/runner/waves_test.go`
- Modify: `internal/runner/run.go` to run waves
- Add: `golang.org/x/sync/errgroup` if not already a direct dep

## Types

```go
type step struct {
    Name     string
    Parallel bool
    Runner   program.Runner
}

func groupWaves(steps []step) [][]step
```

Algorithm (walk once):

- If current item `Parallel == false`: flush any open wave, emit
  `[item]` as its own wave.
- If `Parallel == true`: append to the current open parallel wave,
  or start one.

Example from the design:

```
F F F F T T T  =>  [m] [p] [a] [r] [n,y,h]
```

Also:

```
T T F  =>  [n,y] [p]
F T F  =>  [m] [n] [p]   // wave of one true is fine
```

## Run

For a wave of len 1: same as 05/07 (Pre/Run/Post on that ctx).

For a wave of len > 1:

**Hard fail** (`continueOnErr == false`):

```
g, gctx := errgroup.WithContext(ctx)
for each step: g.Go(func() error { return runOne(gctx, step) })
err := g.Wait()
```

First error cancels `gctx`, siblings' `CommandContext` die. Return
that `StepError`.

**Continue**:

Do **not** use `errgroup.WithContext` cancel-on-error. Use
`errgroup.Group` bound to `ctx` only (or manual WaitGroup). Wait for
every sibling. Collect `StepError`s into `*Errors`. Then next wave.

`runOne` is the Pre/Run/Post sequence.

Cap GOMAXPROCS? No. Wave size is tiny (3-5).

## Tests

Table tests on `groupWaves` for the four patterns above plus empty
and single F / single T.

Run tests with fakes that record start/end times:

| Case | Expect |
| --- | --- |
| T,T with 50ms sleep each, hard fail none | both overlap (start of second before end of first) |
| T,T, second errors, hard fail | first's ctx cancelled or Wait returns before 10s sleep finishes |
| T,T, second errors, continue | both finish, Errors len 1 |
| F then T,T | F fully finishes before either T starts |

Overlap assertion: use a barrier fake (channel handshake) rather
than timing flakiness where possible.

## Out of scope

Changing user filenames. Do not auto-mark rustup parallel.

## Done when

`go test ./internal/runner/` includes wave grouping and at least one
overlap/cancel test. Manual: three `sleep 2` programs with
`parallel = true` finish in ~2s, not ~6s.
