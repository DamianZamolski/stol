package main

import (
	"fmt"
	"os"
	"strconv"
)

func resolveGame(cfg *Config, query string) (*Game, error) {
	q := normalize(query)

	if id, ok := cfg.Aliases[q]; ok {
		if g, ok := cfg.Games[strconv.Itoa(id)]; ok {
			return g, nil
		}
	}

	if alias, id, ok := fuzzyAlias(cfg, q); ok {
		if g, ok := cfg.Games[strconv.Itoa(id)]; ok {
			if confirm(fmt.Sprintf("Czy chodzi o: %s (zapamiętane jako %q)?", g.NamePl, alias)) {
				cfg.remember(g, query)
				return g, nil
			}
		}
	}

	cands, err := search(query)
	if err != nil {
		return nil, err
	}

	var chosen *candidate
	if len(cands) == 0 {
		fmt.Fprintf(os.Stderr, "Nie znaleziono gry: %s\n", query)
		chosen, err = askForURL()
	} else {
		chosen, err = pickGame(cands)
	}
	if err != nil {
		return nil, err
	}

	namePl, err := pickPolishName(chosen)
	if err != nil {
		return nil, err
	}

	g := &Game{ID: chosen.ID, Name: chosen.Name, NamePl: namePl, URL: chosen.URL}
	cfg.remember(g, query)
	return g, nil
}

func fuzzyAlias(cfg *Config, q string) (string, int, bool) {
	bestScore, bestAlias, bestID := 0, "", 0
	for alias, id := range cfg.Aliases {
		if s := fuzzyScore(q, alias); s > bestScore {
			bestScore, bestAlias, bestID = s, alias, id
		}
	}
	if bestScore < len([]rune(q)) {
		return "", 0, false
	}
	return bestAlias, bestID, true
}

func fuzzyScore(needle, haystack string) int {
	n, h := []rune(needle), []rune(haystack)
	if len(n) == 0 {
		return 0
	}
	score, ni, prev := 0, 0, -2
	for hi := 0; hi < len(h) && ni < len(n); hi++ {
		if h[hi] != n[ni] {
			continue
		}
		score += 1
		if hi == prev+1 {
			score += 3
		}
		if hi == 0 || h[hi-1] == ' ' {
			score += 2
		}
		prev, ni = hi, ni+1
	}
	if ni < len(n) {
		return 0
	}
	return score
}
