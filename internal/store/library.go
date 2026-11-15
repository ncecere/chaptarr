package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ncecere/chaptarr/internal/model"
)

// ---------------------------------------------------------------- authors

// A book counts as missing when it is monitored, released, has no files and
// has no live download. The same rule is applied in Go by bookStatus.
const authorSelect = `SELECT a.id, a.asin, a.name, a.description, a.image_url, a.monitored,
	a.monitor_new_books, a.added_at, a.last_sync_at,
	(SELECT COUNT(*) FROM books b WHERE b.author_id = a.id),
	(SELECT COUNT(*) FROM books b WHERE b.author_id = a.id
		AND EXISTS (SELECT 1 FROM book_files f WHERE f.book_id = b.id)),
	(SELECT COUNT(*) FROM books b WHERE b.author_id = a.id AND b.monitored = 1
		AND b.release_date <= ?1
		AND NOT EXISTS (SELECT 1 FROM book_files f WHERE f.book_id = b.id)
		AND NOT EXISTS (SELECT 1 FROM downloads d WHERE d.book_id = b.id AND d.status != 'failed'))
	FROM authors a`

type scanner interface{ Scan(dest ...any) error }

func scanAuthor(row scanner) (*model.Author, error) {
	var a model.Author
	var added int64
	var sync sql.NullInt64
	err := row.Scan(&a.ID, &a.ASIN, &a.Name, &a.Description, &a.ImageURL, &a.Monitored,
		&a.MonitorNewBooks, &added, &sync, &a.BookCount, &a.DownloadedCount, &a.MissingCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.AddedAt = fromUnix(added)
	if sync.Valid {
		t := fromUnix(sync.Int64)
		a.LastSyncAt = &t
	}
	return &a, nil
}

func (s *Store) Authors() ([]model.Author, error) {
	rows, err := s.db.Query(authorSelect+` ORDER BY a.name COLLATE NOCASE`, today())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Author{}
	for rows.Next() {
		a, err := scanAuthor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (s *Store) Author(id int64) (*model.Author, error) {
	return scanAuthor(s.db.QueryRow(authorSelect+` WHERE a.id = ?2`, today(), id))
}

func (s *Store) AuthorByASIN(asin string) (*model.Author, error) {
	return scanAuthor(s.db.QueryRow(authorSelect+` WHERE a.asin = ?2`, today(), asin))
}

func (s *Store) CreateAuthor(a *model.Author) error {
	a.AddedAt = time.Now().UTC()
	res, err := s.db.Exec(`INSERT INTO authors (asin, name, description, image_url, monitored, monitor_new_books, added_at)
		VALUES (?,?,?,?,?,?,?)`, a.ASIN, a.Name, a.Description, a.ImageURL, b2i(a.Monitored), b2i(a.MonitorNewBooks), unix(a.AddedAt))
	if err != nil {
		return err
	}
	a.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateAuthorMeta(id int64, name, description, imageURL string) error {
	_, err := s.db.Exec(`UPDATE authors SET name = ?, description = ?, image_url = ? WHERE id = ?`,
		name, description, imageURL, id)
	return err
}

func (s *Store) SetAuthorFlags(id int64, monitored, monitorNew *bool) error {
	if monitored != nil {
		if _, err := s.db.Exec(`UPDATE authors SET monitored = ? WHERE id = ?`, b2i(*monitored), id); err != nil {
			return err
		}
	}
	if monitorNew != nil {
		if _, err := s.db.Exec(`UPDATE authors SET monitor_new_books = ? WHERE id = ?`, b2i(*monitorNew), id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) TouchAuthorSync(id int64) error {
	_, err := s.db.Exec(`UPDATE authors SET last_sync_at = ? WHERE id = ?`, time.Now().Unix(), id)
	return err
}

func (s *Store) DeleteAuthor(id int64) error {
	_, err := s.db.Exec(`DELETE FROM authors WHERE id = ?`, id)
	if err == nil {
		err = s.pruneSeries()
	}
	return err
}

// ---------------------------------------------------------------- series

// EnsureSeries returns the id of the series, creating it if needed. Audible
// series carry an ASIN; when one is missing the name is the key.
func (s *Store) EnsureSeries(asin, name string) (int64, error) {
	if asin == "" {
		asin = "name:" + strings.ToLower(name)
	}
	if _, err := s.db.Exec(`INSERT INTO series (asin, name) VALUES (?,?)
		ON CONFLICT(asin) DO UPDATE SET name = excluded.name`, asin, name); err != nil {
		return 0, err
	}
	var id int64
	err := s.db.QueryRow(`SELECT id FROM series WHERE asin = ?`, asin).Scan(&id)
	return id, err
}

func (s *Store) pruneSeries() error {
	_, err := s.db.Exec(`DELETE FROM series WHERE NOT EXISTS (SELECT 1 FROM books b WHERE b.series_id = series.id)`)
	return err
}

func (s *Store) SeriesList() ([]model.Series, error) {
	rows, err := s.db.Query(`SELECT s.id, s.asin, s.name FROM series s ORDER BY s.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	out := []model.Series{}
	for rows.Next() {
		var x model.Series
		if err := rows.Scan(&x.ID, &x.ASIN, &x.Name); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	books, err := s.Books(model.BookFilter{})
	if err != nil {
		return nil, err
	}
	bySeries := map[int64][]model.Book{}
	for _, b := range books {
		if b.Series != nil {
			bySeries[b.Series.ID] = append(bySeries[b.Series.ID], b)
		}
	}
	res := out[:0]
	for _, x := range out {
		bs := bySeries[x.ID]
		if len(bs) == 0 {
			continue
		}
		fillSeries(&x, bs)
		res = append(res, x)
	}
	return res, nil
}

func fillSeries(x *model.Series, books []model.Book) {
	SortBySeriesPosition(books)
	x.BookCount = len(books)
	x.CoverURL = books[0].CoverURL
	x.AuthorName = books[0].AuthorName
	for _, b := range books {
		if b.Status == model.StatusDownloaded {
			x.DownloadedCount++
		}
	}
	if strings.HasPrefix(x.ASIN, "name:") {
		x.ASIN = ""
	}
}

func (s *Store) SeriesDetail(id int64) (*model.SeriesDetail, error) {
	var d model.SeriesDetail
	err := s.db.QueryRow(`SELECT id, asin, name FROM series WHERE id = ?`, id).Scan(&d.ID, &d.ASIN, &d.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d.Books, err = s.Books(model.BookFilter{SeriesID: id})
	if err != nil {
		return nil, err
	}
	if len(d.Books) > 0 {
		fillSeries(&d.Series, d.Books)
	}
	return &d, nil
}

// positionKey turns "2", "2.5" or "1-3" into a sortable number; unknown
// positions sort last.
func positionKey(p string) float64 {
	p = strings.TrimSpace(p)
	if i := strings.IndexAny(p, "-,"); i > 0 {
		p = p[:i]
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
	if err != nil {
		return 1e9
	}
	return f
}

func SortBySeriesPosition(books []model.Book) {
	sort.SliceStable(books, func(i, j int) bool {
		var pi, pj string
		if books[i].Series != nil {
			pi = books[i].Series.Position
		}
		if books[j].Series != nil {
			pj = books[j].Series.Position
		}
		ki, kj := positionKey(pi), positionKey(pj)
		if ki != kj {
			return ki < kj
		}
		return books[i].ReleaseDate < books[j].ReleaseDate
	})
}

// ---------------------------------------------------------------- books

const bookSelect = `SELECT b.id, b.asin, b.title, b.subtitle, b.author_id, a.name, b.authors, b.narrators,
	b.series_id, COALESCE(s.name, ''), b.series_position, b.description, b.cover_url, b.release_date,
	b.runtime_minutes, b.language, b.publisher, b.monitored, b.path, b.size_bytes, b.format, b.added_at,
	EXISTS (SELECT 1 FROM book_files f WHERE f.book_id = b.id),
	EXISTS (SELECT 1 FROM downloads d WHERE d.book_id = b.id AND d.status != 'failed')
	FROM books b JOIN authors a ON a.id = b.author_id LEFT JOIN series s ON s.id = b.series_id`

func scanBook(row scanner) (*model.Book, error) {
	var b model.Book
	var authors, narrators, seriesName string
	var seriesID sql.NullInt64
	var position string
	var added int64
	var hasFiles, downloading bool
	err := row.Scan(&b.ID, &b.ASIN, &b.Title, &b.Subtitle, &b.AuthorID, &b.AuthorName, &authors, &narrators,
		&seriesID, &seriesName, &position, &b.Description, &b.CoverURL, &b.ReleaseDate,
		&b.RuntimeMinutes, &b.Language, &b.Publisher, &b.Monitored, &b.Path, &b.SizeBytes, &b.Format, &added,
		&hasFiles, &downloading)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	b.Authors = []string{}
	b.Narrators = []string{}
	_ = json.Unmarshal([]byte(authors), &b.Authors)
	_ = json.Unmarshal([]byte(narrators), &b.Narrators)
	if seriesID.Valid {
		b.Series = &model.SeriesRef{ID: seriesID.Int64, Name: seriesName, Position: position}
	}
	b.AddedAt = fromUnix(added)
	switch {
	case hasFiles:
		b.Status = model.StatusDownloaded
	case downloading:
		b.Status = model.StatusDownloading
	case b.ReleaseDate > today():
		b.Status = model.StatusUnreleased
	default:
		b.Status = model.StatusMissing
	}
	return &b, nil
}

func (s *Store) Books(f model.BookFilter) ([]model.Book, error) {
	q := bookSelect + ` WHERE 1=1`
	var args []any
	if f.AuthorID != 0 {
		q += ` AND b.author_id = ?`
		args = append(args, f.AuthorID)
	}
	if f.SeriesID != 0 {
		q += ` AND b.series_id = ?`
		args = append(args, f.SeriesID)
	}
	if f.Monitored != nil {
		q += ` AND b.monitored = ?`
		args = append(args, b2i(*f.Monitored))
	}
	if f.Query != "" {
		q += ` AND (b.title LIKE ? OR b.authors LIKE ? OR b.narrators LIKE ? OR s.name LIKE ?)`
		like := "%" + f.Query + "%"
		args = append(args, like, like, like, like)
	}
	q += ` ORDER BY b.title COLLATE NOCASE`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Book{}
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		if f.Status != "" && b.Status != f.Status {
			continue
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (s *Store) Book(id int64) (*model.Book, error) {
	return scanBook(s.db.QueryRow(bookSelect+` WHERE b.id = ?`, id))
}

func (s *Store) BookByASIN(asin string) (*model.Book, error) {
	return scanBook(s.db.QueryRow(bookSelect+` WHERE b.asin = ?`, asin))
}

// BookIDsByASIN maps the given ASINs to library book ids.
func (s *Store) BookIDsByASIN() (map[string]int64, error) {
	rows, err := s.db.Query(`SELECT asin, id FROM books`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var asin string
		var id int64
		if err := rows.Scan(&asin, &id); err != nil {
			return nil, err
		}
		out[asin] = id
	}
	return out, rows.Err()
}

// BookMeta is the provider-owned part of a book, written on add and refresh.
type BookMeta struct {
	ASIN, Title, Subtitle            string
	Authors, Narrators               []string
	SeriesID                         *int64
	SeriesPosition                   string
	Description, CoverURL            string
	ReleaseDate, Language, Publisher string
	RuntimeMinutes                   int
}

func (s *Store) CreateBook(authorID int64, m BookMeta, monitored bool) (int64, error) {
	authors, _ := json.Marshal(m.Authors)
	narrators, _ := json.Marshal(m.Narrators)
	res, err := s.db.Exec(`INSERT INTO books (asin, title, subtitle, author_id, authors, narrators, series_id,
		series_position, description, cover_url, release_date, runtime_minutes, language, publisher, monitored, added_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		m.ASIN, m.Title, m.Subtitle, authorID, string(authors), string(narrators), m.SeriesID,
		m.SeriesPosition, m.Description, m.CoverURL, m.ReleaseDate, m.RuntimeMinutes, m.Language, m.Publisher,
		b2i(monitored), time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateBookMeta(id int64, m BookMeta) error {
	authors, _ := json.Marshal(m.Authors)
	narrators, _ := json.Marshal(m.Narrators)
	_, err := s.db.Exec(`UPDATE books SET title = ?, subtitle = ?, authors = ?, narrators = ?, series_id = ?,
		series_position = ?, description = ?, cover_url = ?, release_date = ?, runtime_minutes = ?, language = ?,
		publisher = ? WHERE id = ?`,
		m.Title, m.Subtitle, string(authors), string(narrators), m.SeriesID, m.SeriesPosition, m.Description,
		m.CoverURL, m.ReleaseDate, m.RuntimeMinutes, m.Language, m.Publisher, id)
	return err
}

func (s *Store) SetBooksMonitored(ids []int64, monitored bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if _, err := tx.Exec(`UPDATE books SET monitored = ? WHERE id = ?`, b2i(monitored), id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) TouchBookSearch(id int64) error {
	_, err := s.db.Exec(`UPDATE books SET last_search_at = ? WHERE id = ?`, time.Now().Unix(), id)
	return err
}

// MissingBookIDs returns monitored, released books with no files and no live
// download, least recently searched first. Books searched after notSince are
// skipped so a scheduled run does not hammer indexers for the same titles.
func (s *Store) MissingBookIDs(authorID int64, notSince time.Time, limit int) ([]int64, error) {
	q := `SELECT b.id FROM books b WHERE b.monitored = 1 AND b.release_date <= ? AND b.last_search_at <= ?
		AND NOT EXISTS (SELECT 1 FROM book_files f WHERE f.book_id = b.id)
		AND NOT EXISTS (SELECT 1 FROM downloads d WHERE d.book_id = b.id AND d.status != 'failed')`
	args := []any{today(), notSince.Unix()}
	if authorID != 0 {
		q += ` AND b.author_id = ?`
		args = append(args, authorID)
	}
	q += ` ORDER BY b.last_search_at, b.id`
	if limit > 0 {
		q += ` LIMIT ` + strconv.Itoa(limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) DeleteBook(id int64) error {
	_, err := s.db.Exec(`DELETE FROM books WHERE id = ?`, id)
	if err == nil {
		err = s.pruneSeries()
	}
	return err
}

// ---------------------------------------------------------------- files

func (s *Store) BookFiles(bookID int64) ([]model.BookFile, error) {
	rows, err := s.db.Query(`SELECT id, path, size_bytes, format FROM book_files WHERE book_id = ? ORDER BY path`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.BookFile{}
	for rows.Next() {
		var f model.BookFile
		if err := rows.Scan(&f.ID, &f.Path, &f.SizeBytes, &f.Format); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

type FileRecord struct {
	model.BookFile
	BookID int64
}

func (s *Store) AllFiles() ([]FileRecord, error) {
	rows, err := s.db.Query(`SELECT id, book_id, path, size_bytes, format FROM book_files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileRecord
	for rows.Next() {
		var f FileRecord
		if err := rows.Scan(&f.ID, &f.BookID, &f.Path, &f.SizeBytes, &f.Format); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// SetBookFiles replaces the book's file list and refreshes its path, size and
// primary format. dir is the folder holding the files ("" when files is empty).
func (s *Store) SetBookFiles(bookID int64, dir string, files []model.BookFile) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM book_files WHERE book_id = ?`, bookID); err != nil {
		return err
	}
	var total int64
	sizeByFormat := map[string]int64{}
	for _, f := range files {
		// A path can only belong to one book; the newest claim wins.
		if _, err := tx.Exec(`DELETE FROM book_files WHERE path = ?`, f.Path); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO book_files (book_id, path, size_bytes, format) VALUES (?,?,?,?)`,
			bookID, f.Path, f.SizeBytes, f.Format); err != nil {
			return err
		}
		total += f.SizeBytes
		sizeByFormat[f.Format] += f.SizeBytes
	}
	format := ""
	var best int64 = -1
	for f, sz := range sizeByFormat {
		if sz > best || (sz == best && f < format) {
			format, best = f, sz
		}
	}
	if _, err := tx.Exec(`UPDATE books SET path = ?, size_bytes = ?, format = ? WHERE id = ?`,
		dir, total, format, bookID); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------------------------------------------------------------- counts

type Counts struct {
	Authors, Books, Downloaded, Missing, Queue int
	LibrarySize                                int64
}

func (s *Store) Counts() (Counts, error) {
	var c Counts
	err := s.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM authors),
		(SELECT COUNT(*) FROM books),
		(SELECT COUNT(*) FROM books b WHERE EXISTS (SELECT 1 FROM book_files f WHERE f.book_id = b.id)),
		(SELECT COUNT(*) FROM books b WHERE b.monitored = 1 AND b.release_date <= ?
			AND NOT EXISTS (SELECT 1 FROM book_files f WHERE f.book_id = b.id)
			AND NOT EXISTS (SELECT 1 FROM downloads d WHERE d.book_id = b.id AND d.status != 'failed')),
		(SELECT COUNT(*) FROM downloads),
		(SELECT COALESCE(SUM(size_bytes), 0) FROM book_files)`, today()).
		Scan(&c.Authors, &c.Books, &c.Downloaded, &c.Missing, &c.Queue, &c.LibrarySize)
	return c, err
}
