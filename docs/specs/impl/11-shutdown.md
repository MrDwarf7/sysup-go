---
title: "11 shutdown"
description: "Countdown then shutdown -h now. Only on a clean run unless force."
keywords: [impl, shutdown, force-shutdown, countdown]
order: 31
---

# 11 shutdown

After cache (10), maybe shut the machine down. Default: never, unless
asked. Dirty plan: never, unless `--force-shutdown` or
`config.shutdown.force`.

## Depends on

05, 07, 10 (cache errors count as failures). 01 (`Shutdown.Wait`,
`Force`).

## Files

- Create: `internal/shutdown/shutdown.go`
- Create: `internal/shutdown/error.go`
- Create: `internal/shutdown/shutdown_test.go`
- Modify: `cmd/root.go` last step of RunE

## Types

```go
package shutdown

type Halt func(ctx context.Context) error // default: sudo shutdown -h now

func Run(ctx context.Context, log *slog.Logger, wait time.Duration, halt Halt) error
```

cmd decision table **before** calling `Run`:

| force flag/config | -d / --shutdown | any failure (steps or cache) | action |
| --- | --- | --- | --- |
| true | * | * | countdown + halt |
| false | true | none | countdown + halt |
| false | true | yes | do not halt; log warn |
| false | false | * | do not halt |

`--force-shutdown` implies halt even without `-d`.

`wait` default 1m. `wait <= 0`: still log once, then halt (tests use
0).

## Countdown

Sleep in 1s ticks (or `wait` if smaller). Each tick: log
`shutdown in Ns` at warn. Honour `ctx`: cancel => return `ctx.Err()`,
**do not halt**. That is how Ctrl-C aborts the countdown (Fish
behaviour).

After wait completes: `halt(ctx)`. Default:

```
exec.CommandContext(ctx, "sudo", "shutdown", "-h", "now")
```

Failure: return error (exit 1, not 5; the update already finished).

## Tests

Inject `Halt` that records calls. Fake wait 0 or 10ms.

| Case | Expect |
| --- | --- |
| wait 0, ctx live | Halt called once |
| ctx cancelled during wait | Halt not called, `ctx.Err()` |
| decision table (pure func `ShouldHalt(force, want, dirty bool) bool`) | four rows above |

Put `ShouldHalt` in this package so cmd does not inline the table
wrong.

## Out of scope

systemd inhibit, GUI confirm. Short flag for force (there is none).

## Done when

`go test ./internal/shutdown/` green. Manual: `--shutdown` with
`wait = "3s"` on a plan of `true` logs the countdown; Ctrl-C during
countdown does not halt (use a Halt fake in an integration test if
you are not willing to risk a real shutdown; **never** call real
`shutdown -h` in tests).
