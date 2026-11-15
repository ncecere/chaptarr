package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/ncecere/chaptarr/internal/model"
)

type UserRecord struct {
	model.User
	PasswordHash string
	APIKey       string
}

func (s *Store) UserCount() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(username, passwordHash, apiKey string) (*UserRecord, error) {
	res, err := s.db.Exec(`INSERT INTO users (username, password_hash, api_key) VALUES (?,?,?)`,
		username, passwordHash, apiKey)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &UserRecord{User: model.User{ID: id, Username: username}, PasswordHash: passwordHash, APIKey: apiKey}, nil
}

func (s *Store) scanUser(row *sql.Row) (*UserRecord, error) {
	var u UserRecord
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.APIKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

const userCols = `id, username, password_hash, api_key`

func (s *Store) UserByName(username string) (*UserRecord, error) {
	return s.scanUser(s.db.QueryRow(`SELECT `+userCols+` FROM users WHERE username = ?`, username))
}

func (s *Store) UserByID(id int64) (*UserRecord, error) {
	return s.scanUser(s.db.QueryRow(`SELECT `+userCols+` FROM users WHERE id = ?`, id))
}

func (s *Store) UserByAPIKey(key string) (*UserRecord, error) {
	return s.scanUser(s.db.QueryRow(`SELECT `+userCols+` FROM users WHERE api_key = ?`, key))
}

func (s *Store) UserBySession(token string) (*UserRecord, error) {
	return s.scanUser(s.db.QueryRow(`SELECT u.id, u.username, u.password_hash, u.api_key
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token = ? AND s.expires_at > ?`, token, time.Now().Unix()))
}

func (s *Store) SetPassword(userID int64, hash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, userID)
	return err
}

func (s *Store) SetAPIKey(userID int64, key string) error {
	_, err := s.db.Exec(`UPDATE users SET api_key = ? WHERE id = ?`, key, userID)
	return err
}

func (s *Store) CreateSession(token string, userID int64, expires time.Time) error {
	_, err := s.db.Exec(`INSERT INTO sessions (token, user_id, expires_at) VALUES (?,?,?)`,
		token, userID, expires.Unix())
	return err
}

func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

// DeleteUserSessions signs the user out everywhere except the session in keep.
func (s *Store) DeleteUserSessions(userID int64, keep string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ? AND token != ?`, userID, keep)
	return err
}

func (s *Store) PurgeExpiredSessions() error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, time.Now().Unix())
	return err
}
