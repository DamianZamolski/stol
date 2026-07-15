package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
)

var (
	plDiacritics = regexp.MustCompile(`[ąćęłńóśźżĄĆĘŁŃÓŚŹŻ]`)
)

func plScore(s string) int {
	score := 0
	if plDiacritics.MatchString(s) {
		score += 2
	}
	return score
}

type plOption struct {
	name string
	orig bool
}

func pickPolishName(c *candidate) (string, Error) {
	full := []plOption{{c.Name, true}}
	for _, a := range c.Alts {
		full = append(full, plOption{a, false})
	}
	sort.SliceStable(full, func(i, j int) bool { return plScore(full[i].name) > plScore(full[j].name) })

	shown := full
	if len(shown) > pickerShown {
		shown = shown[:pickerShown]
	}

	for {
		fmt.Fprintf(os.Stderr, "\nNazwa polska dla: %s\n", c.Name)
		other := len(shown) + 1
		fmt.Fprintf(os.Stderr, "  %d. inna (pokaż wszystkie)\n", other)
		for i, s := range slices.Backward(shown) {
			tag := ""
			if s.orig {
				tag = "  (oryginalna)"
			}
			fmt.Fprintf(os.Stderr, "  %d. %s%s\n", i+1, s.name, tag)
		}
		ans, error := ask("> ")
		if error != nil {
			return "", error
		}
		n, error := strconv.Atoi(ans)
		if error != nil || n < 1 || n > other {
			fmt.Fprintln(os.Stderr, "nie rozumiem, spróbuj jeszcze raz")
			continue
		}
		if n == other {
			if len(shown) < len(full) {
				shown = full
			}
			continue
		}
		return shown[n-1].name, nil
	}
}
