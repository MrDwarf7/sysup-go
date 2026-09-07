---
title: "06 sudo"
description: "Keepalive goroutine: sudo -v, then ticker sudo -n true until ctx.Done."
keywords: [impl, sudo, keepalive, goroutine, context, ticker]
order: 26
---

# 06 sudo

Stop sudo from timing out during a long pacman/aur run. One
goroutine, one context.

## Depends on

05 (root ctx already exists). 01 (`Config.Sudo`).

## Files

- Create: `internal/sudo/keepalive.go`
- Create: `internal/sudo/error.go`
- Create: `internal/sudo/keepalive_test.go`
- Modify: `cmd/root.go` `RunE` to start/stop keepalive when enabled

## Types

```go
package sudo

type Runner func(ctx context.Context, name string, args ...string) error

type KeepAlive struct {
    Interval time.Duration
    Run      Runner // tests inject; default exec.CommandContext
}

func (k KeepAlive) Start(ctx context.Context) (stop func(), err error)
```

Default `Run` executes `exec.CommandContext(ctx, name, args...).Run()`
with stdout/stderr discarded (keepalive must not spam).

## Behaviour

If `cfg.Sudo.Keepalive == false`, cmd does not call `Start`.

`Start`:

1. Interval <= 0: use 50s.
2. Foreground: `Run(ctx, "sudo", "-v")`. Failure => return error
   (cannot prime). Do not start the goroutine.
3. Launch a goroutine:

```
ticker := time.NewTicker(interval)
defer ticker.Stop()
for {
    select {
    case <-ctx.Done():
        return
    case <-ticker.C:
        _ = Run(ctx, "sudo", "-n", "true")
        // log debug on failure; do not cancel the plan
    }
}
```

4. Return `stop` that is a no-op beyond relying on `ctx` cancel.
   cmd already `defer stop()` on the signal ctx; that is enough.
   If `Start` wants its own `stop`, it may call a child cancel; do
   not leak the goroutine if `RunE` returns without the process
   exiting (tests).

Recommended: `Start` derives `bg, cancel := context.WithCancel(ctx)`
for the goroutine and returns `cancel` as `stop`. cmd:

```
stopKA, err := ka.Start(ctx)
if err != nil { return err }
defer stopKA()
```

So tests can `stopKA()` without signalling the process.

`sudo -n true` failure (timestamp expired, sudo missing): log warn,
keep ticking. Do not abort the update. The next program's sudo
prompt is the recovery.

## Tests

Inject `Run` that records calls with timestamps.

| Case | Expect |
| --- | --- |
| Start, immediate cancel | `sudo -v` once, no `-n` required |
| Interval 20ms, live 70ms | `sudo -v` once, at least two `-n true` |
| `sudo -v` fails | Start returns error, no goroutine work after return |
| after stop(), no further Run calls (wait 2 intervals) | count stable |

Use a fake clock **only if** you already have one; `time.Ticker` with
short intervals is enough. Never call real sudo in tests.

## Out of scope

Asking for a password in Go. `sudo -v` uses the TTY as today.

## Done when

`go test ./internal/sudo/` green. Manual: run a dummy long `sleep`
program with keepalive interval 10s and confirm `sudo -n true`
appears in debug logs (or `ps`) until Ctrl-C.
