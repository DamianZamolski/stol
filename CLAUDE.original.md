# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- `make` — build to `~/.local/bin/stol`
- `go build ./...` — compile
- `gofmt -l stol.go` — format check (empty output = OK)
- `go vet ./...` — vet
- No tests exist.

Run: `stol '<gra>' [godzina] [liczba-graczy] [imiona...] [--balagra] [--retkinia]`
Requires `BGG_TOKEN` env var (BGG XML API2 Bearer auth).

## What it does

CLI that generates a Polish-language Facebook post for board-game session signups.
Output goes to stdout **and** the clipboard via `wl-copy` (Wayland-only). `--balagra`/
`--retkinia` also `xdg-open` that venue's Facebook events page.

## Architecture

Single file, `stol.go`. Flow: `run` → parse args → load config → `resolveGame` → render.

- **Game resolution** (`resolveGame`): tries remembered aliases first (exact, then
  fzf-like `fuzzyScore` subsequence match), else hits BGG. Every resolved game is
  cached back into config under a new alias, so repeat queries skip the network.

- **BGG layer**: `search` → `things`/`thingsBatch` against `/xmlapi2` (batches of ≤20
  ids, filters to `type="boardgame"`, drops expansions/promos). `sortAndFill` ranks
  candidates by **geek rating** (`bayesaverage`, descending — every game has one) then
  fetches canonical URL slugs concurrently from the `api/geekitems` JSON endpoint
  (the XML API omits slugs, and hand-slugging breaks on apostrophes).

- **Config**: `$XDG_CONFIG_HOME/stol/config.json`, holding `games` (by id) and
  `aliases` (user phrase → id). Only written when `dirty`.

- **Interactive pickers** (`pickGame`, `pickPolishName`): lists print *reversed* via
  `slices.Backward` so the best option lands at index 1, nearest the terminal cursor.
  Polish name selection is heuristic-scored (`plScore`, diacritics) but always
  human-confirmed — BGG doesn't tag alt-name languages.

## Conventions

User-facing strings (prompts, errors) are in Polish. Keep them Polish.
