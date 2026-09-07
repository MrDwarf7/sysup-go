---
title: CI
---

# CI

Native preference: version pins live in native files (`go.mod`, `go.env`), not
duplicated in shell.

## Toolchain pinning

| Tool          | Pin file                                          | Composite input                                 | Bump procedure                                                               |
| ------------- | ------------------------------------------------- | ----------------------------------------------- | ---------------------------------------------------------------------------- |
| Go            | `go.mod` (`go 1.27.0`) + `go.env` (`GOTOOLCHAIN`) | `setup-go` `go-version-file` (default `go.mod`) | Edit `go.mod` and `go.env`, run `just t`                                     |
| just          | `setup-go` `inputs.tools` (`just`)                | `setup-go`                                      | Edit `.github/actions/setup-go/action.yml`                                   |
| golangci-lint | `go.mod` `tool` directive                         | `just check`                                    | `go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` |

## Setup actions

- `.github/actions/setup-go` -- `actions/setup-go@v7` from `go.mod`, module cache, just, optional MinGW on Windows for `-race`
- `.github/actions/checkout` -- thin wrapper around `actions/checkout@v7`

Workflows call `actions/checkout@v7` then `./.github/actions/setup-go`.

## Recipes CI runs

- `just format-check` -- `gofmt -l` must be empty
- `just check` -- golangci-lint
- `just t` -- race tests
- `just ba` / `just br` -- debug+release / release
- `just vuln` -- govulncheck

`just be` / `just re` are local experimental builds. They are not CI gates.
