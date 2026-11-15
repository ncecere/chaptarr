// Package textutil normalises titles and names so they can be compared
// across metadata, release names and folder names.
package textutil

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var stripMarks = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// Normalize lowercases, strips accents and apostrophes, and turns every other
// run of non-alphanumerics into a single space: "The Hitchhiker's Guide" and
// "the.hitchhikers.guide" both become "the hitchhikers guide".
func Normalize(s string) string {
	if t, _, err := transform.String(stripMarks, s); err == nil {
		s = t
	}
	var sb strings.Builder
	space := true
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(r)
			space = false
		case r == '\'' || r == '’' || r == '`':
			// "don't" -> "dont"
		case r == '&':
			if !space {
				sb.WriteByte(' ')
			}
			sb.WriteString("and ")
			space = true
		default:
			if !space {
				sb.WriteByte(' ')
				space = true
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// Tokens splits a normalised string into words.
func Tokens(s string) []string { return strings.Fields(Normalize(s)) }

var stopwords = map[string]bool{"the": true, "a": true, "an": true, "and": true, "of": true}

// SignificantTokens drops articles and joiners, unless nothing would be left.
func SignificantTokens(s string) []string {
	all := Tokens(s)
	var out []string
	for _, t := range all {
		if !stopwords[t] {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return all
	}
	return out
}

// ContainsAll reports whether every needle is present in haystack as a whole
// token.
func ContainsAll(haystack []string, needles []string) bool {
	set := make(map[string]bool, len(haystack))
	for _, h := range haystack {
		set[h] = true
	}
	for _, n := range needles {
		if !set[n] {
			return false
		}
	}
	return true
}

// ContainsPhrase reports whether needle appears in haystack as a contiguous
// run of whole words. Both are normalised first.
func ContainsPhrase(haystack, needle string) bool {
	h, n := " "+Normalize(haystack)+" ", " "+Normalize(needle)+" "
	return n != "  " && strings.Contains(h, n)
}

// MainTitle cuts a title at its subtitle separator: "Dune: Book One" -> "Dune".
func MainTitle(title string) string {
	for _, sep := range []string{": ", " - ", " ("} {
		if i := strings.Index(title, sep); i > 0 {
			title = title[:i]
		}
	}
	return strings.TrimSpace(title)
}
