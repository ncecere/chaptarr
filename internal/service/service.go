// Package service holds the application logic: following authors, searching
// indexers, grabbing releases, tracking downloads and importing them.
package service

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ncecere/chaptarr/internal/download"
	"github.com/ncecere/chaptarr/internal/indexer"
	"github.com/ncecere/chaptarr/internal/metadata"
	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/store"
)

// Error is a failure with a message meant for the user and the HTTP status
// the API should answer with.
type Error struct {
	Status int
	Msg    string
}

func (e *Error) Error() string { return e.Msg }

func badRequest(format string, a ...any) error {
	return &Error{http.StatusBadRequest, fmt.Sprintf(format, a...)}
}

func conflict(format string, a ...any) error {
	return &Error{http.StatusConflict, fmt.Sprintf(format, a...)}
}

// upstream marks a failure of something we depend on (metadata provider,
// indexer, download client) rather than of the request itself.
func upstream(format string, a ...any) error {
	return &Error{http.StatusBadGateway, fmt.Sprintf(format, a...)}
}

var ErrNotFound = &Error{http.StatusNotFound, "Not found"}

func notFound(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

type Service struct {
	Store   *store.Store
	Meta    *metadata.Client
	Indexer *indexer.Client
	Log     *slog.Logger

	Version   string
	DataDir   string
	StartedAt time.Time

	releaseMu sync.Mutex
	releases  map[string]cachedRelease

	clientMu sync.Mutex
	clients  map[int64]cachedClient

	// importMu serialises imports: the queue sync and a manual retry must
	// not copy the same download at once.
	importMu sync.Mutex

	tasks *taskRunner
}

type cachedRelease struct {
	release model.Release
	bookID  int64
	expires time.Time
}

type cachedClient struct {
	cfg    model.DownloadClient
	client download.Client
}

func New(st *store.Store, log *slog.Logger, version, dataDir string) *Service {
	s := &Service{
		Store:     st,
		Meta:      metadata.New(),
		Indexer:   indexer.New(),
		Log:       log,
		Version:   version,
		DataDir:   dataDir,
		StartedAt: time.Now().UTC(),
		releases:  map[string]cachedRelease{},
		clients:   map[int64]cachedClient{},
	}
	s.tasks = newTaskRunner(s)
	if set, err := st.Settings(); err == nil {
		s.configureMetadata(set)
	}
	return s
}

// client returns a (cached) connection for a download client. The cache keeps
// qBittorrent's login session between polls; it is rebuilt when the stored
// configuration changes.
func (s *Service) client(cfg model.DownloadClient) (download.Client, error) {
	s.clientMu.Lock()
	defer s.clientMu.Unlock()
	if c, ok := s.clients[cfg.ID]; ok && c.cfg == cfg {
		return c.client, nil
	}
	c, err := download.New(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.ID != 0 {
		s.clients[cfg.ID] = cachedClient{cfg, c}
	}
	return c, nil
}
