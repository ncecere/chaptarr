package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A series lists every recording and translation of each book; SeriesBooks
// must return one English edition per position, in order.
func TestSeriesBooks(t *testing.T) {
	type p = map[string]any
	book := func(asin, title, lang string, ratings int) p {
		return p{
			"asin": asin, "title": title, "language": lang, "content_delivery_type": "SinglePartBook",
			"authors": []p{{"asin": "A1", "name": "John Scalzi"}},
			"rating":  p{"overall_distribution": p{"num_ratings": ratings}},
		}
	}
	catalogue := map[string]p{
		"B1":  book("B1", "Old Man's War", "english", 900),
		"B1R": book("B1R", "Old Man's War: Book 1", "english", 12),
		"B1D": book("B1D", "Krieg der Klone", "german", 400),
		"B2":  book("B2", "The Ghost Brigades", "english", 700),
		"B45": book("B45", "The Sagan Diary", "english", 50),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/SERIES") {
			rel := func(asin, seq, sort string) p {
				return p{"asin": asin, "relationship_to_product": "child", "relationship_type": "series", "sequence": seq, "sort": sort}
			}
			_ = json.NewEncoder(w).Encode(p{"product": p{"relationships": []p{
				rel("B2", "2", "2"), rel("B45", "1.5", "3"), rel("B1R", "1", "1"), rel("B1D", "1", "1"), rel("B1", "1", "1"),
				{"asin": "PARENT", "relationship_to_product": "parent", "relationship_type": "series"},
			}}})
			return
		}
		var out []p
		for _, a := range strings.Split(r.URL.Query().Get("asins"), ",") {
			if b, ok := catalogue[a]; ok {
				out = append(out, b)
			}
		}
		_ = json.NewEncoder(w).Encode(p{"products": out, "total_results": len(out)})
	}))
	defer srv.Close()

	c := New()
	c.AudibleBase = srv.URL
	got, err := c.SeriesBooks(context.Background(), "us", "SERIES", "Old Man's War", "english")
	if err != nil {
		t.Fatal(err)
	}
	var have []string
	for _, b := range got {
		if b.SeriesASIN != "SERIES" || b.SeriesName != "Old Man's War" {
			t.Errorf("%s filed under %q/%q", b.ASIN, b.SeriesASIN, b.SeriesName)
		}
		have = append(have, b.ASIN+"#"+b.SeriesPosition)
	}
	if want := "B1#1 B2#2 B45#1.5"; strings.Join(have, " ") != want {
		t.Fatalf("got %v, want %s", have, want)
	}
}
