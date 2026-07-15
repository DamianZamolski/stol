# stol

CLI make Polish Facebook posts for board-game session signups. Look up game on [BoardGameGeek](https://boardgamegeek.com), print post to stdout.

## Build

```sh
make            # -> ~/.local/bin/stol
```

## Usage

```sh
stol '<gra>' [godzina] [liczba-graczy] [imiona...]
```

Time + player count optional. Recognised by format, not position:

```sh
stol 'ark nova' 1800 4 damian ksenia
```

Need `BGG_TOKEN` env var (BGG XML API2 Bearer auth). Remembered games + aliases stored in `$XDG_CONFIG_HOME/stol/config.json`.