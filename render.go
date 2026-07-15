package main

import (
	"fmt"
	"strings"
)

func render(g *Game, in input) string {
	names := in.players
	rows := max(in.slots, len(names))
	free := rows - len(names)

	var b strings.Builder
	fmt.Fprintf(&b, "👥 %s\n", freeSlots(free))
	fmt.Fprintf(&b, "🎲 %s\n", g.NamePl)
	fmt.Fprintf(&b, "⏰ %s\n", in.hhmm)
	b.WriteString("\n")
	for i := range rows {
		who := "?"
		if i < len(names) {
			who = names[i]
		}
		fmt.Fprintf(&b, "%d. %s\n", i+1, who)
	}
	b.WriteString("\n")
	b.WriteString(g.URL)
	return b.String()
}

func freeSlots(n int) string {
	switch {
	case n == 0:
		return "Brak wolnych miejsc"
	case n == 1:
		return "1 wolne miejsce"
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
		return fmt.Sprintf("%d wolne miejsca", n)
	default:
		return fmt.Sprintf("%d wolnych miejsc", n)
	}
}
