package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	bggAPI      = "https://boardgamegeek.com/xmlapi2"
	bggItems    = "https://boardgamegeek.com/api/geekitems"
	bggWeb      = "https://boardgamegeek.com"
	defaultTime = "18:00"
	maxPlayers  = 12
	pickerShown = 3
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "błąd:", err)
		os.Exit(1)
	}
}

func run(argv []string) error {
	in, err := parseArgs(argv)
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	game, err := resolveGame(cfg, in.query)
	if err != nil {
		return err
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}

	post := render(game, in)
	fmt.Println(post)
	return nil
}

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

func render(g *Game, in input) string {
	names := in.players
	rows := max(in.slots, len(names))
	free := rows - len(names)

	var b strings.Builder
	fmt.Fprintf(&b, "👥 %s\n", freeSlots(free))
	fmt.Fprintf(&b, "⏰ %s\n", in.hhmm)
	fmt.Fprintf(&b, "🎲 %s\n", g.NamePl)
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

type Game struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	NamePl string `json:"namePl"`
	URL    string `json:"url"`
}

type Config struct {
	Games   map[string]*Game `json:"games"`
	Aliases map[string]int   `json:"aliases"`
	path    string
	dirty   bool
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "stol", "config.json"), nil
}

func loadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	cfg := &Config{Games: map[string]*Game{}, Aliases: map[string]int{}, path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("uszkodzony %s: %w", path, err)
	}
	if cfg.Games == nil {
		cfg.Games = map[string]*Game{}
	}
	if cfg.Aliases == nil {
		cfg.Aliases = map[string]int{}
	}
	return cfg, nil
}

func saveConfig(cfg *Config) error {
	if !cfg.dirty {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(cfg.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.path, append(data, '\n'), 0o644)
}

func (c *Config) remember(g *Game, alias string) {
	c.Games[strconv.Itoa(g.ID)] = g
	c.Aliases[normalize(alias)] = g.ID
	c.dirty = true
}

func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

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

type candidate struct {
	ID   int
	Name string
	Alts []string
	Year string
	Geek float64
	URL  string
}

func token() (string, error) {
	t := os.Getenv("BGG_TOKEN")
	if t == "" {
		return "", errors.New("brak BGG_TOKEN w środowisku")
	}
	return t, nil
}

func bggGet(url string, auth bool) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "stol/1.0")
	if auth {
		t, err := token()
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+t)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("BGG odrzuciło token (401) — sprawdź BGG_TOKEN")
		}
		return nil, fmt.Errorf("BGG %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func search(query string) ([]*candidate, error) {
	raw, err := bggGet(fmt.Sprintf("%s/search?query=%s&type=boardgame", bggAPI, urlEscape(query)), true)
	if err != nil {
		return nil, err
	}
	var sr struct {
		Items []struct {
			ID int `xml:"id,attr"`
		} `xml:"item"`
	}
	if err := xml.Unmarshal(raw, &sr); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(sr.Items))
	for _, it := range sr.Items {
		ids = append(ids, strconv.Itoa(it.ID))
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return things(ids)
}

func things(ids []string) ([]*candidate, error) {
	const batch = 20
	var out []*candidate
	for start := 0; start < len(ids); start += batch {
		end := min(start+batch, len(ids))
		part, err := thingsBatch(ids[start:end])
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	sortAndFill(out)
	return out, nil
}

func thingsBatch(ids []string) ([]*candidate, error) {
	raw, err := bggGet(fmt.Sprintf("%s/thing?id=%s&stats=1", bggAPI, strings.Join(ids, ",")), true)
	if err != nil {
		return nil, err
	}
	var tr struct {
		Items []struct {
			Type  string `xml:"type,attr"`
			ID    int    `xml:"id,attr"`
			Names []struct {
				Type  string `xml:"type,attr"`
				Value string `xml:"value,attr"`
			} `xml:"name"`
			Year struct {
				Value string `xml:"value,attr"`
			} `xml:"yearpublished"`
			Stats struct {
				Ratings struct {
					Bayesaverage struct {
						Value string `xml:"value,attr"`
					} `xml:"bayesaverage"`
				} `xml:"ratings"`
			} `xml:"statistics"`
		} `xml:"item"`
	}
	if err := xml.Unmarshal(raw, &tr); err != nil {
		return nil, err
	}

	var out []*candidate
	for _, it := range tr.Items {
		if it.Type != "boardgame" {
			continue
		}
		c := &candidate{ID: it.ID, Year: it.Year.Value}
		for _, n := range it.Names {
			if n.Type == "primary" {
				c.Name = n.Value
			} else {
				c.Alts = append(c.Alts, n.Value)
			}
		}
		if c.Name == "" {
			continue
		}
		if v, err := strconv.ParseFloat(it.Stats.Ratings.Bayesaverage.Value, 64); err == nil {
			c.Geek = v
		}
		out = append(out, c)
	}
	return out, nil
}

func sortAndFill(out []*candidate) {
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Geek > out[j].Geek
	})
	fillURLs(out)
}

func fillURLs(cands []*candidate) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, c := range cands {
		wg.Add(1)
		go func(c *candidate) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c.URL = gameURL(c.ID, c.Name)
		}(c)
	}
	wg.Wait()
}

func gameURL(id int, name string) string {
	raw, err := bggGet(fmt.Sprintf("%s?objecttype=thing&objectid=%d", bggItems, id), false)
	if err == nil {
		var gi struct {
			Item struct {
				Href string `json:"href"`
			} `json:"item"`
		}
		if json.Unmarshal(raw, &gi) == nil && gi.Item.Href != "" {
			return bggWeb + gi.Item.Href
		}
	}
	return fmt.Sprintf("%s/boardgame/%d/%s", bggWeb, id, slugify(name))
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "’", "")
	return strings.Trim(nonSlug.ReplaceAllString(s, "-"), "-")
}

func urlEscape(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "+")
}

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

func pickPolishName(c *candidate) (string, error) {
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
		ans, err := ask("> ")
		if err != nil {
			return "", err
		}
		n, err := strconv.Atoi(ans)
		if err != nil || n < 1 || n > other {
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
