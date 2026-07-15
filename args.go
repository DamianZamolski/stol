package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	defaultTime = "18:00"
	maxPlayers  = 12
)

type input struct {
	query   string
	hhmm    string
	slots   int
	players []string
}

func parseArgs(args []string) (input, error) {
	var in input
	if len(args) == 0 {
		return in, errors.New("użycie: stol <gra> [godzina] [liczba-graczy] [imiona...]")
	}
	in.query = args[0]
	rest := args[1:]

	if len(rest) > 0 {
		if hhmm, ok := parseTime(rest[0]); ok {
			in.hhmm = hhmm
			rest = rest[1:]
		}
	}
	if in.hhmm == "" {
		in.hhmm = defaultTime
	}
	if len(rest) > 0 {
		if n, err := strconv.Atoi(rest[0]); err == nil && n >= 1 && n <= maxPlayers {
			in.slots = n
			rest = rest[1:]
		}
	}
	for _, p := range rest {
		in.players = append(in.players, titleCase(p))
	}
	return in, nil
}

func parseTime(s string) (string, bool) {
	var h, m int
	switch {
	case regexp.MustCompile(`^\d{1,2}:\d{2}$`).MatchString(s):
		parts := strings.SplitN(s, ":", 2)
		h, _ = strconv.Atoi(parts[0])
		m, _ = strconv.Atoi(parts[1])
	case regexp.MustCompile(`^\d{4}$`).MatchString(s):
		h, _ = strconv.Atoi(s[:2])
		m, _ = strconv.Atoi(s[2:])
	default:
		return "", false
	}
	if h > 23 || m > 59 {
		return "", false
	}
	return fmt.Sprintf("%02d:%02d", h, m), true
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}
