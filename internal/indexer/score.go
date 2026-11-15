package indexer

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/textutil"
)

// AudioFormats are the containers treated as audiobook files, on disk and in
// release names.
var AudioFormats = []string{"m4b", "mp3", "m4a", "flac", "opus", "ogg", "aac", "wma", "wav", "mp4", "mka"}

var ebookWords = map[string]bool{
	"epub": true, "mobi": true, "azw3": true, "azw": true, "pdf": true, "ebook": true, "cbz": true, "cbr": true,
	"kindle": true,
}

// Queries builds the indexer searches for a book, most specific first.
func Queries(b *model.Book) []string {
	title := textutil.MainTitle(b.Title)
	qs := []string{strings.TrimSpace(title + " " + b.AuthorName)}
	// Many trackers name releases "Author - Series 01 - Title", and some
	// search engines choke on the author's middle initials, so also try the
	// bare title when it is distinctive enough not to flood the results.
	if len(textutil.SignificantTokens(title)) >= 2 {
		qs = append(qs, title)
	}
	return qs
}

// Criteria is what a release is judged against.
type Criteria struct {
	Book             *model.Book
	PreferredFormats []string
	MinSeeders       int
	// FailedGUIDs are releases that already failed for this book.
	FailedGUIDs map[string]bool
	// OtherTitles are the author's other books. A release naming one of
	// them is a different book or a multi-book pack: "Dune Messiah" must
	// not satisfy a search for "Dune".
	OtherTitles []string
}

var abridgedRe = regexp.MustCompile(`(?i)\babridged\b`)
var unabridgedRe = regexp.MustCompile(`(?i)\bunabridged\b`)

// DetectFormat finds an audio format named in a release title.
func DetectFormat(title string) string {
	tokens := textutil.Tokens(title)
	set := map[string]bool{}
	for _, t := range tokens {
		set[t] = true
	}
	for _, f := range AudioFormats {
		if set[f] {
			return f
		}
	}
	return ""
}

// Score fills in Format, Score and Rejections on each release and sorts them
// best first. A release with any rejection is never grabbed automatically.
func Score(releases []model.Release, c Criteria) []model.Release {
	b := c.Book
	titleTokens := textutil.SignificantTokens(textutil.MainTitle(b.Title))
	authorTokens := textutil.Tokens(b.AuthorName)
	var surname string
	if len(authorTokens) > 0 {
		surname = authorTokens[len(authorTokens)-1]
	}

	for i := range releases {
		r := &releases[i]
		r.Rejections = []string{}
		tokens := textutil.Tokens(r.Title)
		score := 0

		titleMatch := textutil.ContainsAll(tokens, titleTokens)
		if titleMatch {
			score += 100
			if textutil.ContainsPhrase(r.Title, textutil.MainTitle(b.Title)) {
				score += 20
			}
		} else {
			r.Rejections = append(r.Rejections, "Release name does not contain the book title")
		}

		authorMatch := surname != "" && textutil.ContainsAll(tokens, []string{surname})
		if authorMatch {
			score += 30
		} else if len(titleTokens) <= 2 {
			// "Dune" or "The Stand" match far too much on their own.
			r.Rejections = append(r.Rejections, "Release name does not mention the author")
		}

		for _, t := range tokens {
			if ebookWords[t] {
				r.Rejections = append(r.Rejections, "Looks like an ebook, not an audiobook")
				break
			}
		}
		for _, other := range c.OtherTitles {
			other = textutil.MainTitle(other)
			// Skip titles contained in this book's own ("Dune" when
			// searching for "Dune Messiah").
			if textutil.ContainsPhrase(b.Title, other) {
				continue
			}
			if textutil.ContainsPhrase(r.Title, other) {
				r.Rejections = append(r.Rejections, fmt.Sprintf("Names another book by the author (%s)", other))
				break
			}
		}
		if abridgedRe.MatchString(r.Title) && !unabridgedRe.MatchString(r.Title) {
			r.Rejections = append(r.Rejections, "Abridged")
		}

		r.Format = DetectFormat(r.Title)
		if r.Format != "" {
			rank := -1
			for i, f := range c.PreferredFormats {
				if f == r.Format {
					rank = i
					break
				}
			}
			if rank >= 0 {
				bonus := 40 - 8*rank
				if bonus < 8 {
					bonus = 8
				}
				score += bonus
			}
		}

		if r.Protocol == model.ProtocolTorrent {
			if r.Seeders != nil {
				if *r.Seeders < c.MinSeeders {
					r.Rejections = append(r.Rejections, fmt.Sprintf("Only %d seeders (minimum %d)", *r.Seeders, c.MinSeeders))
				}
				s := *r.Seeders
				if s > 50 {
					s = 50
				}
				score += s / 5
			}
		} else {
			score += 5
		}

		// Sanity-check the size against the runtime: below ~16 kbps it is a
		// sample or an ebook, above ~1.4 Mbps (lossless stereo) it is
		// probably a multi-book pack or something else entirely.
		if b.RuntimeMinutes > 0 && r.SizeBytes > 0 {
			perMin := r.SizeBytes / int64(b.RuntimeMinutes)
			switch {
			case perMin < 120_000:
				r.Rejections = append(r.Rejections, "Too small for the book's runtime")
			case perMin > 11_000_000:
				r.Rejections = append(r.Rejections, "Too large for the book's runtime (pack or wrong item?)")
			}
		}

		if c.FailedGUIDs[r.GUID] {
			r.Rejections = append(r.Rejections, "This release failed before")
		}
		r.Score = score
	}

	sort.SliceStable(releases, func(i, j int) bool {
		a, b := releases[i], releases[j]
		if (len(a.Rejections) == 0) != (len(b.Rejections) == 0) {
			return len(a.Rejections) == 0
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return a.SizeBytes > b.SizeBytes
	})
	return releases
}
