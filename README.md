# sysup-go

Port of the Fish `sysup` orchestrator to Go. Run an ordered list of user-defined programs from TOML config -- no rebuild needed.

```
sysup-go            # run the plan
sysup-go list       # show programs in order
sysup-go validate   # check config without running
sysup-go -c         # continue on errors
sysup-go -s pacman  # skip a program by name or alias
```

## What It Does

You write TOML files describing programs (commands to run, in order). sysup-go executes them sequentially, with optional waves (parallel groups), sudo keepalive, mise wrapping, and end-of-run cache sweeps. Programs are just argv arrays -- no shell, no magic filenames.

Config lives in `$XDG_CONFIG_HOME/sysup-go/`:

```
config.toml         # our behavior (sudo, mise, cache, shutdown)
programs/*.toml     # OR programs.toml -- what to run, in what order
```

## Quick Config

```toml
# config.toml
[sudo]
keepalive = true
interval = "50s"

[mise]
wrap = true

[cache]
enabled = true

[shutdown]
force = false
wait = "1m"
```

```toml
# programs/20-pacman.toml
name = "pacman"
alias = "p"
description = "Update official repository packages"
command = ["sudo", "pacman", "-Syyu", "--needed", "--noconfirm"]
```

## Install

```bash
git clone https://github.com/MrDwarf7/sysup-go
cd sysup-go
just build-release    # -> bin/sysup
```

Or download a release binary from the [Releases](https://github.com/MrDwarf7/sysup-go/releases/latest) page.

## Build

```bash
just b      # debug (race + no-opt)
just br     # release (CGO off, stripped)
just t      # test (race, shuffle, vet)
just a      # full pipeline: format -> check -> test -> build-all
```

## Project Structure

```
sysup-go/
  main.go              # entry point
  cmd/                 # cobra flags, logger, exit codes
  internal/
    config/            # viper TOML + slog
    program/           # discovery, sort, decode
    resolve/           # PKG_MANAGER chain
    runner/            # plan, waves, continue
  justfile             # task runner
  .taplo.toml          # TOML formatting config
```

## License

MIT
