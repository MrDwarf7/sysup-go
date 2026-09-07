---
title: "01 config"
description: "XDG directory, viper TOML, defaults, slog, config.Error."
keywords: [impl, config, viper, xdg, toml, slog]
order: 21
---

# 01 config

Load `$XDG_CONFIG_HOME/sysup-go/config.toml` (fallback
`~/.config/sysup-go/config.toml`) via viper as TOML. Replace the
cobra-cli YAML default. Build an injected `*slog.Logger`.

## Depends on

Nothing. First code milestone.

## Files

- Create: `internal/config/config.go` (`Config` types, defaults)
- Create: `internal/config/load.go` (`Dir`, `Load`)
- Create: `internal/config/error.go`
- Create: `internal/config/load_test.go`
- Modify: `cmd/root.go` (`initConfig`, drop YAML, `--config`, logger)

## Types

```go
package config

type Error struct {
    Op   string // "dir", "read", "decode"
    Path string
    Err  error
}

func (e *Error) Error() string
func (e *Error) Unwrap() error

func Dir() (string, error) // UserConfigDir + "/sysup-go"
func Load(path string) (Config, error)
func Defaults() Config
```

`Load("")` means "use Dir()/config.toml". `Load("/abs/file.toml")` is
the `--config` override (must exist).

TOML keys match the design:

```toml
[sudo]
keepalive = true
interval = "50s"

[mise]
wrap = true

[cache]
enabled = true
include_dirs = []
exclude_dirs = []

[pkg_manager]
name = ""

[shutdown]
force = false
wait = "1m"
```

`interval` and `wait` decode as `time.Duration`. Invalid duration is a
decode error.

## Behaviour

1. `Dir` uses `os.UserConfigDir` (honours `XDG_CONFIG_HOME` on Linux).
   Join `"sysup-go"`. Do not mkdir; load does not create the tree.
2. Missing default `config.toml`: return `Defaults()`, not an error.
3. `--config` / `Load(explicit)` and the file is missing: `Error{Op:"read"}`.
4. Present but invalid TOML: `Error{Op:"decode"}`.
5. Unknown keys in config.toml: error (fail fast). Wire viper /
   mapstructure so unused keys are not silently dropped.
6. Logger: `slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))`.
   Default level info. `--log-level` may land in 04; until then, info
   is fine. Pass `*slog.Logger` into later constructors; do not call
   `slog.SetDefault` except as a last resort in `main`.

Viper: `SetConfigType("toml")`, `SetConfigName("config")`,
`AddConfigPath(dir)`. Do not search `$HOME/.sysup-go.yaml`.

## Tests

| Case | Expect |
| --- | --- |
| no file in temp XDG | Defaults(), err nil |
| valid full toml | fields match, durations 50s and 1m |
| bad duration (`interval = "nope"`) | decode error, path set |
| missing explicit path | read error |
| unknown key `[foo] bar = 1` | decode error |
| `Dir` with `XDG_CONFIG_HOME` set | `$XDG_CONFIG_HOME/sysup-go` |

Set `XDG_CONFIG_HOME` to a `t.TempDir()` in tests. Do not touch the
real home dir.

## Out of scope

Program files, cobra subcommands, running anything.

## Done when

`go test ./internal/config/` is green. `cmd/root.go` no longer mentions
YAML. `sysup-go --help` still runs (root `Run` may still be empty).
