package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/ncecere/chaptarr/internal/download"
	"github.com/ncecere/chaptarr/internal/indexer"
	"github.com/ncecere/chaptarr/internal/model"
)

const releaseTTL = 30 * time.Minute

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// BookReleases searches every enabled indexer for a book and returns the
// scored results. Each is remembered for a while so it can be grabbed by id.
func (s *Service) BookReleases(ctx context.Context, bookID int64) ([]model.Release, error) {
	b, err := s.Store.Book(bookID)
	if err != nil {
		return nil, notFound(err)
	}
	return s.findReleases(ctx, b)
}

func (s *Service) findReleases(ctx context.Context, b *model.Book) ([]model.Release, error) {
	all, err := s.Store.Indexers()
	if err != nil {
		return nil, err
	}
	var enabled []model.Indexer
	for _, ix := range all {
		if ix.Enabled {
			enabled = append(enabled, ix)
		}
	}
	if len(enabled) == 0 {
		return nil, badRequest("No indexers are enabled. Add one under Settings → Indexers.")
	}
	set, err := s.Store.Settings()
	if err != nil {
		return nil, err
	}
	failed, err := s.Store.FailedGUIDs(b.ID)
	if err != nil {
		return nil, err
	}

	siblings, err := s.Store.Books(model.BookFilter{AuthorID: b.AuthorID})
	if err != nil {
		return nil, err
	}
	var otherTitles []string
	for _, o := range siblings {
		if o.ID != b.ID {
			otherTitles = append(otherTitles, o.Title)
		}
	}

	queries := indexer.Queries(b)
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		releases []model.Release
		seen     = map[string]bool{}
		lastErr  error
		okCount  int
	)
	for _, ix := range enabled {
		wg.Add(1)
		go func(ix model.Indexer) {
			defer wg.Done()
			ok := false
			for _, q := range queries {
				found, err := s.Indexer.Search(ctx, ix, q)
				if err != nil {
					s.Log.Warn("indexer search failed", "indexer", ix.Name, "error", err)
					mu.Lock()
					lastErr = fmt.Errorf("%s: %w", ix.Name, err)
					mu.Unlock()
					break
				}
				ok = true
				mu.Lock()
				for _, r := range found {
					key := fmt.Sprintf("%d\x00%s", ix.ID, r.GUID)
					if !seen[key] {
						seen[key] = true
						releases = append(releases, r)
					}
				}
				mu.Unlock()
			}
			if ok {
				mu.Lock()
				okCount++
				mu.Unlock()
			}
		}(ix)
	}
	wg.Wait()
	if okCount == 0 && lastErr != nil {
		return nil, upstream("Every indexer search failed. Last error: %v", lastErr)
	}

	releases = indexer.Score(releases, indexer.Criteria{
		Book:             b,
		PreferredFormats: set.PreferredFormats,
		MinSeeders:       set.MinSeeders,
		FailedGUIDs:      failed,
		OtherTitles:      otherTitles,
	})

	s.releaseMu.Lock()
	now := time.Now()
	for id, c := range s.releases {
		if now.After(c.expires) {
			delete(s.releases, id)
		}
	}
	for i := range releases {
		releases[i].ID = randomID()
		s.releases[releases[i].ID] = cachedRelease{releases[i], b.ID, now.Add(releaseTTL)}
	}
	s.releaseMu.Unlock()
	if releases == nil {
		releases = []model.Release{}
	}
	return releases, nil
}

// GrabRelease sends a release from a recent search to a download client.
func (s *Service) GrabRelease(ctx context.Context, bookID int64, releaseID string) (*model.QueueItem, error) {
	s.releaseMu.Lock()
	c, ok := s.releases[releaseID]
	s.releaseMu.Unlock()
	if !ok || c.bookID != bookID || time.Now().After(c.expires) {
		return nil, badRequest("That search result has expired. Search again and retry.")
	}
	b, err := s.Store.Book(bookID)
	if err != nil {
		return nil, notFound(err)
	}
	return s.grab(ctx, b, c.release)
}

func (s *Service) grab(ctx context.Context, b *model.Book, r model.Release) (*model.QueueItem, error) {
	clients, err := s.Store.DownloadClients()
	if err != nil {
		return nil, err
	}
	var cfg *model.DownloadClient
	for i := range clients {
		if clients[i].Enabled && download.Protocol(clients[i].Type) == r.Protocol {
			cfg = &clients[i]
			break
		}
	}
	if cfg == nil {
		return nil, badRequest("No enabled download client handles %s releases. Add one under Settings → Download Clients.", r.Protocol)
	}
	client, err := s.client(*cfg)
	if err != nil {
		return nil, badRequest("%v", err)
	}
	payload, err := download.Fetch(ctx, r.Protocol, r.Title, r.DownloadURL)
	if err != nil {
		return nil, upstream("%v", capitalize(err.Error()))
	}
	itemID, err := client.Add(ctx, payload)
	if err != nil {
		return nil, upstream("%s: %v", cfg.Name, err)
	}
	q := &model.QueueItem{
		BookID: b.ID, ReleaseTitle: r.Title, GUID: r.GUID, Indexer: r.Indexer, Protocol: r.Protocol,
		ClientID: cfg.ID, DownloadClient: cfg.Name, ClientItemID: itemID,
		Status: model.QueueQueued, SizeBytes: r.SizeBytes,
	}
	if err := s.Store.AddQueueItem(q); err != nil {
		return nil, err
	}
	_ = s.Store.AddHistory(b, model.EventGrabbed, r.Title, r.GUID, fmt.Sprintf("%s → %s", r.Indexer, cfg.Name))
	s.Log.Info("release grabbed", "book", b.Title, "release", r.Title, "indexer", r.Indexer, "client", cfg.Name)
	// Pick up the client's first status report without waiting for the next
	// scheduled sync, so the queue does not sit at "queued 0%".
	time.AfterFunc(3*time.Second, func() { _ = s.RunTask("syncQueue") })
	return s.Store.QueueItem(q.ID)
}

func capitalize(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(s[0]-32) + s[1:]
}

// SearchBook searches for a book and grabs the best acceptable release.
func (s *Service) SearchBook(ctx context.Context, bookID int64) (grabbed bool, message string, err error) {
	b, err := s.Store.Book(bookID)
	if err != nil {
		return false, "", notFound(err)
	}
	switch b.Status {
	case model.StatusDownloaded:
		return false, "The book is already downloaded", nil
	case model.StatusDownloading:
		return false, "The book is already in the download queue", nil
	}
	_ = s.Store.TouchBookSearch(b.ID)
	releases, err := s.findReleases(ctx, b)
	if err != nil {
		return false, "", err
	}
	var grabErr error
	attempts := 0
	for _, r := range releases {
		if len(r.Rejections) > 0 {
			break // sorted: acceptable releases come first
		}
		if attempts == 3 {
			break
		}
		attempts++
		if _, err := s.grab(ctx, b, r); err != nil {
			grabErr = err
			s.Log.Warn("grab failed, trying next release", "book", b.Title, "release", r.Title, "error", err)
			continue
		}
		return true, "Grabbed " + r.Title, nil
	}
	if grabErr != nil {
		return false, "", grabErr
	}
	if len(releases) == 0 {
		return false, "No releases found", nil
	}
	return false, fmt.Sprintf("Found %d releases, but none passed the automatic checks. Use interactive search to pick one.", len(releases)), nil
}

// searchMissing searches for missing monitored books, oldest search first.
// cooldown skips books searched recently; limit caps the batch (0 = all).
func (s *Service) searchMissing(ctx context.Context, authorID int64, cooldown time.Duration, limit int) error {
	ids, err := s.Store.MissingBookIDs(authorID, time.Now().Add(-cooldown), limit)
	if err != nil {
		return err
	}
	grabbed := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ok, _, err := s.SearchBook(ctx, id)
		if err != nil {
			// A configuration problem (no indexers, no client) will fail
			// the same way for every book.
			if e, isErr := err.(*Error); isErr && e.Status == 400 {
				return err
			}
			s.Log.Warn("search failed", "bookId", id, "error", err)
			continue
		}
		if ok {
			grabbed++
		}
		// Be polite to indexers, many of which rate-limit API hits.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if len(ids) > 0 {
		s.Log.Info("missing book search finished", "searched", len(ids), "grabbed", grabbed)
	}
	return nil
}

// SearchMissingInBackground queues a search for an author's missing books
// and returns how many will be searched.
func (s *Service) SearchMissingInBackground(authorID int64) int {
	ids, err := s.Store.MissingBookIDs(authorID, time.Now(), 0)
	if err != nil {
		return 0
	}
	go func() {
		if err := s.searchMissing(context.Background(), authorID, 0, 0); err != nil {
			s.Log.Warn("author search failed", "authorId", authorID, "error", err)
		}
	}()
	return len(ids)
}
