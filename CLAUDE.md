# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repo.

## Commands

- `make` — build to `~/.local/bin/stol`
- `go build stol.go` — compile
- `gofmt -l stol.go` — format check (empty output = OK)
- `go vet stol.go` — vet
- No tests. No `go.mod` (stdlib-only, single-file build).

Run: `stol '<gra>' [godzina] [liczba-graczy] [imiona...] [--balagra] [--retkinia]`
Need `BGG_TOKEN` env var (BGG XML API2 Bearer auth).

## What it does

CLI make Polish Facebook post for board-game session signups. Output to stdout **and** clipboard via `wl-copy` (Wayland-only). `--balagra`/`--retkinia` also `xdg-open` venue Facebook events page.

## Architecture

Single file `stol.go`. Flow: `run` → parse args → load config → `resolveGame` → render.

- **Game resolution** (`resolveGame`): try remembered aliases first (exact, then fzf-like `fuzzyScore` subsequence match), else hit BGG. Every resolved game cached back to config under new alias → repeat queries skip network.

- **BGG layer**: `search` → `things`/`thingsBatch` against `/xmlapi2` (batches ≤20 ids, filter to `type="boardgame"`, drop expansions/promos). `sortAndFill` ranks candidates by **geek rating** (`bayesaverage`, descending — every game has one) then fetches canonical URL slugs concurrently from `api/geekitems` JSON endpoint (XML API omits slugs; hand-slugging breaks on apostrophes).

- **Config**: `$XDG_CONFIG_HOME/stol/config.json`, holds `games` (by id) and `aliases` (user phrase → id). Written only when `dirty`.

- **Interactive pickers** (`pickGame`, `pickPolishName`): lists print *reversed* via `slices.Backward` so best option lands index 1, nearest cursor. Polish name selection heuristic-scored (`plScore`, diacritics) but always human-confirmed — BGG no tag alt-name languages.

## Conventions

User-facing strings (prompts, errors) in Polish. Keep Polish.