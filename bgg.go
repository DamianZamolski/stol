package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	bggAPI   = "https://boardgamegeek.com/xmlapi2"
	bggItems = "https://boardgamegeek.com/api/geekitems"
	bggWeb   = "https://boardgamegeek.com"
)

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
