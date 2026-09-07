---
title: "Documentation index"
description: "Entry point for sysup-go docs. Read this first, then open only the files you need."
keywords: [index, toc, docs]
order: 0
---

# Documentation index

Read this file first. Open only the docs your task needs.

| Topic                                              | What it is                                             | Keywords                                        |
| -------------------------------------------------- | ------------------------------------------------------ | ----------------------------------------------- |
| [Design spec](specs/2026-09-07-sysup-go-design.md) | v1 architecture: XDG config vs programs, runner, wraps | design, spec, programs, config, runner, context |
| [Impl overview](specs/impl/00-overview.md)         | Milestone order, shared types, cmd vs internal         | impl, overview, milestones                      |
| [Goal prompt](specs/impl/GOAL.md)                  | Paste-ready work order; parallel package split         | goal, prompt, subagents                         |
| [AGENTS.md](../AGENTS.md)                          | Project conventions                                    | conventions, go, layout, slog, jj               |
| [CI](CI.md)                                        | GitHub Actions, toolchain pins, setup-go               | ci, github-actions, setup-go                    |

## Implementation specs

Approve these before a goal call. One milestone per call, in order.

| Topic                                      | What it is                                | Keywords                         |
| ------------------------------------------ | ----------------------------------------- | -------------------------------- |
| [01 config](specs/impl/01-config.md)       | XDG, viper TOML, defaults, slog           | config, viper, xdg, toml         |
| [02 discovery](specs/impl/02-discovery.md) | programs.toml XOR programs/*.toml         | discovery, programs, sort, alias |
| [03 resolve](specs/impl/03-resolve.md)     | PKG_MANAGER / paru / yay                  | resolve, pkg_manager, goroutine  |
| [04 cli](specs/impl/04-cli.md)             | flags, list, validate, exit codes         | cobra, flags, list, validate     |
| [05 runner](specs/impl/05-runner.md)       | sequential CommandContext                 | runner, context, exec            |
| [06 sudo](specs/impl/06-sudo.md)           | keepalive goroutine                       | sudo, ticker, goroutine          |
| [07 continue](specs/impl/07-continue.md)   | `-c` accumulate errors                    | continue, errors                 |
| [08 waves](specs/impl/08-waves.md)         | consecutive `parallel` + errgroup         | parallel, waves, errgroup        |
| [09 mise](specs/impl/09-mise.md)           | strip shims once; restore iff `did_strip` | mise, did_strip, PATH            |
| [10 cache](specs/impl/10-cache.md)         | end-of-run dir sweep                      | cache, include_dirs, exclude     |
| [11 shutdown](specs/impl/11-shutdown.md)   | countdown; force vs clean-only            | shutdown, force-shutdown         |

## Adding a doc

1. Put it under `docs/` (or `docs/specs/` for dated design notes).
2. Add a row to the matching table (design vs impl).
3. Keep filenames stable. ASCII hyphens only.

Filenames prefixed with `_` are drafts. Do not link them from this table
until they are real.
