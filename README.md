# stol

CLI generating Polish-language Facebook posts for board-game session signups.
Looks games up on [BoardGameGeek](https://boardgamegeek.com), prints the post to
stdout and copies it to the clipboard.

## Build

```sh
make            # -> ~/.local/bin/stol
```

## Usage

```sh
stol '<gra>' [godzina] [liczba-graczy] [imiona...] [--balagra] [--retkinia]
```

Time and player count are optional and recognised by format, not position:

```sh
stol 'ark nova' 1800 4 damian ksenia --balagra
```

`--balagra` / `--retkinia` also open that venue's Facebook events page.

Requires a `BGG_TOKEN` environment variable (BGG XML API2 Bearer auth).
Remembered games and aliases are stored in `$XDG_CONFIG_HOME/stol/config.json`.
