package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ncecere/chaptarr/internal/download"
	"github.com/ncecere/chaptarr/internal/library"
	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/store"
)

func (s *Service) clientFor(q *model.QueueItem) (download.Client, error) {
	cfg, err := s.Store.DownloadClient(q.ClientID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, errors.New("the download client for this item was deleted")
		}
		return nil, err
	}
	return s.client(*cfg)
}

// SyncQueue refreshes every queue item from its download client and imports
// the ones that have finished.
func (s *Service) SyncQueue(ctx context.Context) error {
	items, err := s.Store.Queue()
	if err != nil {
		return err
	}
	for i := range items {
		q := &items[i]
		if q.Status == model.QueueFailed {
			continue // stays until the user removes it
		}
		if err := s.syncItem(ctx, q); err != nil {
			s.Log.Warn("queue sync failed", "release", q.ReleaseTitle, "error", err)
		}
	}
	return nil
}

func (s *Service) syncItem(ctx context.Context, q *model.QueueItem) error {
	client, err := s.clientFor(q)
	if err != nil {
		return s.failItem(q, err.Error())
	}
	it, err := client.Status(ctx, q.ClientItemID)
	if errors.Is(err, download.ErrGone) {
		return s.failItem(q, "The download was removed from "+q.DownloadClient)
	}
	if err != nil {
		// The client is unreachable: keep the last known state and say why.
		return s.Store.UpdateQueueItem(q.ID, q.Status, q.Progress, q.SizeBytes, q.ETASeconds,
			fmt.Sprintf("%s is not responding: %v", q.DownloadClient, err))
	}
	size := it.SizeBytes
	if size == 0 {
		size = q.SizeBytes
	}
	switch it.State {
	case download.StateFailed:
		q.Progress, q.SizeBytes = it.Progress, size
		return s.failItem(q, it.Error)
	case download.StateCompleted:
		if err := s.Store.UpdateQueueItem(q.ID, model.QueueCompleted, 1, size, nil, q.Error); err != nil {
			return err
		}
		q.Status, q.Progress, q.SizeBytes = model.QueueCompleted, 1, size
		return s.importItem(q, it.Path)
	}
	status := map[download.State]string{
		download.StateQueued:      model.QueueQueued,
		download.StateDownloading: model.QueueDownloading,
		download.StatePaused:      model.QueuePaused,
	}[it.State]
	return s.Store.UpdateQueueItem(q.ID, status, it.Progress, size, it.ETASeconds, "")
}

func (s *Service) failItem(q *model.QueueItem, msg string) error {
	if b, err := s.Store.Book(q.BookID); err == nil {
		_ = s.Store.AddHistory(b, model.EventDownloadFailed, q.ReleaseTitle, q.GUID, msg)
	}
	s.Log.Warn("download failed", "release", q.ReleaseTitle, "reason", msg)
	return s.Store.UpdateQueueItem(q.ID, model.QueueFailed, q.Progress, q.SizeBytes, nil, msg)
}

// importItem moves a finished download into the library. On failure the item
// stays "completed" with the error, and the next sync tries again; the
// failure is written to history only when the reason changes.
func (s *Service) importItem(q *model.QueueItem, src string) error {
	s.importMu.Lock()
	defer s.importMu.Unlock()
	// A manual retry may have imported it while we waited for the lock.
	if _, err := s.Store.QueueItem(q.ID); errors.Is(err, store.ErrNotFound) {
		return nil
	}
	b, err := s.Store.Book(q.BookID)
	if err != nil {
		return err
	}
	_ = s.Store.UpdateQueueItem(q.ID, model.QueueImporting, 1, q.SizeBytes, nil, "")

	dir, files, err := s.doImport(b, q, src)
	if err != nil {
		msg := err.Error()
		if msg != q.Error {
			_ = s.Store.AddHistory(b, model.EventImportFailed, q.ReleaseTitle, "", msg)
			s.Log.Warn("import failed", "book", b.Title, "error", msg)
		}
		return s.Store.UpdateQueueItem(q.ID, model.QueueCompleted, 1, q.SizeBytes, nil, msg)
	}
	if err := s.Store.SetBookFiles(b.ID, dir, files); err != nil {
		return err
	}
	_ = s.Store.AddHistory(b, model.EventImported, q.ReleaseTitle, q.GUID, dir)
	s.Log.Info("book imported", "book", b.Title, "files", len(files), "path", dir)
	return s.Store.DeleteQueueItem(q.ID)
}

func (s *Service) doImport(b *model.Book, q *model.QueueItem, src string) (string, []model.BookFile, error) {
	if src == "" {
		return "", nil, fmt.Errorf("%s did not report where the download was saved", q.DownloadClient)
	}
	set, err := s.Store.Settings()
	if err != nil {
		return "", nil, err
	}
	roots, err := s.Store.RootFolders()
	if err != nil {
		return "", nil, err
	}
	if len(roots) == 0 {
		return "", nil, errors.New("no root folder is configured; add one under Settings → Root Folders")
	}
	root := roots[0].Path
	if set.DefaultRootFolderID != nil {
		for _, r := range roots {
			if r.ID == *set.DefaultRootFolderID {
				root = r.Path
			}
		}
	}
	dir := filepath.Join(root, library.BookFolder(set.FolderFormat, b))
	// Torrents keep seeding from the download folder, so their files are
	// never moved out from under the client.
	keepSource := q.Protocol == model.ProtocolTorrent
	files, err := library.Import(src, dir, set.ImportMode, keepSource)
	return dir, files, err
}

// RetryImport re-checks one queue item now, importing it if it has finished.
func (s *Service) RetryImport(ctx context.Context, id int64) error {
	q, err := s.Store.QueueItem(id)
	if err != nil {
		return notFound(err)
	}
	client, err := s.clientFor(q)
	if err != nil {
		return badRequest("%v", capitalize(err.Error()))
	}
	it, err := client.Status(ctx, q.ClientItemID)
	if err != nil {
		return upstream("%s: %v", q.DownloadClient, err)
	}
	if it.State != download.StateCompleted {
		return conflict("The download has not finished in %s yet", q.DownloadClient)
	}
	if err := s.importItem(q, it.Path); err != nil {
		return err
	}
	// importItem records a failed attempt on the item rather than returning it.
	if after, err := s.Store.QueueItem(id); err == nil && after.Error != "" {
		return &Error{Status: 422, Msg: "Import failed: " + after.Error}
	}
	return nil
}

func (s *Service) RemoveQueueItem(ctx context.Context, id int64, removeFromClient bool) error {
	q, err := s.Store.QueueItem(id)
	if err != nil {
		return notFound(err)
	}
	if removeFromClient {
		client, err := s.clientFor(q)
		if err != nil {
			return badRequest("%v", capitalize(err.Error()))
		}
		if err := client.Remove(ctx, q.ClientItemID, true); err != nil && !errors.Is(err, download.ErrGone) {
			return upstream("Could not remove the download from %s: %v", q.DownloadClient, err)
		}
	}
	return s.Store.DeleteQueueItem(id)
}

// ScanLibrary reconciles the database with the disk: files that vanished are
// dropped, and audio found in the root folders is matched to books without
// files.
func (s *Service) ScanLibrary(ctx context.Context) error {
	tracked, err := s.Store.AllFiles()
	if err != nil {
		return err
	}
	known := map[string]bool{}
	byBook := map[int64][]model.BookFile{}
	lost := map[int64]bool{}
	roots, err := s.rootPaths()
	if err != nil {
		return err
	}
	// A file only counts as deleted when its root folder is demonstrably
	// there. Otherwise an unmounted share would empty the library and send
	// every book back to the indexers.
	mounted := func(path string) bool {
		for _, r := range roots {
			if library.Under(r, path) {
				return library.Mounted(r)
			}
		}
		return false
	}
	for _, f := range tracked {
		if library.Exists(f.Path) || !mounted(f.Path) {
			known[f.Path] = true
			byBook[f.BookID] = append(byBook[f.BookID], f.BookFile)
		} else {
			lost[f.BookID] = true
		}
	}
	for bookID := range lost {
		b, err := s.Store.Book(bookID)
		if err != nil {
			continue
		}
		left := byBook[bookID]
		dir := b.Path
		if len(left) == 0 {
			dir = ""
		}
		if err := s.Store.SetBookFiles(bookID, dir, left); err != nil {
			return err
		}
		_ = s.Store.AddHistory(b, model.EventFileDeleted, b.Path, "", "Files are no longer on disk")
		s.Log.Info("book files missing from disk", "book", b.Title)
	}

	books, err := s.Store.Books(model.BookFilter{})
	if err != nil {
		return err
	}
	found := library.Scan(roots, books, known)
	for _, f := range found {
		if err := s.Store.SetBookFiles(f.BookID, f.Dir, f.Files); err != nil {
			return err
		}
		if b, err := s.Store.Book(f.BookID); err == nil {
			_ = s.Store.AddHistory(b, model.EventImported, f.Dir, "", "Found on disk by library scan")
		}
	}
	s.Log.Info("library scan finished", "matched", len(found), "lostBooks", len(lost))
	return nil
}
