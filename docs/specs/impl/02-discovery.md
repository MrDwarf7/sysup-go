---
title: "02 discovery"
description: "Load programs.toml XOR programs/*.toml, lexical sort, unique name/alias."
keywords: [impl, discovery, programs, toml, sort, alias]
order: 22
---

# 02 discovery

Turn the XDG programs file or directory into an ordered `[]Spec`.
This is the user registry. Files are execution specs only.

## Depends on

01 (config dir path). Does not use viper.

## Files

- Create: `internal/program/spec.go`
- Create: `internal/program/load.go`
- Create: `internal/program/error.go`
- Create: `internal/program/load_test.go`
- Create: `internal/program/testdata/` fixtures as needed

Decode with `github.com/pelletier/go-toml/v2` (already in go.mod).
Disallow unknown fields.

## Types

```go
package program

type Spec struct {
    Name        string   `toml:"name"`
    Alias       string   `toml:"alias"`
    Description string   `toml:"description"`
    Optional    bool     `toml:"optional"`
    Parallel    bool     `toml:"parallel"`
    Command     []string `toml:"command"`
    Source      string   `toml:"-"`
}

type Error struct {
    Op   string
    Path string
    Err  error
}

type SkipError struct {
    Token string
}

func Load(dir string) ([]Spec, error)
func Filter(specs []Spec, skip []string) ([]Spec, error)
```

`Load(dir)` looks at `dir` (the sysup-go config dir, not a programs
subdir path the caller invents twice):

1. If `dir/programs.toml` exists as a file: load it, ignore
   `dir/programs/`. Array order. Table name `[[program]]`.
2. Else if `dir/programs/` is a directory: glob `*.toml`, lexical
   sort of **filename** (`slices.Sort`), load each file as one
   `Spec` (top-level keys, not `[[program]]`).
3. Else: error "nothing to run", `Op: "discover"`.

A single file in `programs/` is one program. Extra `[[program]]` in a
per-file doc is an error (unknown/wrong shape).

Empty `programs.toml` (`[[program]]` missing / zero tables): error,
nothing to run.

Empty `programs/` directory (no `*.toml`): same error.

## Validation (fail fast, before any exec)

- `name` required, non-empty, unique. Collision: error cites **both**
  `Source` paths.
- `alias` optional. If set, unique among aliases. Collision cites both
  paths. Empty alias is "no alias", not a token.
- `command` required, len >= 1, every element non-empty.
- `optional` and `parallel` default false.
- `[retries] always` / `max_attempts` optional. 0 max is unset.
  Global `[retries].max_attempts` > 0 is a hard cap on the recipe.
- No `kind` field. Presence of `kind` is an unknown-field error.

Lexical sort: `10-mirror.toml` < `20-pacman.toml`. `9-x.toml` >
`10-x.toml`. Do not parse integers. Do not require a numeric prefix;
unprefixed names sort by the same string compare and will sit wherever
ASCII puts them.

## Filter

`Filter` applies `--skip`. Each token must match a `Name` or `Alias`.
Unknown token => `SkipError{Token}` (cmd maps to exit 3 in 04).
Matching is exact string, case-sensitive. Duplicates in `--skip` are
fine (idempotent). Filter does not reorder.

Quirk: skipping a name also drops specs named `name-*`. Skipping
`mirror` (or alias `m`) drops `mirror-stage` and friends so a split
recipe is not left half-run. Skipping only the child does not skip
the parent. The cut is `parent + "-"`; `mirrors` is not a child of
`mirror`.

## Tests

| Case | Expect |
| --- | --- |
| neither file nor dir | error, nothing to run |
| `programs.toml` AND `programs/foo.toml` | file wins, dir ignored |
| two files `20-b.toml`, `10-a.toml` | order a then b |
| `9-z.toml` vs `10-a.toml` | `10-a` first (lexical) |
| duplicate names | error, both paths in `Error()` text |
| duplicate aliases | same |
| missing command | error |
| unknown field `kind = "exec"` | error |
| `Filter` by name and by alias | dropped from result, others kept |
| `Filter` unknown token | `SkipError` |
| `[[program]]` array of 2 in programs.toml | order preserved |

Use `t.TempDir()` trees. Do not read real XDG.

## Out of scope

Exec, PATH checks, cobra. `optional` missing-binary behaviour is 05.

## Done when

`go test ./internal/program/` is green. No cobra changes required
except what 04 will add.
