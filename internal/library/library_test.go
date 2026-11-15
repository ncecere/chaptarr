package library

import (
	"path/filepath"
	"testing"

	"github.com/ncecere/chaptarr/internal/model"
)

func TestBookFolder(t *testing.T) {
	series := &model.Book{
		Title: "Dune: Messiah?", AuthorName: "Frank Herbert", ASIN: "B01", ReleaseDate: "1969-10-15",
		Narrators: []string{"Scott Brick"}, Series: &model.SeriesRef{Name: "Dune", Position: "2"},
	}
	standalone := &model.Book{Title: "AC/DC: A History.", AuthorName: "J. R. R. Tolkien Jr.", ASIN: "B02"}
	cases := []struct {
		format string
		book   *model.Book
		want   string
	}{
		{"{Author}/{Series}/{Title}", series, "Frank Herbert/Dune/Dune- Messiah"},
		// Empty tokens drop their segment and the punctuation around them.
		{"{Author}/{Series}/{Title}", standalone, "J. R. R. Tolkien Jr/AC-DC- A History"},
		{"{Author}/{Series}/{SeriesPosition} - {Title} ({Year})", standalone, "J. R. R. Tolkien Jr/AC-DC- A History"},
		{"{Author}/{Series}/{SeriesPosition} - {Title} ({Year})", series, "Frank Herbert/Dune/2 - Dune- Messiah (1969)"},
		{"{Author} - {Title} [{ASIN}] {Narrator}", series, "Frank Herbert - Dune- Messiah [B01] Scott Brick"},
		{"", series, "Frank Herbert/Dune/Dune- Messiah"},
		// A template cannot climb out of the root folder.
		{"../../{Title}", series, "Dune- Messiah"},
	}
	for _, c := range cases {
		if got := BookFolder(c.format, c.book); got != filepath.FromSlash(c.want) {
			t.Errorf("BookFolder(%q) = %q, want %q", c.format, got, c.want)
		}
	}
	evil := &model.Book{Title: "..", AuthorName: "../../etc", ASIN: "B03"}
	if got := BookFolder("{Author}/{Title}", evil); got != filepath.FromSlash("..-..-etc") {
		t.Errorf("hostile metadata produced %q", got)
	}
}
