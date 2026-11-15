package indexer

import (
	"testing"

	"github.com/ncecere/chaptarr/internal/model"
)

func seeds(n int) *int { return &n }

func TestScore(t *testing.T) {
	book := &model.Book{Title: "Dune", AuthorName: "Frank Herbert", RuntimeMinutes: 1260}
	rel := func(title string, size int64) model.Release {
		return model.Release{Title: title, GUID: title, Protocol: model.ProtocolTorrent, SizeBytes: size, Seeders: seeds(10)}
	}
	const mb = 1 << 20
	out := Score([]model.Release{
		rel("Frank Herbert - Dune [MP3]", 600*mb),
		rel("Frank.Herbert.Dune.Unabridged.M4B", 600*mb),
		rel("Dune - A Documentary [M4B]", 600*mb),           // no author, short title
		rel("Frank Herbert - Dune Messiah [M4B]", 600*mb),   // different book, same words
		rel("Frank Herbert - Dune (Abridged) MP3", 600*mb),  // abridged
		rel("Frank Herbert - Dune epub", 2*mb),              // ebook
		rel("Frank Herbert - Dune Saga 1-6 M4B", 60_000*mb), // pack
		rel("Frank Herbert - Dune [M4B] failed", 600*mb),    // failed before
	}, Criteria{
		Book: book, PreferredFormats: []string{"m4b", "mp3"}, MinSeeders: 1,
		FailedGUIDs: map[string]bool{"Frank Herbert - Dune [M4B] failed": true},
		OtherTitles: []string{"Dune Messiah", "Children of Dune"},
	})

	accepted := map[string]bool{}
	for _, r := range out {
		accepted[r.Title] = len(r.Rejections) == 0
	}
	if out[0].Title != "Frank.Herbert.Dune.Unabridged.M4B" || out[0].Format != "m4b" {
		t.Errorf("best release is %q", out[0].Title)
	}
	if out[1].Title != "Frank Herbert - Dune [MP3]" {
		t.Errorf("second release is %q", out[1].Title)
	}
	for title, ok := range accepted {
		want := title == "Frank.Herbert.Dune.Unabridged.M4B" || title == "Frank Herbert - Dune [MP3]"
		if ok != want {
			t.Errorf("%q accepted=%v, want %v", title, ok, want)
		}
	}

	// The reverse direction: searching for the sequel must not be blocked
	// by the shorter title it contains.
	sequel := &model.Book{Title: "Dune Messiah", AuthorName: "Frank Herbert", RuntimeMinutes: 540}
	out = Score([]model.Release{rel("Frank Herbert - Dune Messiah [M4B]", 300*mb)},
		Criteria{Book: sequel, MinSeeders: 1, OtherTitles: []string{"Dune", "Children of Dune"}})
	if len(out[0].Rejections) != 0 {
		t.Errorf("sequel rejected: %v", out[0].Rejections)
	}
}
