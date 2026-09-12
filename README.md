# sysup-go

Port of the Fish `sysup` orchestrator to Go. Run an ordered list of user-defined programs from TOML config -- no rebuild needed.

```
sysup-go                  # run the plan
sysup-go list             # show programs in order
sysup-go validate         # check config without running
sysup-go --generate-config  # write default config.toml and exit
sysup-go -c               # continue on errors
sysup-go -s pacman        # skip a program by name or alias
sysup-go -s m             # skip mirror AND mirror-* follow-ups
```

## What It Does

You write TOML files describing programs (commands to run, in order). sysup-go executes them sequentially, with optional waves (parallel groups), sudo keepalive, mise wrapping, and end-of-run cache sweeps. Programs are just argv arrays -- no shell, no magic filenames.

Config lives in `$XDG_CONFIG_HOME/sysup/` (`AppDir` / `AppConfig`).
Missing or empty `config.toml` is written from `Defaults()` and the
process exits so you can edit it. `--generate-config` does the same
on demand.

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
strip_bins_from_path = true

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

## Retries

`[retries]` in `config.toml` is the global policy. The same table on a
program file applies to that recipe only. `max_attempts` is total runs
including the first (0 = unset). A positive **global** max always wins:
recipe 10 + global 3 => 3 tries.

`always = false` retries only classified failures (paru
`can not install conflicting packages with --noconfirm`, dropped AUR
RPC). That noconfirm conflict retries with paru `--useask` (pacman's
`--ask`, auto-confirm conflicts) still under `--noconfirm`.
`always = true` retries any non-zero exit.

## Mise

`[mise] wrap` (default true) plus `strip_bins_from_path` (default true)
drops mise shims **and** `mise/installs/*/bin` from PATH for child
processes (paru/makepkg must see `/usr/bin/python`, not mise's). This
is not `mise deactivate`. PATH is restored after the plan; `mise up`
runs only if we stripped and the plan was clean. Set
`strip_bins_from_path = false` to leave PATH alone.

## Logging

Stderr is `level | time | msg` so columns line up. The level is colored
on a TTY (the usual slog-console look; we keep `log/slog` and do not
use zap/zerolog). A second copy is appended to
`os.TempDir()/sysup.log` by default (macOS has no `$TEMP`; `os.TempDir`
uses `$TMPDIR` or `/tmp`). `--log-file -` or an empty value turns the
file off.

## Skip

`-s` / `--skip` matches a program `name` or `alias` (exact, case-sensitive).
Unknown tokens exit 3.

Quirk: skipping a name also drops later programs named `name-*`.
Recipes that are one logical step split across files (for example
`mirror`, then `mirror-stage` / `mirror-backup` / `mirror-swap`) stay
consistent when the first step is omitted. `-s m` (alias of `mirror`)
does not run `mirror-stage`. Skipping only `mirror-stage` still runs
`mirror`. `mirrors` is not a child of `mirror`; the cut is the hyphen
after the skipped name.

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
