package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ncecere/chaptarr/internal/model"
)

func serve(t *testing.T, routes map[string]string, check func(*http.Request)) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestITunesSource(t *testing.T) {
	item := `{"wrapperType":"audiobook","collectionId":1442041162,"collectionName":"Good Omens (Unabridged)",
		"artistName":"Terry Pratchett & Neil Gaiman","artworkUrl100":"https://img/x/100x100bb.jpg",
		"releaseDate":"2007-08-21T07:00:00Z","description":"<b>Funny.</b>"}`
	var country string
	base := serve(t, map[string]string{
		"/search": `{"results":[` + item + `]}`,
		"/lookup": `{"results":[` + item + `]}`,
	}, func(r *http.Request) { country = r.URL.Query().Get("country") })
	cfg := SourceConfig{Type: model.SourceITunes, URL: base + "/"}

	got, err := New().SearchSource(context.Background(), cfg, "uk", "english", "good omens")
	if err != nil || len(got) != 1 {
		t.Fatalf("search: %v %+v", err, got)
	}
	b := got[0]
	if b.ASIN != "it:1442041162" || b.Source != model.SourceITunes || b.Title != "Good Omens" ||
		len(b.Authors) != 2 || b.Authors[1].Name != "Neil Gaiman" ||
		b.CoverURL != "https://img/x/600x600bb.jpg" || b.ReleaseDate != "2007-08-21" {
		t.Fatalf("unexpected result %+v", b)
	}
	if country != "gb" {
		t.Errorf("uk region searched the %q store", country)
	}
	one, err := New().SourceBook(context.Background(), cfg, "us", "it:1442041162")
	if err != nil || one.Title != "Good Omens" {
		t.Fatalf("lookup: %v %+v", err, one)
	}
}

func TestOpenLibrarySource(t *testing.T) {
	doc := `{"key":"/works/OL1W","title":"Old Man's War","author_name":["John Scalzi"],
		"first_publish_year":2005,"cover_i":524402,"language":["heb","eng"]}`
	var query string
	base := serve(t, map[string]string{
		"/search.json":     `{"docs":[` + doc + `,{"key":"/works/OL2W","title":"No Author"}]}`,
		"/works/OL1W.json": `{"description":{"type":"/type/text","value":"John Perry joins the army."}}`,
	}, func(r *http.Request) {
		if r.URL.Path == "/search.json" {
			query = r.URL.Query().Get("q")
		}
	})
	cfg := SourceConfig{Type: model.SourceOpenLibrary, URL: base}

	got, err := New().SearchSource(context.Background(), cfg, "us", "english", "old mans war")
	if err != nil || len(got) != 1 {
		t.Fatalf("search: %v %+v", err, got)
	}
	b := got[0]
	if b.ASIN != "ol:OL1W" || b.Language != "english" || b.ReleaseDate != "2005-01-01" ||
		b.CoverURL != "https://covers.openlibrary.org/b/id/524402-L.jpg" {
		t.Fatalf("unexpected result %+v", b)
	}
	if !strings.HasSuffix(query, " language:eng") {
		t.Errorf("query %q does not restrict the language", query)
	}
	one, err := New().SourceBook(context.Background(), cfg, "us", "ol:OL1W")
	if err != nil || one.Description != "John Perry joins the army." {
		t.Fatalf("lookup: %v %+v", err, one)
	}
	if query != "key:/works/OL1W" {
		t.Errorf("lookup searched %q", query)
	}
}

func TestGoogleBooksSource(t *testing.T) {
	vol := `{"id":"abc","volumeInfo":{"title":"Dune","authors":["Frank Herbert"],"publishedDate":"1965",
		"description":"Spice.","language":"en","imageLinks":{"thumbnail":"http://books/img"}}}`
	var key, inURL string
	base := serve(t, map[string]string{
		"/volumes":     `{"items":[` + vol + `]}`,
		"/volumes/abc": vol,
	}, func(r *http.Request) { key, inURL = r.Header.Get("X-Goog-Api-Key"), r.URL.RawQuery })
	cfg := SourceConfig{Type: model.SourceGoogleBooks, URL: base, APIKey: "secret"}

	got, err := New().SearchSource(context.Background(), cfg, "us", "english", "dune")
	if err != nil || len(got) != 1 {
		t.Fatalf("search: %v %+v", err, got)
	}
	b := got[0]
	if b.ASIN != "gb:abc" || b.Language != "english" || b.ReleaseDate != "1965-01-01" || b.CoverURL != "https://books/img" {
		t.Fatalf("unexpected result %+v", b)
	}
	// The key travels in a header: in the URL it would reach error messages.
	if key != "secret" || strings.Contains(inURL, "secret") {
		t.Errorf("api key sent as header %q, query %q", key, inURL)
	}
	if _, err := New().SourceBook(context.Background(), cfg, "us", "gb:abc"); err != nil {
		t.Fatal(err)
	}
	cfg.APIKey = ""
	if _, err := New().SearchSource(context.Background(), cfg, "us", "", "dune"); err == nil {
		t.Error("search without a key should fail")
	}
}

func TestSameWork(t *testing.T) {
	a := model.BookResult{Title: "Good Omens", Authors: authorRefs("Neil Gaiman", "Terry Pratchett")}
	for title, want := range map[string]bool{
		"Good Omens: The Nice and Accurate Prophecies": true,
		"Good Omens (Full Cast)":                       true,
		"Bad Omens":                                    false,
	} {
		b := model.BookResult{Title: title, Authors: authorRefs("Terry Pratchett")}
		if SameWork(a, b) != want {
			t.Errorf("SameWork(%q) = %v, want %v", title, !want, want)
		}
	}
	if SameWork(a, model.BookResult{Title: "Good Omens", Authors: authorRefs("Someone Else")}) {
		t.Error("different authors matched")
	}
}
