package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ncecere/chaptarr/internal/library"
	"github.com/ncecere/chaptarr/internal/model"
)

// fakeSAB is a SABnzbd holding one finished job, saved in dir.
type fakeSAB struct {
	mu      sync.Mutex
	dir     string
	removed []url.Values // history deletions
}

func (s *fakeSAB) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		s.mu.Lock()
		defer s.mu.Unlock()
		gone := len(s.removed) > 0
		var out any = map[string]any{"status": true}
		switch {
		case q.Get("mode") == "addfile":
			out = map[string]any{"status": true, "nzo_ids": []string{"SABnzbd_nzo_1"}}
		case q.Get("mode") == "queue" && q.Get("name") == "":
			out = map[string]any{"queue": map[string]any{"slots": []any{}}}
		case q.Get("mode") == "history" && q.Get("name") == "delete":
			s.removed = append(s.removed, q)
		case q.Get("mode") == "history":
			slots := []any{}
			if !gone {
				slots = append(slots, map[string]any{"nzo_id": "SABnzbd_nzo_1", "status": "Completed", "storage": s.dir, "bytes": 1234})
			}
			out = map[string]any{"history": map[string]any{"slots": slots}}
		}
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fakeUsenetIndexer(t *testing.T) *httptest.Server {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/dl/"):
			fmt.Fprint(w, `<?xml version="1.0"?><nzb xmlns="http://www.newzbin.com/DTD/2003/nzb"></nzb>`)
		case r.URL.Query().Get("t") == "caps":
			fmt.Fprint(w, `<caps><server title="fake"/></caps>`)
		default:
			fmt.Fprintf(w, `<?xml version="1.0"?><rss><channel><item><title>Ann.Leckie-Ancillary.Justice.[AUDIOBOOK]</title>
				<guid>nzb-1</guid><pubDate>Mon, 02 Jan 2023 15:04:05 +0000</pubDate>
				<enclosure url="%s/dl/nzb-1" length="%d" type="application/x-nzb"/></item></channel></rss>`, srv.URL, 400<<20)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A usenet download that cannot be imported on its own (here: its audio is
// a format Chaptarr does not know) is settled with a manual import from a
// folder the user picks, and is then cleared out of SABnzbd's history.
func TestManualImportSettlesStuckDownload(t *testing.T) {
	h := newHarness(t)
	h.signUp()
	root := t.TempDir()
	write(t, filepath.Join(root, ".keep"))
	h.must(200, "POST", "/api/v1/rootfolders", map[string]string{"path": root}, nil)

	downloads := t.TempDir()
	job := filepath.Join(downloads, "Ann.Leckie-Ancillary.Justice.[AUDIOBOOK]")
	write(t, filepath.Join(job, "justice.xyz")) // not audio: the automatic import fails
	sab := &fakeSAB{dir: job}
	h.must(200, "POST", "/api/v1/indexers", map[string]any{"name": "NZB", "type": "newznab", "url": fakeUsenetIndexer(t).URL, "apiKey": "k", "categories": "3030", "enabled": true, "priority": 25}, nil)
	h.must(200, "POST", "/api/v1/downloadclients", map[string]any{"name": "SAB", "type": "sabnzbd", "url": sab.server(t).URL, "apiKey": "k", "enabled": true}, nil)

	var author model.AuthorDetail
	h.must(200, "POST", "/api/v1/authors", map[string]any{"asin": authorASIN, "name": "Ann Leckie", "monitor": "all"}, &author)
	var book model.Book
	for _, b := range author.Books {
		if b.ASIN == "B00BOOK001" {
			book = b
		}
	}
	var releases []model.Release
	h.must(200, "GET", fmt.Sprintf("/api/v1/books/%d/releases", book.ID), nil, &releases)
	if len(releases) != 1 || releases[0].Protocol != model.ProtocolUsenet {
		t.Fatalf("releases: %+v", releases)
	}
	h.must(200, "POST", fmt.Sprintf("/api/v1/books/%d/grab", book.ID), map[string]string{"releaseId": releases[0].ID}, nil)
	if err := h.svc.SyncQueue(context.Background()); err != nil {
		t.Fatal(err)
	}
	var queue []model.QueueItem
	h.must(200, "GET", "/api/v1/queue", nil, &queue)
	if len(queue) != 1 || queue[0].Status != model.QueueCompleted || !strings.Contains(queue[0].Error, "1 × .xyz") {
		t.Fatalf("queue after a failed import: %+v", queue)
	}
	if len(sab.removed) != 0 {
		t.Fatal("a download that was not imported must stay in SABnzbd")
	}

	// With no path, the preview looks where SABnzbd saved the job.
	var contents library.Contents
	h.must(200, "GET", fmt.Sprintf("/api/v1/manualimport?queueId=%d", queue[0].ID), nil, &contents)
	if contents.Path != job || len(contents.Files) != 0 || !strings.Contains(contents.Other, ".xyz") {
		t.Fatalf("preview of the download: %+v", contents)
	}
	h.must(400, "GET", "/api/v1/manualimport?path="+url.QueryEscape(filepath.Join(downloads, "nope")), nil, nil)

	// The user has the audio elsewhere, in nested folders.
	fixed := filepath.Join(downloads, "fixed")
	write(t, filepath.Join(fixed, "CD1", "01.mp3"))
	write(t, filepath.Join(fixed, "CD2", "02.mp3"))
	h.must(200, "GET", "/api/v1/manualimport?path="+url.QueryEscape(fixed), nil, &contents)
	if len(contents.Files) != 2 || contents.Files[0].Name != filepath.Join("CD1", "01.mp3") || contents.TotalBytes != 10 {
		t.Fatalf("preview: %+v", contents)
	}

	var detail model.BookDetail
	h.must(200, "POST", fmt.Sprintf("/api/v1/books/%d/manualimport", book.ID),
		map[string]any{"path": fixed, "mode": "copy", "queueId": queue[0].ID}, &detail)
	if detail.Status != model.StatusDownloaded || len(detail.Files) != 2 {
		t.Fatalf("book after manual import: %s, %d files", detail.Status, len(detail.Files))
	}
	want := filepath.Join(root, "Ann Leckie", "Imperial Radch", "Ancillary Justice", "CD2", "02.mp3")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("imported file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fixed, "CD1", "01.mp3")); err != nil {
		t.Fatal("copy mode must leave the source in place")
	}
	h.must(200, "GET", "/api/v1/queue", nil, &queue)
	if len(queue) != 0 {
		t.Fatalf("the queue item should be settled: %+v", queue)
	}
	if len(sab.removed) != 1 || sab.removed[0].Get("value") != "SABnzbd_nzo_1" {
		t.Fatalf("the imported download should be removed from SABnzbd once: %+v", sab.removed)
	}

	// Importing the book's own library folder is refused rather than nested.
	h.must(422, "POST", fmt.Sprintf("/api/v1/books/%d/manualimport", book.ID),
		map[string]any{"path": filepath.Dir(filepath.Dir(want))}, nil)
	h.must(400, "POST", fmt.Sprintf("/api/v1/books/%d/manualimport", book.ID), map[string]any{"path": fixed, "mode": "teleport"}, nil)
}

// With the setting off, an imported download stays in the client.
func TestImportedDownloadKeptWhenRemovalIsOff(t *testing.T) {
	h := newHarness(t)
	h.signUp()
	root := t.TempDir()
	write(t, filepath.Join(root, ".keep"))
	h.must(200, "POST", "/api/v1/rootfolders", map[string]string{"path": root}, nil)
	var set model.Settings
	h.must(200, "GET", "/api/v1/settings", nil, &set)
	if !set.RemoveCompletedDownloads {
		t.Fatal("removing completed downloads should be on by default")
	}
	set.RemoveCompletedDownloads = false
	h.must(200, "PUT", "/api/v1/settings", set, nil)

	job := filepath.Join(t.TempDir(), "job")
	write(t, filepath.Join(job, "book.m4b"))
	sab := &fakeSAB{dir: job}
	h.must(200, "POST", "/api/v1/indexers", map[string]any{"name": "NZB", "type": "newznab", "url": fakeUsenetIndexer(t).URL, "apiKey": "k", "categories": "3030", "enabled": true, "priority": 25}, nil)
	h.must(200, "POST", "/api/v1/downloadclients", map[string]any{"name": "SAB", "type": "sabnzbd", "url": sab.server(t).URL, "apiKey": "k", "enabled": true}, nil)
	var author model.AuthorDetail
	h.must(200, "POST", "/api/v1/authors", map[string]any{"asin": authorASIN, "name": "Ann Leckie", "monitor": "all"}, &author)
	var book model.Book
	for _, b := range author.Books {
		if b.ASIN == "B00BOOK001" {
			book = b
		}
	}
	var releases []model.Release
	h.must(200, "GET", fmt.Sprintf("/api/v1/books/%d/releases", book.ID), nil, &releases)
	h.must(200, "POST", fmt.Sprintf("/api/v1/books/%d/grab", book.ID), map[string]string{"releaseId": releases[0].ID}, nil)
	if err := h.svc.SyncQueue(context.Background()); err != nil {
		t.Fatal(err)
	}
	var detail model.BookDetail
	h.must(200, "GET", fmt.Sprintf("/api/v1/books/%d", book.ID), nil, &detail)
	if detail.Status != model.StatusDownloaded {
		t.Fatalf("status %q, want downloaded", detail.Status)
	}
	if len(sab.removed) != 0 {
		t.Fatalf("the download was removed although the setting is off: %+v", sab.removed)
	}
}
