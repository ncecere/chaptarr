package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/ncecere/chaptarr/internal/model"
)

// ---------------------------------------------------------------- settings

func (s *Store) Settings() (model.Settings, error) {
	set := model.DefaultSettings()
	var data string
	err := s.db.QueryRow(`SELECT data FROM settings WHERE id = 1`).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return set, nil
	}
	if err != nil {
		return set, err
	}
	// Unmarshal over the defaults so settings added in later versions keep
	// their default until the user saves.
	err = json.Unmarshal([]byte(data), &set)
	set.MetadataSources = model.NormalizeSources(set.MetadataSources)
	return set, err
}

func (s *Store) SaveSettings(set model.Settings) error {
	data, err := json.Marshal(set)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO settings (id, data) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET data = excluded.data`, string(data))
	return err
}

// ---------------------------------------------------------------- root folders

func (s *Store) RootFolders() ([]model.RootFolder, error) {
	rows, err := s.db.Query(`SELECT id, path FROM root_folders ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.RootFolder{}
	for rows.Next() {
		var f model.RootFolder
		if err := rows.Scan(&f.ID, &f.Path); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) AddRootFolder(path string) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO root_folders (path) VALUES (?)`, path)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteRootFolder(id int64) error {
	_, err := s.db.Exec(`DELETE FROM root_folders WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------- indexers

func (s *Store) Indexers() ([]model.Indexer, error) {
	rows, err := s.db.Query(`SELECT id, name, type, url, api_key, categories, enabled, priority
		FROM indexers ORDER BY priority, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Indexer{}
	for rows.Next() {
		var i model.Indexer
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.URL, &i.APIKey, &i.Categories, &i.Enabled, &i.Priority); err != nil {
			return nil, err
		}
		i.HasAPIKey = i.APIKey != ""
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) Indexer(id int64) (*model.Indexer, error) {
	all, err := s.Indexers()
	if err != nil {
		return nil, err
	}
	for _, i := range all {
		if i.ID == id {
			return &i, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) SaveIndexer(i *model.Indexer) error {
	if i.ID == 0 {
		res, err := s.db.Exec(`INSERT INTO indexers (name, type, url, api_key, categories, enabled, priority)
			VALUES (?,?,?,?,?,?,?)`, i.Name, i.Type, i.URL, i.APIKey, i.Categories, b2i(i.Enabled), i.Priority)
		if err != nil {
			return err
		}
		i.ID, _ = res.LastInsertId()
		return nil
	}
	_, err := s.db.Exec(`UPDATE indexers SET name = ?, type = ?, url = ?, api_key = ?, categories = ?,
		enabled = ?, priority = ? WHERE id = ?`, i.Name, i.Type, i.URL, i.APIKey, i.Categories, b2i(i.Enabled), i.Priority, i.ID)
	return err
}

func (s *Store) DeleteIndexer(id int64) error {
	_, err := s.db.Exec(`DELETE FROM indexers WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------- download clients

func (s *Store) DownloadClients() ([]model.DownloadClient, error) {
	rows, err := s.db.Query(`SELECT id, name, type, url, username, password, api_key, category, enabled
		FROM download_clients ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.DownloadClient{}
	for rows.Next() {
		var c model.DownloadClient
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.URL, &c.Username, &c.Password, &c.APIKey, &c.Category, &c.Enabled); err != nil {
			return nil, err
		}
		c.HasPassword = c.Password != ""
		c.HasAPIKey = c.APIKey != ""
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) DownloadClient(id int64) (*model.DownloadClient, error) {
	all, err := s.DownloadClients()
	if err != nil {
		return nil, err
	}
	for _, c := range all {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) SaveDownloadClient(c *model.DownloadClient) error {
	if c.ID == 0 {
		res, err := s.db.Exec(`INSERT INTO download_clients (name, type, url, username, password, api_key, category, enabled)
			VALUES (?,?,?,?,?,?,?,?)`, c.Name, c.Type, c.URL, c.Username, c.Password, c.APIKey, c.Category, b2i(c.Enabled))
		if err != nil {
			return err
		}
		c.ID, _ = res.LastInsertId()
		return nil
	}
	_, err := s.db.Exec(`UPDATE download_clients SET name = ?, type = ?, url = ?, username = ?, password = ?,
		api_key = ?, category = ?, enabled = ? WHERE id = ?`,
		c.Name, c.Type, c.URL, c.Username, c.Password, c.APIKey, c.Category, b2i(c.Enabled), c.ID)
	return err
}

func (s *Store) DeleteDownloadClient(id int64) error {
	_, err := s.db.Exec(`DELETE FROM download_clients WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------- queue

const queueSelect = `SELECT d.id, d.book_id, b.title, a.name, b.cover_url, d.release_title, d.indexer, d.protocol,
	d.client_name, d.status, d.progress, d.size_bytes, d.eta_seconds, d.error, d.added_at,
	d.client_id, d.client_item_id, d.guid
	FROM downloads d JOIN books b ON b.id = d.book_id JOIN authors a ON a.id = b.author_id`

func scanQueue(row scanner) (*model.QueueItem, error) {
	var q model.QueueItem
	var eta sql.NullInt64
	var added int64
	err := row.Scan(&q.ID, &q.BookID, &q.BookTitle, &q.AuthorName, &q.CoverURL, &q.ReleaseTitle, &q.Indexer,
		&q.Protocol, &q.DownloadClient, &q.Status, &q.Progress, &q.SizeBytes, &eta, &q.Error, &added,
		&q.ClientID, &q.ClientItemID, &q.GUID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if eta.Valid {
		q.ETASeconds = &eta.Int64
	}
	q.AddedAt = fromUnix(added)
	return &q, nil
}

func (s *Store) Queue() ([]model.QueueItem, error) {
	rows, err := s.db.Query(queueSelect + ` ORDER BY d.added_at DESC, d.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.QueueItem{}
	for rows.Next() {
		q, err := scanQueue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *q)
	}
	return out, rows.Err()
}

func (s *Store) QueueItem(id int64) (*model.QueueItem, error) {
	return scanQueue(s.db.QueryRow(queueSelect+` WHERE d.id = ?`, id))
}

func (s *Store) AddQueueItem(q *model.QueueItem) error {
	q.AddedAt = time.Now().UTC()
	res, err := s.db.Exec(`INSERT INTO downloads (book_id, release_title, guid, indexer, protocol, client_id,
		client_name, client_item_id, status, size_bytes, added_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		q.BookID, q.ReleaseTitle, q.GUID, q.Indexer, q.Protocol, q.ClientID, q.DownloadClient, q.ClientItemID,
		q.Status, q.SizeBytes, unix(q.AddedAt))
	if err != nil {
		return err
	}
	q.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) UpdateQueueItem(id int64, status string, progress float64, size int64, eta *int64, errMsg string) error {
	_, err := s.db.Exec(`UPDATE downloads SET status = ?, progress = ?, size_bytes = ?, eta_seconds = ?, error = ?
		WHERE id = ?`, status, progress, size, eta, errMsg, id)
	return err
}

func (s *Store) DeleteQueueItem(id int64) error {
	_, err := s.db.Exec(`DELETE FROM downloads WHERE id = ?`, id)
	return err
}

// ---------------------------------------------------------------- history

func (s *Store) AddHistory(b *model.Book, typ, sourceTitle, guid, message string) error {
	_, err := s.db.Exec(`INSERT INTO history (book_id, book_title, author_name, type, source_title, guid, message, date)
		VALUES (?,?,?,?,?,?,?,?)`, b.ID, b.Title, b.AuthorName, typ, sourceTitle, guid, message, time.Now().Unix())
	return err
}

func (s *Store) history(where string, limit, offset int, args ...any) ([]model.HistoryEvent, error) {
	args = append(args, limit, offset)
	rows, err := s.db.Query(`SELECT id, book_id, book_title, author_name, type, source_title, message, date
		FROM history `+where+` ORDER BY date DESC, id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.HistoryEvent{}
	for rows.Next() {
		var h model.HistoryEvent
		var date int64
		if err := rows.Scan(&h.ID, &h.BookID, &h.BookTitle, &h.AuthorName, &h.Type, &h.SourceTitle, &h.Message, &date); err != nil {
			return nil, err
		}
		h.Date = fromUnix(date)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) History(page, pageSize int) (*model.HistoryPage, error) {
	items, err := s.history(``, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	p := &model.HistoryPage{Items: items, Page: page, PageSize: pageSize}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM history`).Scan(&p.Total)
	return p, err
}

func (s *Store) BookHistory(bookID int64) ([]model.HistoryEvent, error) {
	return s.history(`WHERE book_id = ?`, 100, 0, bookID)
}

// FailedGUIDs returns the releases that already failed for a book, so
// automatic search does not grab them again.
func (s *Store) FailedGUIDs(bookID int64) (map[string]bool, error) {
	rows, err := s.db.Query(`SELECT guid FROM history WHERE book_id = ? AND guid != ''
		AND type IN ('downloadFailed', 'importFailed')`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		out[g] = true
	}
	return out, rows.Err()
}
