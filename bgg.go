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
	bggApi   = "https://boardgamegeek.com/xmlapi2"
	bggItems = "https://boardgamegeek.com/api/geekitems"
	bggWeb   = "https://boardgamegeek.com"
)

func token() (string, Error) {
	t := os.Getenv("BGG_TOKEN")
	if t == "" {
		return "", errors.New("brak BGG_TOKEN w środowisku")
	}
	return t, nil
}

func bggGet(url string, auth bool) ([]byte, Error) {
	request, error := http.NewRequest(http.MethodGet, url, nil)
	if error != nil {
		return nil, error
	}
	request.Header.Set("User-Agent", "stol/1.0")
	if auth {
		t, error := token()
		if error != nil {
			return nil, error
		}
		request.Header.Set("Authorization", "Bearer "+t)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	response, error := client.Do(request)
	if error != nil {
		return nil, error
	}
	defer response.Body.Close()
	body, error := io.ReadAll(response.Body)
	if error != nil {
		return nil, error
	}
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("BGG odrzuciło token (401) — sprawdź BGG_TOKEN")
		}
		return nil, fmt.Errorf("BGG %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func search(query string) ([]*candidate, Error) {
	raw, error := bggGet(fmt.Sprintf("%s/search?query=%s&type=boardgame", bggApi, urlEscape(query)), true)
	if error != nil {
		return nil, error
	}
	var searchResponse struct {
		Items []struct {
			Id int `xml:"id,attr"`
		} `xml:"item"`
	}
	if error := xml.Unmarshal(raw, &searchResponse); error != nil {
		return nil, error
	}
	ids := make([]string, 0, len(searchResponse.Items))
	for _, it := range searchResponse.Items {
		ids = append(ids, strconv.Itoa(it.Id))
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return things(ids)
}

func things(ids []string) ([]*candidate, Error) {
	const batch = 20
	var out []*candidate
	for start := 0; start < len(ids); start += batch {
		end := min(start+batch, len(ids))
		part, error := thingsBatch(ids[start:end])
		if error != nil {
			return nil, error
		}
		out = append(out, part...)
	}
	sortAndFill(out)
	return out, nil
}

func thingsBatch(ids []string) ([]*candidate, Error) {
	raw, error := bggGet(fmt.Sprintf("%s/thing?id=%s&stats=1", bggApi, strings.Join(ids, ",")), true)
	if error != nil {
		return nil, error
	}
	var thingResponse struct {
		Items []struct {
			Type  string `xml:"type,attr"`
			Id    int    `xml:"id,attr"`
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
	if error := xml.Unmarshal(raw, &thingResponse); error != nil {
		return nil, error
	}

	var out []*candidate
	for _, it := range thingResponse.Items {
		if it.Type != "boardgame" {
			continue
		}
		c := &candidate{Id: it.Id, Year: it.Year.Value}
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
		if v, error := strconv.ParseFloat(it.Stats.Ratings.Bayesaverage.Value, 64); error == nil {
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
	fillUrls(out)
}

func fillUrls(candidates []*candidate) {
	var waitGroup sync.WaitGroup
	semaphore := make(chan struct{}, 8)
	for _, c := range candidates {
		waitGroup.Add(1)
		go func(c *candidate) {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			c.Url = gameUrl(c.Id, c.Name)
		}(c)
	}
	waitGroup.Wait()
}

func gameUrl(id int, name string) string {
	raw, error := bggGet(fmt.Sprintf("%s?objecttype=thing&objectid=%d", bggItems, id), false)
	if error == nil {
		var geekItem struct {
			Item struct {
				Href string `json:"href"`
			} `json:"item"`
		}
		if json.Unmarshal(raw, &geekItem) == nil && geekItem.Item.Href != "" {
			return bggWeb + geekItem.Item.Href
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
