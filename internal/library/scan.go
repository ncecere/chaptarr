package library

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/textutil"
)

// Found is a set of files on disk matched to a library book.
type Found struct {
	BookID int64
	Dir    string
	Files  []model.BookFile
}

// Scan walks the root folders looking for audio that belongs to books which
// have no files yet. A folder (or, for loose files, a file name) matches a
// book when it contains the book's title and the author's surname appears
// somewhere on the path below the root. known holds paths already tracked.
func Scan(roots []string, books []model.Book, known map[string]bool) []Found {
	type cand struct {
		book    *model.Book
		title   string // normalised main title
		surname string
	}
	var cands []cand
	for i := range books {
		b := &books[i]
		if b.Status == model.StatusDownloaded {
			continue
		}
		title := textutil.Normalize(textutil.MainTitle(b.Title))
		toks := textutil.Tokens(b.AuthorName)
		if title == "" || len(toks) == 0 {
			continue
		}
		cands = append(cands, cand{b, title, toks[len(toks)-1]})
	}
	// Longest title first, so "Dune Messiah" is tried before "Dune".
	sort.SliceStable(cands, func(i, j int) bool { return len(cands[i].title) > len(cands[j].title) })

	match := func(name, relPath string) *model.Book {
		pathTokens := textutil.Tokens(relPath)
		for _, c := range cands {
			if textutil.ContainsPhrase(name, c.title) && textutil.ContainsAll(pathTokens, []string{c.surname}) {
				return c.book
			}
		}
		return nil
	}

	byBook := map[int64]*Found{}
	add := func(b *model.Book, dir string, files []model.BookFile) {
		f := byBook[b.ID]
		if f == nil {
			f = &Found{BookID: b.ID, Dir: dir}
			byBook[b.ID] = f
		}
		f.Files = append(f.Files, files...)
	}

	for _, root := range roots {
		dirs := map[string][]model.BookFile{}
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // unreadable subtree: skip it, keep scanning
			}
			if d.IsDir() || strings.HasPrefix(d.Name(), ".") || !IsAudio(p) || known[p] {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			dir := filepath.Dir(p)
			dirs[dir] = append(dirs[dir], model.BookFile{Path: p, SizeBytes: info.Size(), Format: Format(p)})
			return nil
		})
		for dir, files := range dirs {
			rel, err := filepath.Rel(root, dir)
			if err != nil {
				continue
			}
			// "Author/Title/CD1/*.mp3": try the folder, then its parent.
			matched := false
			for d, r := dir, rel; r != "." && r != ""; d, r = filepath.Dir(d), filepath.Dir(r) {
				if b := match(filepath.Base(d), r); b != nil {
					add(b, d, files)
					matched = true
					break
				}
				if d != dir {
					break // only one level up
				}
			}
			if matched {
				continue
			}
			// Loose files: "Author/Title.m4b".
			for _, f := range files {
				name := strings.TrimSuffix(filepath.Base(f.Path), filepath.Ext(f.Path))
				if b := match(name, filepath.Join(rel, name)); b != nil {
					add(b, dir, []model.BookFile{f})
				}
			}
		}
	}

	out := make([]Found, 0, len(byBook))
	for _, f := range byBook {
		sort.Slice(f.Files, func(i, j int) bool { return f.Files[i].Path < f.Files[j].Path })
		out = append(out, *f)
	}
	return out
}
