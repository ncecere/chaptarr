// Package indexer queries Torznab and Newznab feeds (Prowlarr, Jackett,
// NZBHydra, or an indexer directly) and scores what comes back against a book.
package indexer

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ncecere/chaptarr/internal/model"
)

type Client struct {
	HTTP *http.Client
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 45 * time.Second}}
}

type rss struct {
	Channel struct {
		Items []item `xml:"item"`
	} `xml:"channel"`
}

type item struct {
	Title     string `xml:"title"`
	GUID      string `xml:"guid"`
	Link      string `xml:"link"`
	Comments  string `xml:"comments"`
	PubDate   string `xml:"pubDate"`
	Size      int64  `xml:"size"`
	Enclosure struct {
		URL    string `xml:"url,attr"`
		Length int64  `xml:"length,attr"`
	} `xml:"enclosure"`
	// Matches both torznab:attr and newznab:attr.
	Attrs []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	} `xml:"attr"`
}

type apiError struct {
	XMLName     xml.Name `xml:"error"`
	Code        string   `xml:"code,attr"`
	Description string   `xml:"description,attr"`
}

func endpoint(ix model.Indexer, params url.Values) (string, error) {
	u, err := url.Parse(strings.TrimSpace(ix.URL))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("invalid indexer URL %q", ix.URL)
	}
	// Accept both "http://host/1/api" and a bare newznab host.
	if !strings.HasSuffix(strings.TrimRight(u.Path, "/"), "/api") {
		u.Path = strings.TrimRight(u.Path, "/") + "/api"
	}
	q := u.Query()
	for k, v := range params {
		q[k] = v
	}
	if ix.APIKey != "" {
		q.Set("apikey", ix.APIKey)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *Client) fetch(ctx context.Context, ix model.Indexer, params url.Values) ([]byte, error) {
	u, err := endpoint(ix, params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Chaptarr")
	res, err := c.HTTP.Do(req)
	if err != nil {
		// The URL carries the API key; keep it out of logs and the UI.
		if ue, ok := err.(*url.Error); ok {
			err = ue.Err
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("indexer returned %s", res.Status)
	}
	var apiErr apiError
	if xml.Unmarshal(body, &apiErr) == nil && apiErr.Description != "" {
		return nil, fmt.Errorf("indexer error %s: %s", apiErr.Code, apiErr.Description)
	}
	return body, nil
}

// Test checks that the feed answers a capabilities request.
func (c *Client) Test(ctx context.Context, ix model.Indexer) error {
	body, err := c.fetch(ctx, ix, url.Values{"t": {"caps"}})
	if err != nil {
		return err
	}
	var caps struct {
		XMLName xml.Name `xml:"caps"`
	}
	if err := xml.Unmarshal(body, &caps); err != nil {
		return fmt.Errorf("response is not a Torznab/Newznab capabilities document")
	}
	return nil
}

// Search runs one free-text query and returns unscored releases.
func (c *Client) Search(ctx context.Context, ix model.Indexer, query string) ([]model.Release, error) {
	params := url.Values{"t": {"search"}, "q": {query}, "limit": {"100"}, "extended": {"1"}}
	if cats := strings.TrimSpace(ix.Categories); cats != "" {
		params.Set("cat", cats)
	}
	body, err := c.fetch(ctx, ix, params)
	if err != nil {
		return nil, err
	}
	var feed rss
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("could not parse indexer response: %w", err)
	}
	protocol := model.ProtocolTorrent
	if ix.Type == "newznab" {
		protocol = model.ProtocolUsenet
	}
	out := make([]model.Release, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		r := model.Release{
			Title:       strings.TrimSpace(it.Title),
			Indexer:     ix.Name,
			IndexerID:   ix.ID,
			Priority:    ix.Priority,
			Protocol:    protocol,
			GUID:        strings.TrimSpace(it.GUID),
			DownloadURL: strings.TrimSpace(it.Enclosure.URL),
			SizeBytes:   it.Size,
			InfoURL:     strings.TrimSpace(it.Comments),
			Rejections:  []string{},
		}
		if r.DownloadURL == "" {
			r.DownloadURL = strings.TrimSpace(it.Link)
		}
		if r.SizeBytes == 0 {
			r.SizeBytes = it.Enclosure.Length
		}
		attrs := map[string]string{}
		for _, a := range it.Attrs {
			attrs[strings.ToLower(a.Name)] = a.Value
		}
		if v, err := strconv.ParseInt(attrs["size"], 10, 64); err == nil && v > 0 {
			r.SizeBytes = v
		}
		if protocol == model.ProtocolTorrent {
			if v, err := strconv.Atoi(attrs["seeders"]); err == nil {
				r.Seeders = &v
				if p, err := strconv.Atoi(attrs["peers"]); err == nil {
					l := p - v
					if l < 0 {
						l = 0
					}
					r.Leechers = &l
				}
			}
			if m := attrs["magneturl"]; m != "" && r.DownloadURL == "" {
				r.DownloadURL = m
			}
		}
		if r.InfoURL == "" && strings.HasPrefix(r.GUID, "http") {
			r.InfoURL = r.GUID
		}
		if r.GUID == "" {
			r.GUID = r.DownloadURL
		}
		if t, err := parseDate(it.PubDate); err == nil {
			if d := int(time.Since(t).Hours() / 24); d > 0 {
				r.AgeDays = d
			}
		}
		if r.Title == "" || r.DownloadURL == "" {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date %q", s)
}
