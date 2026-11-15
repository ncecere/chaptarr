// Package download talks to download clients: it hands them a release and
// reports how the download is going.
package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ncecere/chaptarr/internal/model"
)

// State is a download's condition as the client reports it.
type State string

const (
	StateQueued      State = "queued"
	StateDownloading State = "downloading"
	StatePaused      State = "paused"
	StateCompleted   State = "completed"
	StateFailed      State = "failed"
)

type Item struct {
	State      State
	Progress   float64 // 0..1
	SizeBytes  int64
	ETASeconds *int64
	// Path is the finished download: a directory, or a single file.
	Path  string
	Error string
}

// ErrGone means the client no longer knows the download.
var ErrGone = errors.New("download not found in the client")

// Payload is a release ready to hand to a client: either the fetched
// .torrent/.nzb file or, for torrents, a magnet link.
type Payload struct {
	Title    string
	Magnet   string
	File     []byte
	Filename string
	// InfoHash is the lowercase hex v1 info-hash (torrents only).
	InfoHash string
}

type Client interface {
	Test(ctx context.Context) error
	// Add queues the payload and returns the client's id for it.
	Add(ctx context.Context, p Payload) (string, error)
	Status(ctx context.Context, id string) (*Item, error)
	Remove(ctx context.Context, id string, deleteData bool) error
}

// New builds the client for a stored configuration.
func New(c model.DownloadClient) (Client, error) {
	base := strings.TrimRight(strings.TrimSpace(c.URL), "/")
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("invalid download client URL %q", c.URL)
	}
	switch c.Type {
	case "qbittorrent":
		return newQBittorrent(base, c), nil
	case "sabnzbd":
		return &sabnzbd{base: base, apiKey: c.APIKey, category: c.Category, http: &http.Client{Timeout: 30 * time.Second}}, nil
	}
	return nil, fmt.Errorf("unknown download client type %q", c.Type)
}

// Protocol returns the protocol a client type handles.
func Protocol(clientType string) string {
	if clientType == "sabnzbd" {
		return model.ProtocolUsenet
	}
	return model.ProtocolTorrent
}

// Fetch downloads a release from an indexer. Torznab links often redirect to
// a magnet URI, which is returned as the payload instead of being followed.
func Fetch(ctx context.Context, protocol, title, link string) (Payload, error) {
	p := Payload{Title: title}
	if strings.HasPrefix(link, "magnet:") {
		return magnetPayload(p, link)
	}
	hc := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme == "magnet" || len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return p, fmt.Errorf("invalid release link")
	}
	req.Header.Set("User-Agent", "Chaptarr")
	res, err := hc.Do(req)
	if err != nil {
		if ue, ok := err.(*url.Error); ok {
			err = ue.Err // the link usually embeds an API key
		}
		return p, fmt.Errorf("could not fetch the release from the indexer: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 && res.StatusCode < 400 {
		if loc := res.Header.Get("Location"); strings.HasPrefix(loc, "magnet:") {
			return magnetPayload(p, loc)
		}
	}
	if res.StatusCode != http.StatusOK {
		return p, fmt.Errorf("indexer returned %s for the release download", res.Status)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return p, fmt.Errorf("could not read the release from the indexer: %w", err)
	}
	p.File = data
	if protocol == model.ProtocolUsenet {
		if !strings.Contains(string(data[:min(len(data), 2048)]), "<nzb") {
			return p, errors.New("indexer did not return an NZB file")
		}
		p.Filename = safeName(title) + ".nzb"
		return p, nil
	}
	hash, err := torrentInfoHash(data)
	if err != nil {
		return p, fmt.Errorf("indexer did not return a valid torrent: %w", err)
	}
	p.InfoHash = hash
	p.Filename = safeName(title) + ".torrent"
	return p, nil
}

func magnetPayload(p Payload, link string) (Payload, error) {
	hash, err := magnetInfoHash(link)
	if err != nil {
		return p, err
	}
	p.Magnet, p.InfoHash = link, hash
	return p, nil
}

func safeName(s string) string {
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) || r < 32 {
			return '_'
		}
		return r
	}, s)
	if len(s) > 150 {
		s = s[:150]
	}
	if strings.Trim(s, " ._") == "" {
		return "release"
	}
	return s
}
