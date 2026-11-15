package download

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ncecere/chaptarr/internal/model"
)

// qbittorrent drives the qBittorrent Web API (v2).
type qbittorrent struct {
	base, username, password, category string
	http                               *http.Client
	mu                                 sync.Mutex
	loggedIn                           bool
}

func newQBittorrent(base string, c model.DownloadClient) *qbittorrent {
	jar, _ := cookiejar.New(nil)
	return &qbittorrent{
		base: base, username: c.Username, password: c.Password, category: c.Category,
		http: &http.Client{Timeout: 30 * time.Second, Jar: jar},
	}
}

func (q *qbittorrent) login(ctx context.Context) error {
	form := url.Values{"username": {q.username}, "password": {q.password}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, q.base+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", q.base)
	res, err := q.http.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach qBittorrent: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
	switch {
	case res.StatusCode == http.StatusForbidden:
		return errors.New("qBittorrent has temporarily banned this address after too many failed logins")
	case res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent:
		return fmt.Errorf("qBittorrent login returned %s", res.Status)
	case strings.Contains(string(body), "Fails"):
		return errors.New("qBittorrent rejected the username or password")
	}
	q.loggedIn = true
	return nil
}

// do sends a request, logging in first and once more if the session expired.
func (q *qbittorrent) do(ctx context.Context, method, path string, contentType string, body []byte) ([]byte, int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for attempt := 0; ; attempt++ {
		if !q.loggedIn {
			if err := q.login(ctx); err != nil {
				return nil, 0, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, method, q.base+path, bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("Referer", q.base)
		res, err := q.http.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("could not reach qBittorrent: %w", err)
		}
		data, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
		res.Body.Close()
		if err != nil {
			return nil, 0, err
		}
		if res.StatusCode == http.StatusForbidden && attempt == 0 {
			q.loggedIn = false
			continue
		}
		return data, res.StatusCode, nil
	}
}

func (q *qbittorrent) form(ctx context.Context, path string, v url.Values) ([]byte, int, error) {
	return q.do(ctx, http.MethodPost, path, "application/x-www-form-urlencoded", []byte(v.Encode()))
}

func (q *qbittorrent) Test(ctx context.Context) error {
	data, code, err := q.do(ctx, http.MethodGet, "/api/v2/app/version", "", nil)
	if err != nil {
		return err
	}
	if code != http.StatusOK || len(data) == 0 {
		return fmt.Errorf("qBittorrent returned HTTP %d for the version check", code)
	}
	return nil
}

func (q *qbittorrent) Add(ctx context.Context, p Payload) (string, error) {
	if p.InfoHash == "" {
		return "", errors.New("release has no info-hash")
	}
	if q.category != "" {
		// Fails harmlessly with 409 when the category already exists.
		_, _, _ = q.form(ctx, "/api/v2/torrents/createCategory", url.Values{"category": {q.category}})
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if p.Magnet != "" {
		_ = w.WriteField("urls", p.Magnet)
	} else {
		fw, err := w.CreateFormFile("torrents", p.Filename)
		if err != nil {
			return "", err
		}
		_, _ = fw.Write(p.File)
	}
	if q.category != "" {
		_ = w.WriteField("category", q.category)
	}
	w.Close()
	data, code, err := q.do(ctx, http.MethodPost, "/api/v2/torrents/add", w.FormDataContentType(), buf.Bytes())
	if err != nil {
		return "", err
	}
	if code != http.StatusOK || strings.Contains(string(data), "Fails") {
		// "Fails." is also what qBittorrent says when the torrent is already
		// there, in which case tracking the existing one is what we want.
		if _, err := q.Status(ctx, p.InfoHash); err == nil {
			return p.InfoHash, nil
		}
		return "", fmt.Errorf("qBittorrent refused the torrent (HTTP %d: %s)", code, strings.TrimSpace(string(data)))
	}
	return p.InfoHash, nil
}

func (q *qbittorrent) Status(ctx context.Context, id string) (*Item, error) {
	data, code, err := q.do(ctx, http.MethodGet, "/api/v2/torrents/info?hashes="+url.QueryEscape(id), "", nil)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("qBittorrent returned HTTP %d", code)
	}
	var torrents []struct {
		State       string  `json:"state"`
		Progress    float64 `json:"progress"`
		Size        int64   `json:"size"`
		ETA         int64   `json:"eta"`
		ContentPath string  `json:"content_path"`
		SavePath    string  `json:"save_path"`
		Name        string  `json:"name"`
	}
	if err := json.Unmarshal(data, &torrents); err != nil {
		return nil, fmt.Errorf("unexpected response from qBittorrent: %w", err)
	}
	if len(torrents) == 0 {
		return nil, ErrGone
	}
	t := torrents[0]
	it := &Item{Progress: t.Progress, SizeBytes: t.Size, Path: t.ContentPath}
	if it.Path == "" && t.SavePath != "" {
		it.Path = strings.TrimRight(t.SavePath, "/") + "/" + t.Name
	}
	// 8640000 is qBittorrent's "infinity".
	if t.ETA > 0 && t.ETA < 8640000 {
		it.ETASeconds = &t.ETA
	}
	switch t.State {
	case "error":
		it.State, it.Error = StateFailed, "qBittorrent reports an error for this torrent"
	case "missingFiles":
		it.State, it.Error = StateFailed, "qBittorrent reports the torrent's files are missing"
	case "pausedDL", "stoppedDL":
		it.State = StatePaused
	case "queuedDL", "metaDL", "forcedMetaDL", "allocating", "checkingResumeData":
		it.State = StateQueued
	case "uploading", "forcedUP", "stalledUP", "queuedUP", "pausedUP", "stoppedUP":
		it.State = StateCompleted
	default:
		// downloading, stalledDL, forcedDL, checkingDL/UP, moving: not safe
		// to import yet even if progress reads 1.
		it.State = StateDownloading
	}
	return it, nil
}

func (q *qbittorrent) Remove(ctx context.Context, id string, deleteData bool) error {
	del := "false"
	if deleteData {
		del = "true"
	}
	_, code, err := q.form(ctx, "/api/v2/torrents/delete", url.Values{"hashes": {id}, "deleteFiles": {del}})
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("qBittorrent returned HTTP %d", code)
	}
	return nil
}
