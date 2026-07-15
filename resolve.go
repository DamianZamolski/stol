package main

import (
	"fmt"
	"os"
	"strconv"
)

func resolveGame(config *Config, query string) (*Game, Error) {
	normalized := normalize(query)

	if id, ok := config.Aliases[normalized]; ok {
		if g, ok := config.Games[strconv.Itoa(id)]; ok {
			return g, nil
		}
	}

	if alias, id, ok := fuzzyAlias(config, normalized); ok {
		if g, ok := config.Games[strconv.Itoa(id)]; ok {
			if confirm(fmt.Sprintf("Czy chodzi o: %s (zapamiętane jako %q)?", g.NamePl, alias)) {
				config.remember(g, query)
				return g, nil
			}
		}
	}

	candidates, error := search(query)
	if error != nil {
		return nil, error
	}

	var chosen *candidate
	if len(candidates) == 0 {
		fmt.Fprintf(os.Stderr, "Nie znaleziono gry: %s\n", query)
		chosen, error = askForUrl()
	} else {
		chosen, error = pickGame(candidates)
	}
	if error != nil {
		return nil, error
	}

	namePl, error := pickPolishName(chosen)
	if error != nil {
		return nil, error
	}

	g := &Game{Id: chosen.Id, Name: chosen.Name, NamePl: namePl, Url: chosen.Url}
	config.remember(g, query)
	return g, nil
}

func fuzzyAlias(config *Config, normalized string) (string, int, bool) {
	bestScore, bestAlias, bestId := 0, "", 0
	for alias, id := range config.Aliases {
		if s := fuzzyScore(normalized, alias); s > bestScore {
			bestScore, bestAlias, bestId = s, alias, id
		}
	}
	if bestScore < len([]rune(normalized)) {
		return "", 0, false
	}
	return bestAlias, bestId, true
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
