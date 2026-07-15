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
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func ask(prompt string) (string, error) {
	if !interactive() {
		return "", errors.New("potrzebny wybór, a stdin nie jest terminalem")
	}
	fmt.Fprint(os.Stderr, prompt)
	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func confirm(question string) bool {
	ans, err := ask(question + " [T/n] ")
	if err != nil {
		return false
	}
	ans = strings.ToLower(ans)
	return ans == "" || ans == "t" || ans == "tak" || ans == "y"
}

func pickGame(cands []*candidate) (*candidate, error) {
	shown := cands
	if len(shown) > pickerShown {
		shown = shown[:pickerShown]
	}
	for {
		other := len(shown) + 1
		fmt.Fprintf(os.Stderr, "  %d. inna (pokaż wszystkie / podaj link)\n", other)
		for i, c := range slices.Backward(shown) {
			fmt.Fprintf(os.Stderr, "  %d. %s  (%s)\n", i+1, c.URL, c.Year)
		}
		ans, err := ask("> ")
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(ans)
		if err != nil || n < 1 || n > other {
			fmt.Fprintln(os.Stderr, "nie rozumiem, spróbuj jeszcze raz")
			continue
		}
		if n == other {
			if len(shown) < len(cands) {
				shown = cands
				continue
			}
			return askForURL()
		}
		return shown[n-1], nil
	}
}

var bggIDRe = regexp.MustCompile(`(?:boardgame/)?(\d+)`)

func askForURL() (*candidate, error) {
	ans, err := ask("Podaj link BGG lub id (Enter = przerwij): ")
	if err != nil {
		return nil, err
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
	m := bggIDRe.FindStringSubmatch(ans)
	if m == nil {
		return nil, fmt.Errorf("nie znalazłem id gry w: %s", ans)
	}
	cands, err := things([]string{m[1]})
	if err != nil {
		return nil, err
	}
	if len(cands) == 0 {
		return nil, fmt.Errorf("BGG nie zna gry o id %s (albo to dodatek, nie gra)", m[1])
	}
	return cands[0], nil
}
