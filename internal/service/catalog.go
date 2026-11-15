package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ncecere/chaptarr/internal/library"
	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/store"
	"github.com/ncecere/chaptarr/internal/textutil"
)

// ---------------------------------------------------------------- metadata search

func (s *Service) SearchBooks(ctx context.Context, q string) ([]model.BookResult, error) {
	set, err := s.Store.Settings()
	if err != nil {
		return nil, err
	}
	results, err := s.Meta.SearchBooks(ctx, set.AudibleRegion, q)
	if err != nil {
		return nil, upstream("Book search failed: %v", err)
	}
	ids, err := s.Store.BookIDsByASIN()
	if err != nil {
		return nil, err
	}
	for i := range results {
		if id, ok := ids[results[i].ASIN]; ok {
			results[i].LibraryBookID = &id
		}
	}
	return results, nil
}

func (s *Service) SearchAuthors(ctx context.Context, q string) ([]model.AuthorResult, error) {
	set, err := s.Store.Settings()
	if err != nil {
		return nil, err
	}
	results, err := s.Meta.SearchAuthors(ctx, set.AudibleRegion, q)
	if err != nil {
		return nil, upstream("Author search failed: %v", err)
	}
	authors, err := s.Store.Authors()
	if err != nil {
		return nil, err
	}
	ids := map[string]int64{}
	for _, a := range authors {
		ids[a.ASIN] = a.ID
	}
	for i := range results {
		if id, ok := ids[results[i].ASIN]; ok {
			results[i].LibraryAuthorID = &id
		}
	}
	return results, nil
}

// ---------------------------------------------------------------- authors

type AddAuthorRequest struct {
	ASIN            string `json:"asin"`
	Name            string `json:"name"`
	Monitor         string `json:"monitor"`
	MonitorNewBooks bool   `json:"monitorNewBooks"`
	SearchOnAdd     bool   `json:"searchOnAdd"`
}

func (s *Service) AuthorDetail(id int64) (*model.AuthorDetail, error) {
	a, err := s.Store.Author(id)
	if err != nil {
		return nil, notFound(err)
	}
	books, err := s.Store.Books(model.BookFilter{AuthorID: id})
	if err != nil {
		return nil, err
	}
	return &model.AuthorDetail{Author: *a, Books: books}, nil
}

func (s *Service) AddAuthor(ctx context.Context, r AddAuthorRequest) (*model.AuthorDetail, error) {
	if r.ASIN == "" || r.Name == "" {
		return nil, badRequest("An author ASIN and name are required")
	}
	var monitorBook func(model.BookResult) bool
	switch r.Monitor {
	case "all":
		monitorBook = func(model.BookResult) bool { return true }
	case "future":
		today := time.Now().Format("2006-01-02")
		monitorBook = func(b model.BookResult) bool { return b.ReleaseDate > today }
	case "none":
		monitorBook = func(model.BookResult) bool { return false }
	default:
		return nil, badRequest("monitor must be one of all, future, none")
	}
	monitored := r.Monitor != "none"

	// An author can already exist, unmonitored, because one of their books
	// was added on its own. Adding them properly upgrades that record.
	a, err := s.Store.AuthorByASIN(r.ASIN)
	switch {
	case err == nil:
		if err := s.Store.SetAuthorFlags(a.ID, &monitored, &r.MonitorNewBooks); err != nil {
			return nil, err
		}
	case errors.Is(err, store.ErrNotFound):
		a = &model.Author{ASIN: r.ASIN, Name: r.Name, Monitored: monitored, MonitorNewBooks: r.MonitorNewBooks}
		if err := s.Store.CreateAuthor(a); err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	if err := s.syncAuthor(ctx, a, monitorBook); err != nil {
		// The author is in; the catalogue can be pulled again with Refresh.
		s.Log.Warn("author added but catalogue import failed", "author", a.Name, "error", err)
		return nil, upstream("%s was added, but importing their books failed: %v. Use Refresh on the author page to try again.", a.Name, err)
	}
	s.Log.Info("author added", "author", a.Name)
	if r.SearchOnAdd {
		s.SearchMissingInBackground(a.ID)
	}
	return s.AuthorDetail(a.ID)
}

func (s *Service) RefreshAuthor(ctx context.Context, id int64) (*model.AuthorDetail, error) {
	a, err := s.Store.Author(id)
	if err != nil {
		return nil, notFound(err)
	}
	monitorNew := a.Monitored && a.MonitorNewBooks
	if err := s.syncAuthor(ctx, a, func(model.BookResult) bool { return monitorNew }); err != nil {
		return nil, upstream("Refreshing %s failed: %v", a.Name, err)
	}
	return s.AuthorDetail(id)
}

// syncAuthor pulls the author's bio and catalogue, adding books that are new
// and refreshing the metadata of those already in the library.
func (s *Service) syncAuthor(ctx context.Context, a *model.Author, monitorNew func(model.BookResult) bool) error {
	set, err := s.Store.Settings()
	if err != nil {
		return err
	}
	// The bio and photo are nice to have; the catalogue is what matters.
	if strings.HasPrefix(a.ASIN, "name:") {
		// No catalogue id, so no bio to fetch.
	} else if info, err := s.Meta.Author(ctx, set.AudibleRegion, a.ASIN); err == nil {
		name := a.Name
		if info.Name != "" {
			name = info.Name
		}
		if err := s.Store.UpdateAuthorMeta(a.ID, name, info.Description, info.ImageURL); err != nil {
			return err
		}
		a.Name = name
	} else {
		s.Log.Warn("author details lookup failed", "author", a.Name, "error", err)
	}

	// An author known only by name (no catalogue id) is matched by name.
	asin := a.ASIN
	if strings.HasPrefix(asin, "name:") {
		asin = ""
	}
	results, err := s.Meta.AuthorBooks(ctx, set.AudibleRegion, asin, a.Name, set.Language)
	if err != nil {
		return err
	}
	existing, err := s.Store.Books(model.BookFilter{AuthorID: a.ID})
	if err != nil {
		return err
	}
	titles := map[string]bool{}
	for _, b := range existing {
		titles[textutil.Normalize(b.Title)] = true
	}
	added := 0
	for _, r := range results {
		meta, err := s.bookMeta(r)
		if err != nil {
			return err
		}
		if b, err := s.Store.BookByASIN(r.ASIN); err == nil {
			if err := s.Store.UpdateBookMeta(b.ID, meta); err != nil {
				return err
			}
			continue
		} else if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		// The library may hold another edition of this title (one the user
		// picked by hand); do not add a second copy of the same work.
		if titles[textutil.Normalize(r.Title)] {
			continue
		}
		if _, err := s.Store.CreateBook(a.ID, meta, monitorNew(r)); err != nil {
			return err
		}
		titles[textutil.Normalize(r.Title)] = true
		added++
	}
	if added > 0 {
		s.Log.Info("new books found", "author", a.Name, "count", added)
	}
	return s.Store.TouchAuthorSync(a.ID)
}

func (s *Service) bookMeta(r model.BookResult) (store.BookMeta, error) {
	m := store.BookMeta{
		ASIN: r.ASIN, Title: r.Title, Subtitle: r.Subtitle, Narrators: r.Narrators,
		SeriesPosition: r.SeriesPosition, Description: r.Description, CoverURL: r.CoverURL,
		ReleaseDate: r.ReleaseDate, Language: r.Language, Publisher: r.Publisher, RuntimeMinutes: r.RuntimeMinutes,
		Authors: []string{},
	}
	for _, a := range r.Authors {
		m.Authors = append(m.Authors, a.Name)
	}
	if r.SeriesName != "" {
		id, err := s.Store.EnsureSeries(r.SeriesASIN, r.SeriesName)
		if err != nil {
			return m, err
		}
		m.SeriesID = &id
	}
	return m, nil
}

func (s *Service) DeleteAuthor(id int64, deleteFiles bool) error {
	d, err := s.AuthorDetail(id)
	if err != nil {
		return err
	}
	for _, b := range d.Books {
		if err := s.DeleteBook(b.ID, deleteFiles); err != nil {
			return err
		}
	}
	return s.Store.DeleteAuthor(id)
}

// ---------------------------------------------------------------- books

type AddBookRequest struct {
	ASIN        string `json:"asin"`
	Monitored   bool   `json:"monitored"`
	SearchOnAdd bool   `json:"searchOnAdd"`
}

func (s *Service) AddBook(ctx context.Context, r AddBookRequest) (*model.Book, error) {
	if r.ASIN == "" {
		return nil, badRequest("A book ASIN is required")
	}
	if _, err := s.Store.BookByASIN(r.ASIN); err == nil {
		return nil, conflict("That book is already in the library")
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	set, err := s.Store.Settings()
	if err != nil {
		return nil, err
	}
	res, err := s.Meta.Book(ctx, set.AudibleRegion, r.ASIN)
	if err != nil {
		return nil, upstream("Could not load the book: %v", err)
	}
	id, err := s.addResult(ctx, set, *res, r.Monitored)
	if err != nil {
		return nil, err
	}
	if r.SearchOnAdd {
		s.searchInBackground([]int64{id})
	}
	return s.Store.Book(id)
}

// addResult files a catalogue entry under its lead author, creating the
// author (unmonitored) if this is their first book in the library.
func (s *Service) addResult(ctx context.Context, set model.Settings, res model.BookResult, monitored bool) (int64, error) {
	if len(res.Authors) == 0 {
		return 0, upstream("The catalogue lists no author for %q", res.Title)
	}
	// File the book under its first credited author who has a catalogue id.
	lead := res.Authors[0]
	for _, a := range res.Authors {
		if a.ASIN != "" {
			lead = a
			break
		}
	}
	key := lead.ASIN
	if key == "" {
		key = "name:" + textutil.Normalize(lead.Name)
	}
	author, err := s.Store.AuthorByASIN(key)
	if errors.Is(err, store.ErrNotFound) {
		author = &model.Author{ASIN: key, Name: lead.Name}
		if err = s.Store.CreateAuthor(author); err == nil && lead.ASIN != "" {
			if info, ierr := s.Meta.Author(ctx, set.AudibleRegion, lead.ASIN); ierr == nil {
				err = s.Store.UpdateAuthorMeta(author.ID, lead.Name, info.Description, info.ImageURL)
			}
		}
	}
	if err != nil {
		return 0, err
	}
	meta, err := s.bookMeta(res)
	if err != nil {
		return 0, err
	}
	id, err := s.Store.CreateBook(author.ID, meta, monitored)
	if err != nil {
		return 0, err
	}
	s.Log.Info("book added", "title", res.Title, "author", author.Name)
	return id, nil
}

// searchInBackground searches for each book in turn, pausing between them
// as the scheduled search does.
func (s *Service) searchInBackground(ids []int64) {
	go func() {
		for i, id := range ids {
			if i > 0 {
				time.Sleep(2 * time.Second)
			}
			if _, msg, err := s.SearchBook(context.Background(), id); err != nil {
				s.Log.Warn("search on add failed", "bookId", id, "error", err)
				// No indexer or client: the rest would fail the same way.
				if e, ok := err.(*Error); ok && e.Status == 400 {
					return
				}
			} else {
				s.Log.Info("search on add", "bookId", id, "result", msg)
			}
		}
	}()
}

// ---------------------------------------------------------------- series

// SeriesMissing lists the books of a series that are not in the library.
func (s *Service) SeriesMissing(ctx context.Context, id int64) ([]model.BookResult, error) {
	d, err := s.Store.SeriesDetail(id)
	if err != nil {
		return nil, notFound(err)
	}
	if strings.HasPrefix(d.ASIN, "name:") {
		return nil, badRequest("The catalogue has no listing for the series %q, so its other books cannot be looked up.", d.Name)
	}
	set, err := s.Store.Settings()
	if err != nil {
		return nil, err
	}
	all, err := s.Meta.SeriesBooks(ctx, set.AudibleRegion, d.ASIN, d.Name, set.Language)
	if err != nil {
		return nil, upstream("Could not load the series: %v", err)
	}
	inLibrary, err := s.Store.BookIDsByASIN()
	if err != nil {
		return nil, err
	}
	// The library may hold another edition of a title, picked by hand.
	titles, positions := map[string]bool{}, map[string]bool{}
	for _, b := range d.Books {
		titles[textutil.Normalize(b.Title)] = true
		if b.Series != nil && b.Series.Position != "" {
			positions[b.Series.Position] = true
		}
	}
	missing := []model.BookResult{}
	for _, r := range all {
		if _, ok := inLibrary[r.ASIN]; ok || titles[textutil.Normalize(r.Title)] || positions[r.SeriesPosition] {
			continue
		}
		missing = append(missing, r)
	}
	return missing, nil
}

type AddSeriesBooksRequest struct {
	// ASINs limits what is added; empty adds every missing book.
	ASINs       []string `json:"asins"`
	Monitored   bool     `json:"monitored"`
	SearchOnAdd bool     `json:"searchOnAdd"`
}

// AddSeriesBooks adds the series' missing books and returns how many.
func (s *Service) AddSeriesBooks(ctx context.Context, id int64, r AddSeriesBooksRequest) (int, error) {
	missing, err := s.SeriesMissing(ctx, id)
	if err != nil {
		return 0, err
	}
	set, err := s.Store.Settings()
	if err != nil {
		return 0, err
	}
	want := map[string]bool{}
	for _, a := range r.ASINs {
		want[a] = true
	}
	var ids []int64
	for _, res := range missing {
		if len(want) > 0 && !want[res.ASIN] {
			continue
		}
		bookID, err := s.addResult(ctx, set, res, r.Monitored)
		if err != nil {
			return len(ids), err
		}
		ids = append(ids, bookID)
	}
	if r.SearchOnAdd && r.Monitored {
		s.searchInBackground(ids)
	}
	return len(ids), nil
}

func (s *Service) BookDetail(id int64) (*model.BookDetail, error) {
	b, err := s.Store.Book(id)
	if err != nil {
		return nil, notFound(err)
	}
	d := &model.BookDetail{Book: *b}
	if d.Files, err = s.Store.BookFiles(id); err != nil {
		return nil, err
	}
	if d.History, err = s.Store.BookHistory(id); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) DeleteBook(id int64, deleteFiles bool) error {
	b, err := s.Store.Book(id)
	if err != nil {
		return notFound(err)
	}
	if deleteFiles {
		files, err := s.Store.BookFiles(id)
		if err != nil {
			return err
		}
		if len(files) > 0 {
			roots, err := s.rootPaths()
			if err != nil {
				return err
			}
			if err := library.DeleteBookFiles(files, b.Path, roots); err != nil {
				return &Error{500, "Could not delete the book's files: " + err.Error()}
			}
			_ = s.Store.AddHistory(b, model.EventFileDeleted, b.Path, "", "Files deleted with the book")
		}
	}
	return s.Store.DeleteBook(id)
}

func (s *Service) rootPaths() ([]string, error) {
	folders, err := s.Store.RootFolders()
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(folders))
	for i, f := range folders {
		paths[i] = f.Path
	}
	return paths, nil
}
