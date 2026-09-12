---
title: "Implementation specs overview"
description: "Milestone order, shared types, and how cmd composes internal packages."
keywords: [impl, overview, milestones, types, cmd]
order: 20
---

# Implementation specs overview

Per-part specs for v1. The design doc remains the behaviour source of
truth. If an impl spec drifts, the design wins until it is amended.

Do not start code until the matching spec is approved and a goal call
names that milestone.

## Composer vs units

`cmd/` owns process lifetime: flags, logger, root `context`, exit
codes, and the order of wraps. Each `internal/` package is a unit with
its own error type and tests. Wraps (sudo, mise, cache, shutdown) are
called from `cmd`, not hidden inside program files.

```
cmd.Execute
  load config            (01)
  discover programs      (02)
  resolve pkg manager    (03)
  list / validate / run  (04)
  signal.NotifyContext
    sudo.KeepAlive       (06)
    mise.Begin + defer   (09; did_strip)
    runner.RunAll        (05, then 07, 08)
    cache.Sweep          (10)
    shutdown             (11)
```

## Milestone order

| ID | Spec | Builds | Needs |
| --- | --- | --- | --- |
| 01 | [config](01-config.md) | XDG + viper TOML + slog + `config.Error` | nothing |
| 02 | [discovery](02-discovery.md) | program files, sort, uniqueness | 01 (dir path) |
| 03 | [resolve](03-resolve.md) | PKG_MANAGER chain | 01 |
| 04 | [cli](04-cli.md) | flags, `list`, `validate`, exit map | 01-03 |
| 05 | [runner](05-runner.md) | sequential `CommandContext` | 02, 04 |
| 06 | [sudo](06-sudo.md) | keepalive goroutine | 05 |
| 07 | [continue](07-continue.md) | accumulate errors | 05 |
| 08 | [waves](08-waves.md) | `parallel` + errgroup | 05, 07 |
| 09 | [mise](09-mise.md) | PATH strip + `did_strip` | 05 |
| 10 | [cache](10-cache.md) | end-of-run dir sweep | 01, 05 |
| 11 | [shutdown](11-shutdown.md) | countdown + `shutdown -h` | 05, 07 |

01-05 is the first vertical slice (load, list, run one command, Ctrl-C
works). 06 is the first real background goroutine. 07-11 add behaviour
flags and wraps.

## Shared types

Locked here so later specs do not rename them.

```go
// internal/config
type Config struct {
    Sudo       Sudo
    Mise       Mise
    Cache      Cache
    PkgManager PkgManager
    Shutdown   Shutdown
}

type Sudo struct {
    Keepalive bool
    Interval  time.Duration // default 50s
}

type Mise struct {
    Wrap bool // default true
}

type Cache struct {
    Enabled     bool     // default true
    IncludeDirs []string
    ExcludeDirs []string
}

type PkgManager struct {
    Name string // last-resort binary name
}

type Shutdown struct {
    Force bool
    Wait  time.Duration // default 1m
}

// internal/program
type Spec struct {
    Name        string
    Alias       string
    Description string
    Optional    bool
    Parallel    bool
    Command     []string
    Source      string // file path, for errors
}

// cmd / runner
type Options struct {
    Skip          []string
    Continue      bool
    NoCache       bool
    Shutdown      bool
    ForceShutdown bool
}
```

## Defaults

`config.toml` is bootstrapped. Missing or empty default file => write
`Defaults()` to `AppConfig`, print the path, exit 0. `--config`
pointing at a missing file is an error. `--generate-config` writes
on demand and refuses to clobber a non-empty file.

Programs are required: no `programs.toml` and no `programs/*.toml` =>
error, nothing to run.

## Exit codes (implemented in 04, used by later cmd wiring)

| Code | `errors.As` target / case |
| --- | --- |
| 0 | nil |
| 2 | cobra flag parse failure |
| 3 | unknown skip token (`program.SkipError`) |
| 5 | `runner.StepError` (hard fail or continue with failures) |
| 1 | any other typed/untyped error |

## Tests

`go test ./...`. Table-driven. No live pacman. Inject `LookPath`,
filesystem (temp dirs), and `Runner` fakes. `testdata/` for TOML
fixtures where a string in the test is not enough.
