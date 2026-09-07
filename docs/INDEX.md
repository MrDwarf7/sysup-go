---
title: "Documentation index"
description: "Entry point for sysup-go docs. Read this first, then open only the files you need."
keywords: [index, toc, docs]
order: 0
---

# Documentation index

Read this file first. Open only the docs your task needs.

| Topic | What it is | Keywords |
| --- | --- | --- |
| [Design spec](specs/2026-09-07-sysup-go-design.md) | v1 architecture: XDG config vs programs, runner, cache, flags, errors, context | design, spec, programs, config, runner, context, sudo, mise, cache, skip, continue |
| [AGENTS.md](../AGENTS.md) | Project conventions for humans and agents working in this repo | conventions, go, layout, errors, slog, jj, context |

## Adding a doc

1. Put it under `docs/` (or `docs/specs/` for dated design notes).
2. Add a row to the table above (Topic link, one-line "what it is", keywords).
3. Keep filenames stable. ASCII hyphens only.

Filenames prefixed with `_` are drafts. Do not link them from this table
until they are real.
