# CLAUDE.md

Guide for Claude Code (claude.ai/code) in this repo.

## Commands

- `make` — build to `./bin/stol`
- `make install` — install to `~/.local/bin/stol` (`PREFIX` retargets)
- `make lint` — golangci-lint (only linter)
- `make test` — lint then `go test -race ./...`
- `go build .` — compile
- `gofmt -l .` — format check (empty output = OK)
- No tests yet. `go.mod` present (module `stol`, stdlib-only — no deps).

Run: `stol '<gra>' [godzina] [liczba-graczy] [imiona...]`
Need `BGG_TOKEN` env var (BGG XML API2 Bearer auth).

## What it does

CLI make Polish Facebook post for board-game session signups. Output stdout.

## Architecture

One export/unit per file, all `package main`: `main.go` (`run`), `args.go`, `render.go`, `game.go`, `config.go`, `resolve.go`, `candidate.go`, `bgg.go`, `prompt.go` (pickers + `ask`/`confirm`), `polish.go`. Flow: `run` → parse args → load config → `resolveGame` → render.

- **Game resolution** (`resolveGame`): try remembered aliases first (exact, then fzf-like `fuzzyScore` subsequence match), else hit BGG. Resolved game cached back to config under new alias → repeat queries skip network.

- **BGG layer**: `search` → `things`/`thingsBatch` against `/xmlapi2` (batches ≤20 ids, filter `type="boardgame"`, drop expansions/promos). `sortAndFill` ranks candidates by **geek rating** (`bayesaverage`, descending — every game has one), then fetches canonical URL slugs concurrently from `api/geekitems` JSON endpoint (XML API omits slugs; hand-slugging breaks on apostrophes).

- **Config**: `$XDG_CONFIG_HOME/stol/config.json`, holds `games` (by id) and `aliases` (user phrase → id). Written only when `dirty`.

- **Interactive pickers** (`pickGame`, `pickPolishName`): lists print *reversed* via `slices.Backward` so best option lands index 1, nearest cursor. Polish name selection heuristic-scored (`plScore`, diacritics) but always human-confirmed — BGG no tag alt-name languages.

## Conventions

User-facing strings (prompts, errors) Polish. Keep Polish.