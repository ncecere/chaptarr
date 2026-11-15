package api

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/ncecere/chaptarr/internal/logbuf"
	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/service"
	"github.com/ncecere/chaptarr/internal/store"
)

// ---------------------------------------------------------------- fakes

const authorASIN = "B00AUTHOR1"

func product(asin, title, lang, date string, runtime int, series string) map[string]any {
	p := map[string]any{
		"asin": asin, "title": title, "language": lang, "release_date": date,
		"runtime_length_min": runtime, "content_delivery_type": "MultiPartBook", "format_type": "unabridged",
		"authors":        []map[string]string{{"asin": authorASIN, "name": "Ann Leckie"}},
		"narrators":      []map[string]string{{"name": "Adjoa Andoh"}},
		"product_images": map[string]string{"500": "https://example.test/" + asin + ".jpg"},
		"publisher_name": "Hachette",
	}
	if series != "" {
		p["series"] = []map[string]string{{"asin": "B00SERIES1", "title": "Imperial Radch", "sequence": series}}
	}
	return p
}

var catalogue = []map[string]any{
	product("B00BOOK001", "Ancillary Justice", "english", "2013-10-01", 780, "1"),
	product("B00BOOK002", "Ancillary Sword", "english", "2014-10-07", 720, "2"),
	product("B00BOOK003", "Ancillary Justice", "german", "2015-01-01", 800, "1"),
	product("B00BOOK004", "Translation State", "english", "2999-01-01", 600, ""),
}

func fakeMetadata(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /1.0/catalog/products", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"products": catalogue, "total_results": len(catalogue)})
	})
	mux.HandleFunc("GET /1.0/catalog/products/{asin}", func(w http.ResponseWriter, r *http.Request) {
		for _, p := range catalogue {
			if p["asin"] == r.PathValue("asin") {
				json.NewEncoder(w).Encode(map[string]any{"product": p})
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /authors", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]string{{"asin": authorASIN, "name": "Ann Leckie"}, {"asin": authorASIN, "name": "Ann Leckie"}})
	})
	mux.HandleFunc("GET /authors/{asin}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"asin": authorASIN, "name": "Ann Leckie", "description": "Bio", "image": "https://example.test/a.jpg"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// A minimal single-file torrent; the info-hash is computed from infoDict.
const infoDict = "d6:lengthi5e4:name4:test12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaae"

var torrentFile = []byte("d8:announce14:http://tracker4:info" + infoDict + "e")

func fakeIndexer(t *testing.T) *httptest.Server {
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("apikey") != "indexer-key" {
			fmt.Fprint(w, `<error code="100" description="Incorrect user credentials"/>`)
			return
		}
		if r.URL.Query().Get("t") == "caps" {
			fmt.Fprint(w, `<caps><server title="fake"/></caps>`)
			return
		}
		item := func(title, guid string, size int64, seeders int) string {
			return fmt.Sprintf(`<item><title>%s</title><guid>%s</guid><pubDate>Mon, 02 Jan 2023 15:04:05 +0000</pubDate>
				<enclosure url="%s/dl/%s" length="%d" type="application/x-bittorrent"/>
				<torznab:attr name="seeders" value="%d"/><torznab:attr name="peers" value="%d"/></item>`,
				title, guid, srv.URL, guid, size, seeders, seeders+2)
		}
		fmt.Fprint(w, `<?xml version="1.0"?><rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel>`+
			item("Ann Leckie - Ancillary Justice (Imperial Radch 1) [MP3 64kbps]", "g-mp3", 350<<20, 12)+
			item("Ann Leckie - Ancillary Justice [M4B]", "g-m4b", 400<<20, 8)+
			item("Ann Leckie - Ancillary Justice [EPUB]", "g-epub", 1<<20, 90)+
			item("Ancillary Justice by Ann Leckie M4B dead", "g-dead", 400<<20, 0)+
			item("Some Other Book - Jane Doe [M4B]", "g-other", 400<<20, 50)+
			`</channel></rss>`)
	})
	mux.HandleFunc("GET /dl/{guid}", func(w http.ResponseWriter, r *http.Request) { w.Write(torrentFile) })
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// fakeQBit is a qBittorrent that "finishes" a torrent when told to.
type fakeQBit struct {
	mu       sync.Mutex
	torrents map[string]map[string]any
	removed  []string
}

func (q *fakeQBit) server(t *testing.T) *httptest.Server {
	q.torrents = map[string]map[string]any{}
	authed := func(r *http.Request) bool {
		c, err := r.Cookie("SID")
		return err == nil && c.Value == "ok"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("username") != "admin" || r.FormValue("password") != "adminadmin" {
			fmt.Fprint(w, "Fails.")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "ok", Path: "/"})
		fmt.Fprint(w, "Ok.")
	})
	guard := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !authed(r) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("GET /api/v2/app/version", guard(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "v5.0.0") }))
	mux.HandleFunc("POST /api/v2/torrents/createCategory", guard(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("POST /api/v2/torrents/add", guard(func(w http.ResponseWriter, r *http.Request) {
		f, _, err := r.FormFile("torrents")
		if err != nil {
			fmt.Fprint(w, "Fails.")
			return
		}
		data, _ := io.ReadAll(f)
		if !bytes.Equal(data, torrentFile) || r.FormValue("category") != "audiobooks" {
			fmt.Fprint(w, "Fails.")
			return
		}
		sum := sha1.Sum([]byte(infoDict))
		q.mu.Lock()
		q.torrents[hex.EncodeToString(sum[:])] = map[string]any{"state": "downloading", "progress": 0.4, "size": 400 << 20, "eta": 120}
		q.mu.Unlock()
		fmt.Fprint(w, "Ok.")
	}))
	mux.HandleFunc("GET /api/v2/torrents/info", guard(func(w http.ResponseWriter, r *http.Request) {
		q.mu.Lock()
		defer q.mu.Unlock()
		out := []map[string]any{}
		if tr, ok := q.torrents[r.URL.Query().Get("hashes")]; ok {
			out = append(out, tr)
		}
		json.NewEncoder(w).Encode(out)
	}))
	mux.HandleFunc("POST /api/v2/torrents/delete", guard(func(w http.ResponseWriter, r *http.Request) {
		q.mu.Lock()
		defer q.mu.Unlock()
		delete(q.torrents, r.FormValue("hashes"))
		q.removed = append(q.removed, r.FormValue("hashes"))
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func (q *fakeQBit) finish(contentPath string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, tr := range q.torrents {
		tr["state"], tr["progress"], tr["content_path"] = "stalledUP", 1.0, contentPath
	}
}

// ---------------------------------------------------------------- harness

type harness struct {
	t    *testing.T
	svc  *service.Service
	srv  *httptest.Server
	http *http.Client
}

func newHarness(t *testing.T) *harness {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	logs := logbuf.NewBuffer(100)
	svc := service.New(st, logbuf.NewLogger(io.Discard, logs, slog.LevelDebug), "test", t.TempDir())
	meta := fakeMetadata(t)
	svc.Meta.AudibleBase, svc.Meta.AudnexusBase = meta.URL, meta.URL
	ui := fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"assets/app.js": {Data: []byte("js")},
	}
	srv := httptest.NewServer(New(svc, logs, fs.FS(ui)).Handler())
	t.Cleanup(srv.Close)
	jar, _ := cookiejar.New(nil)
	return &harness{t: t, svc: svc, srv: srv, http: &http.Client{Jar: jar}}
}

// call sends a request and decodes the response into out (if non-nil),
// returning the status code.
func (h *harness) call(method, path string, body, out any) int {
	h.t.Helper()
	var rd io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		rd = bytes.NewReader(data)
	}
	req, _ := http.NewRequest(method, h.srv.URL+path, rd)
	res, err := h.http.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			h.t.Fatalf("%s %s: decoding %q: %v", method, path, data, err)
		}
	}
	return res.StatusCode
}

// must is call that fails the test unless the status matches.
func (h *harness) must(want int, method, path string, body, out any) {
	h.t.Helper()
	var raw json.RawMessage
	got := h.call(method, path, body, &raw)
	if got != want {
		h.t.Fatalf("%s %s: status %d, want %d; body %s", method, path, got, want, raw)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			h.t.Fatalf("%s %s: %v", method, path, err)
		}
	}
}

func (h *harness) signUp() {
	h.t.Helper()
	h.must(200, "POST", "/api/v1/auth/setup", map[string]string{"username": "admin", "password": "correct horse"}, nil)
}

// ---------------------------------------------------------------- tests

func TestAuth(t *testing.T) {
	h := newHarness(t)
	var st model.AuthStatus
	h.must(200, "GET", "/api/v1/auth/status", nil, &st)
	if !st.SetupRequired || st.Authenticated {
		t.Fatalf("fresh install: %+v", st)
	}
	if code := h.call("GET", "/api/v1/books", nil, nil); code != 401 {
		t.Fatalf("unauthenticated list: %d", code)
	}
	h.must(400, "POST", "/api/v1/auth/setup", map[string]string{"username": "admin", "password": "short"}, nil)
	h.signUp()
	h.must(409, "POST", "/api/v1/auth/setup", map[string]string{"username": "evil", "password": "another one"}, nil)
	h.must(200, "GET", "/api/v1/auth/status", nil, &st)
	if st.SetupRequired || !st.Authenticated || st.User.Username != "admin" {
		t.Fatalf("after setup: %+v", st)
	}

	var sys model.SystemStatus
	h.must(200, "GET", "/api/v1/system/status", nil, &sys)
	h.must(204, "POST", "/api/v1/auth/logout", map[string]string{}, nil)
	if code := h.call("GET", "/api/v1/books", nil, nil); code != 401 {
		t.Fatalf("after logout: %d", code)
	}
	// The API key works without a session; a wrong one does not.
	req, _ := http.NewRequest("GET", h.srv.URL+"/api/v1/books", nil)
	req.Header.Set("X-Api-Key", sys.APIKey)
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != 200 {
		t.Fatalf("api key: %d", res.StatusCode)
	}
	req.Header.Set("X-Api-Key", "nope")
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != 401 {
		t.Fatalf("bad api key: %d", res.StatusCode)
	}

	h.must(401, "POST", "/api/v1/auth/login", map[string]string{"username": "admin", "password": "wrong"}, nil)
	h.must(200, "POST", "/api/v1/auth/login", map[string]string{"username": "Admin", "password": "correct horse"}, nil)

	// Cross-origin writes are refused even with a valid session.
	req, _ = http.NewRequest("POST", h.srv.URL+"/api/v1/auth/logout", strings.NewReader("{}"))
	req.Header.Set("Origin", "https://evil.example")
	if res, _ := h.http.Do(req); res.StatusCode != 403 {
		t.Fatalf("cross-origin: %d", res.StatusCode)
	}

	// The UI is served for client-side routes, but not for missing assets.
	res, _ := h.http.Get(h.srv.URL + "/authors/12")
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(body), "app") {
		t.Fatalf("spa fallback: %d %q", res.StatusCode, body)
	}
	if res, _ := h.http.Get(h.srv.URL + "/assets/missing.js"); res.StatusCode != 404 {
		t.Fatalf("missing asset: %d", res.StatusCode)
	}
}

func TestFollowAuthorToImportedBook(t *testing.T) {
	h := newHarness(t)
	h.signUp()
	ix := fakeIndexer(t)
	qb := &fakeQBit{}
	qbSrv := qb.server(t)
	root := t.TempDir()
	// Something in the root so it reads as mounted.
	os.WriteFile(filepath.Join(root, ".keep"), nil, 0o644)

	// --- configuration
	var folder model.RootFolder
	h.must(200, "POST", "/api/v1/rootfolders", map[string]string{"path": root}, &folder)
	if !folder.Accessible {
		t.Fatal("root folder should be accessible")
	}
	var set model.Settings
	h.must(200, "GET", "/api/v1/settings", nil, &set)
	if set.DefaultRootFolderID == nil || *set.DefaultRootFolderID != folder.ID {
		t.Fatalf("first root folder should become the default: %+v", set.DefaultRootFolderID)
	}

	indexer := map[string]any{"name": "Fake", "type": "torznab", "url": ix.URL, "apiKey": "wrong", "categories": "3030", "enabled": true, "priority": 25}
	var test model.TestResult
	h.must(200, "POST", "/api/v1/indexers/test", indexer, &test)
	if test.OK {
		t.Fatal("indexer test should fail with a wrong key")
	}
	indexer["apiKey"] = "indexer-key"
	var savedIx model.Indexer
	h.must(200, "POST", "/api/v1/indexers", indexer, &savedIx)
	if savedIx.APIKey != "" || !savedIx.HasAPIKey {
		t.Fatalf("indexer secret must not be echoed: %+v", savedIx)
	}
	// Editing with a blank key keeps the stored one, and so does testing by id.
	indexer["apiKey"], indexer["name"] = "", "Fake Indexer"
	h.must(200, "PUT", fmt.Sprintf("/api/v1/indexers/%d", savedIx.ID), indexer, &savedIx)
	indexer["id"] = savedIx.ID
	h.must(200, "POST", "/api/v1/indexers/test", indexer, &test)
	if !test.OK {
		t.Fatalf("indexer test with stored key: %s", test.Message)
	}

	client := map[string]any{"name": "qBit", "type": "qbittorrent", "url": qbSrv.URL, "username": "admin", "password": "bad", "category": "audiobooks", "enabled": true}
	h.must(200, "POST", "/api/v1/downloadclients/test", client, &test)
	if test.OK {
		t.Fatal("client test should fail with a wrong password")
	}
	client["password"] = "adminadmin"
	h.must(200, "POST", "/api/v1/downloadclients/test", client, &test)
	if !test.OK {
		t.Fatalf("client test: %s", test.Message)
	}
	h.must(200, "POST", "/api/v1/downloadclients", client, nil)

	// --- follow an author
	var authors []model.AuthorResult
	h.must(200, "GET", "/api/v1/search/authors?q=leckie", nil, &authors)
	if len(authors) != 1 || authors[0].ImageURL == "" || authors[0].LibraryAuthorID != nil {
		t.Fatalf("author search: %+v", authors)
	}
	var author model.AuthorDetail
	h.must(200, "POST", "/api/v1/authors", map[string]any{"asin": authorASIN, "name": "Ann Leckie", "monitor": "all", "monitorNewBooks": true}, &author)
	// The German edition is filtered by language, leaving three books.
	if len(author.Books) != 3 || author.BookCount != 3 || author.Description != "Bio" {
		t.Fatalf("author after add: %d books, %+v", len(author.Books), author.Author)
	}
	if author.MissingCount != 2 {
		t.Fatalf("missing count %d, want 2 (the third book is unreleased)", author.MissingCount)
	}
	var justice, future model.Book
	for _, b := range author.Books {
		switch b.ASIN {
		case "B00BOOK001":
			justice = b
		case "B00BOOK004":
			future = b
		}
	}
	if justice.Status != model.StatusMissing || !justice.Monitored || justice.Series == nil || justice.Series.Position != "1" {
		t.Fatalf("book: %+v", justice)
	}
	if future.Status != model.StatusUnreleased {
		t.Fatalf("future book status %q", future.Status)
	}
	h.must(409, "POST", "/api/v1/books", map[string]any{"asin": "B00BOOK001", "monitored": true}, nil)

	var series []model.Series
	h.must(200, "GET", "/api/v1/series", nil, &series)
	if len(series) != 1 || series[0].BookCount != 2 || series[0].Name != "Imperial Radch" {
		t.Fatalf("series: %+v", series)
	}

	// --- interactive search
	var releases []model.Release
	h.must(200, "GET", fmt.Sprintf("/api/v1/books/%d/releases", justice.ID), nil, &releases)
	if len(releases) != 5 {
		t.Fatalf("got %d releases, want 5", len(releases))
	}
	if releases[0].Format != "m4b" || len(releases[0].Rejections) != 0 || releases[0].Seeders == nil || *releases[0].Seeders != 8 {
		t.Fatalf("best release should be the seeded M4B: %+v", releases[0])
	}
	if releases[1].Format != "mp3" || len(releases[1].Rejections) != 0 {
		t.Fatalf("second release should be the MP3: %+v", releases[1])
	}
	for _, r := range releases[2:] {
		if len(r.Rejections) == 0 {
			t.Fatalf("release %q should be rejected", r.Title)
		}
	}

	// --- automatic search grabs the best one
	var result struct {
		Grabbed bool
		Message string
	}
	h.must(200, "POST", fmt.Sprintf("/api/v1/books/%d/search", justice.ID), map[string]any{}, &result)
	if !result.Grabbed || !strings.Contains(result.Message, "[M4B]") {
		t.Fatalf("automatic search: %+v", result)
	}
	var queue []model.QueueItem
	h.must(200, "GET", "/api/v1/queue", nil, &queue)
	if len(queue) != 1 || queue[0].Status != model.QueueQueued || queue[0].DownloadClient != "qBit" {
		t.Fatalf("queue after grab: %+v", queue)
	}
	var book model.BookDetail
	h.must(200, "GET", fmt.Sprintf("/api/v1/books/%d", justice.ID), nil, &book)
	if book.Status != model.StatusDownloading {
		t.Fatalf("status %q, want downloading", book.Status)
	}
	// A second search must not grab a duplicate.
	h.must(200, "POST", fmt.Sprintf("/api/v1/books/%d/search", justice.ID), map[string]any{}, &result)
	if result.Grabbed {
		t.Fatal("grabbed a second copy of a book that is already downloading")
	}

	// --- progress, then completion and import
	ctx := context.Background()
	if err := h.svc.SyncQueue(ctx); err != nil {
		t.Fatal(err)
	}
	h.must(200, "GET", "/api/v1/queue", nil, &queue)
	if queue[0].Status != model.QueueDownloading || queue[0].Progress != 0.4 || queue[0].ETASeconds == nil {
		t.Fatalf("queue while downloading: %+v", queue[0])
	}

	// The client reports a path that does not exist: import fails, item stays.
	qb.finish(filepath.Join(t.TempDir(), "nowhere"))
	h.svc.SyncQueue(ctx)
	h.must(200, "GET", "/api/v1/queue", nil, &queue)
	if len(queue) != 1 || queue[0].Status != model.QueueCompleted || queue[0].Error == "" {
		t.Fatalf("queue after failed import: %+v", queue)
	}

	downloads := t.TempDir()
	src := filepath.Join(downloads, "Ann Leckie - Ancillary Justice [M4B]")
	os.MkdirAll(src, 0o755)
	os.WriteFile(filepath.Join(src, "Ancillary Justice.m4b"), []byte("audio-bytes"), 0o644)
	os.WriteFile(filepath.Join(src, "cover.jpg"), []byte("jpg"), 0o644)
	os.WriteFile(filepath.Join(src, "readme.exe"), []byte("no"), 0o644)
	qb.finish(src)
	h.must(204, "POST", fmt.Sprintf("/api/v1/queue/%d/import", queue[0].ID), map[string]any{}, nil)

	h.must(200, "GET", "/api/v1/queue", nil, &queue)
	if len(queue) != 0 {
		t.Fatalf("queue should be empty after import: %+v", queue)
	}
	h.must(200, "GET", fmt.Sprintf("/api/v1/books/%d", justice.ID), nil, &book)
	wantDir := filepath.Join(root, "Ann Leckie", "Imperial Radch", "Ancillary Justice")
	if book.Status != model.StatusDownloaded || book.Path != wantDir || book.Format != "m4b" || len(book.Files) != 1 {
		t.Fatalf("book after import: status=%s path=%s format=%s files=%d", book.Status, book.Path, book.Format, len(book.Files))
	}
	if got, _ := os.ReadFile(filepath.Join(wantDir, "Ancillary Justice.m4b")); string(got) != "audio-bytes" {
		t.Fatalf("imported file content %q", got)
	}
	if _, err := os.Stat(filepath.Join(wantDir, "cover.jpg")); err != nil {
		t.Fatal("cover art should travel with the book")
	}
	if _, err := os.Stat(filepath.Join(wantDir, "readme.exe")); err == nil {
		t.Fatal("non-audio junk must not be imported")
	}
	// Torrents keep seeding: the source stays put.
	if _, err := os.Stat(filepath.Join(src, "Ancillary Justice.m4b")); err != nil {
		t.Fatal("torrent source was moved away")
	}
	types := map[string]bool{}
	for _, e := range book.History {
		types[e.Type] = true
	}
	for _, want := range []string{model.EventGrabbed, model.EventImportFailed, model.EventImported} {
		if !types[want] {
			t.Fatalf("history is missing a %s event: %+v", want, book.History)
		}
	}

	// --- library scan: a vanished file is noticed, a reappearing one re-matched
	stash := filepath.Join(t.TempDir(), "stash")
	os.Rename(wantDir, stash)
	if err := h.svc.ScanLibrary(ctx); err != nil {
		t.Fatal(err)
	}
	h.must(200, "GET", fmt.Sprintf("/api/v1/books/%d", justice.ID), nil, &book)
	if book.Status != model.StatusMissing || len(book.Files) != 0 {
		t.Fatalf("after deleting files: %s, %d files", book.Status, len(book.Files))
	}
	os.Rename(stash, wantDir)
	h.svc.ScanLibrary(ctx)
	h.must(200, "GET", fmt.Sprintf("/api/v1/books/%d", justice.ID), nil, &book)
	if book.Status != model.StatusDownloaded || len(book.Files) != 1 {
		t.Fatalf("after rescan: %s, %d files", book.Status, len(book.Files))
	}

	// --- an unmounted root must not wipe the library
	os.Rename(root, root+".gone")
	h.svc.ScanLibrary(ctx)
	os.Rename(root+".gone", root)
	h.must(200, "GET", fmt.Sprintf("/api/v1/books/%d", justice.ID), nil, &book)
	if book.Status != model.StatusDownloaded {
		t.Fatal("books were dropped while the root folder was unavailable")
	}

	// --- delete with files
	h.must(204, "DELETE", fmt.Sprintf("/api/v1/books/%d?deleteFiles=true", justice.ID), nil, nil)
	if _, err := os.Stat(wantDir); err == nil {
		t.Fatal("book folder should be gone")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("the root folder itself must survive")
	}
	h.must(404, "GET", fmt.Sprintf("/api/v1/books/%d", justice.ID), nil, nil)
}
