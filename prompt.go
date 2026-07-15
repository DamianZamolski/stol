package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const pickerShown = 3

var stdin = bufio.NewReader(os.Stdin)

func interactive() bool {
	fi, error := os.Stdin.Stat()
	return error == nil && fi.Mode()&os.ModeCharDevice != 0
}

func ask(prompt string) (string, Error) {
	if !interactive() {
		return "", errors.New("potrzebny wybór, a stdin nie jest terminalem")
	}
	fmt.Fprint(os.Stderr, prompt)
	line, error := stdin.ReadString('\n')
	if error != nil && line == "" {
		return "", error
	}
	return strings.TrimSpace(line), nil
}

func confirm(question string) bool {
	ans, error := ask(question + " [T/n] ")
	if error != nil {
		return false
	}
	ans = strings.ToLower(ans)
	return ans == "" || ans == "t" || ans == "tak" || ans == "y"
}

func pickGame(candidates []*candidate) (*candidate, Error) {
	shown := candidates
	if len(shown) > pickerShown {
		shown = shown[:pickerShown]
	}
	for {
		other := len(shown) + 1
		fmt.Fprintf(os.Stderr, "  %d. inna (pokaż wszystkie / podaj link)\n", other)
		for i, c := range slices.Backward(shown) {
			fmt.Fprintf(os.Stderr, "  %d. %s  (%s)\n", i+1, c.Url, c.Year)
		}
		ans, error := ask("> ")
		if error != nil {
			return nil, error
		}
		n, error := strconv.Atoi(ans)
		if error != nil || n < 1 || n > other {
			fmt.Fprintln(os.Stderr, "nie rozumiem, spróbuj jeszcze raz")
			continue
		}
		if n == other {
			if len(shown) < len(candidates) {
				shown = candidates
				continue
			}
			return askForUrl()
		}
		return shown[n-1], nil
	}
}

var bggIdRe = regexp.MustCompile(`(?:boardgame/)?(\d+)`)

func askForUrl() (*candidate, Error) {
	ans, error := ask("Podaj link BGG lub id (Enter = przerwij): ")
	if error != nil {
		return nil, error
	}
	if ans == "" {
		return nil, errors.New("przerwano")
	}
	if !strings.HasPrefix(ans, "http") && !regexp.MustCompile(`^\d+$`).MatchString(ans) {
		return nil, fmt.Errorf("to nie jest link BGG ani id: %s", ans)
	}
	if strings.HasPrefix(ans, "http") && !strings.Contains(ans, "boardgamegeek.com") {
		return nil, fmt.Errorf("to nie jest link BGG: %s", ans)
	}
	m := bggIdRe.FindStringSubmatch(ans)
	if m == nil {
		return nil, fmt.Errorf("nie znalazłem id gry w: %s", ans)
	}
	candidates, error := things([]string{m[1]})
	if error != nil {
		return nil, error
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("BGG nie zna gry o id %s (albo to dodatek, nie gra)", m[1])
	}
	return candidates[0], nil
}
