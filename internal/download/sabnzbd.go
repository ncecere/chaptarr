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
	"net/url"
	"strconv"
	"strings"
)

type sabnzbd struct {
	base, apiKey, category string
	http                   *http.Client
}

func (s *sabnzbd) call(ctx context.Context, params url.Values, contentType string, body io.Reader, out any) error {
	params.Set("apikey", s.apiKey)
	params.Set("output", "json")
	method := http.MethodGet
	if body != nil {
		method = http.MethodPost
	}
	req, err := http.NewRequestWithContext(ctx, method, s.base+"/api?"+params.Encode(), body)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := s.http.Do(req)
	if err != nil {
		if ue, ok := err.(*url.Error); ok {
			err = ue.Err // the URL carries the API key
		}
		return fmt.Errorf("could not reach SABnzbd: %w", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("SABnzbd returned %s", res.Status)
	}
	var fail struct {
		Status *bool  `json:"status"`
		Error  string `json:"error"`
	}
	if json.Unmarshal(data, &fail) == nil && fail.Error != "" {
		return fmt.Errorf("SABnzbd: %s", fail.Error)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("unexpected response from SABnzbd: %w", err)
		}
	}
	return nil
}

func (s *sabnzbd) Test(ctx context.Context) error {
	// "queue" needs a valid API key, unlike "version".
	var out struct {
		Queue *struct{} `json:"queue"`
	}
	if err := s.call(ctx, url.Values{"mode": {"queue"}, "limit": {"1"}}, "", nil, &out); err != nil {
		return err
	}
	if out.Queue == nil {
		return errors.New("SABnzbd did not return its queue; check the URL")
	}
	return nil
}

func (s *sabnzbd) Add(ctx context.Context, p Payload) (string, error) {
	if len(p.File) == 0 {
		return "", errors.New("release has no NZB file")
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("name", p.Filename)
	if err != nil {
		return "", err
	}
	_, _ = fw.Write(p.File)
	w.Close()
	params := url.Values{"mode": {"addfile"}, "nzbname": {p.Title}}
	if s.category != "" {
		params.Set("cat", s.category)
	}
	var out struct {
		Status bool     `json:"status"`
		IDs    []string `json:"nzo_ids"`
	}
	if err := s.call(ctx, params, w.FormDataContentType(), &buf, &out); err != nil {
		return "", err
	}
	if !out.Status || len(out.IDs) == 0 {
		return "", errors.New("SABnzbd did not accept the NZB")
	}
	return out.IDs[0], nil
}

func (s *sabnzbd) Status(ctx context.Context, id string) (*Item, error) {
	var q struct {
		Queue struct {
			Slots []struct {
				ID         string `json:"nzo_id"`
				Status     string `json:"status"`
				Percentage string `json:"percentage"`
				MB         string `json:"mb"`
				TimeLeft   string `json:"timeleft"`
			} `json:"slots"`
		} `json:"queue"`
	}
	if err := s.call(ctx, url.Values{"mode": {"queue"}, "nzo_ids": {id}}, "", nil, &q); err != nil {
		return nil, err
	}
	for _, sl := range q.Queue.Slots {
		if sl.ID != id {
			continue
		}
		it := &Item{State: StateDownloading}
		if pct, err := strconv.ParseFloat(sl.Percentage, 64); err == nil {
			it.Progress = pct / 100
		}
		if mb, err := strconv.ParseFloat(sl.MB, 64); err == nil {
			it.SizeBytes = int64(mb * 1024 * 1024)
		}
		if eta, ok := parseTimeLeft(sl.TimeLeft); ok && eta > 0 {
			it.ETASeconds = &eta
		}
		switch sl.Status {
		case "Paused":
			it.State = StatePaused
		case "Queued", "Grabbing", "Fetching", "Propagating":
			it.State = StateQueued
		}
		return it, nil
	}

	var h struct {
		History struct {
			Slots []struct {
				ID          string `json:"nzo_id"`
				Status      string `json:"status"`
				Storage     string `json:"storage"`
				Bytes       int64  `json:"bytes"`
				FailMessage string `json:"fail_message"`
			} `json:"slots"`
		} `json:"history"`
	}
	if err := s.call(ctx, url.Values{"mode": {"history"}, "nzo_ids": {id}}, "", nil, &h); err != nil {
		return nil, err
	}
	for _, sl := range h.History.Slots {
		if sl.ID != id {
			continue
		}
		it := &Item{SizeBytes: sl.Bytes, Path: sl.Storage}
		switch sl.Status {
		case "Completed":
			it.State, it.Progress = StateCompleted, 1
		case "Failed":
			it.State, it.Error = StateFailed, sl.FailMessage
			if it.Error == "" {
				it.Error = "SABnzbd reports the download failed"
			}
		default:
			// Verifying, Repairing, Extracting, Moving, Running: post-processing.
			it.State, it.Progress = StateDownloading, 1
		}
		return it, nil
	}
	return nil, ErrGone
}

// parseTimeLeft reads SABnzbd's "H:MM:SS" or "D:HH:MM:SS".
func parseTimeLeft(s string) (int64, bool) {
	parts := strings.Split(s, ":")
	if len(parts) < 3 || len(parts) > 4 {
		return 0, false
	}
	var total int64
	mult := []int64{86400, 3600, 60, 1}[4-len(parts):]
	for i, p := range parts {
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return 0, false
		}
		total += n * mult[i]
	}
	return total, true
}

func (s *sabnzbd) Remove(ctx context.Context, id string, deleteData bool) error {
	del := "0"
	if deleteData {
		del = "1"
	}
	// The job is in either the queue or the history; deleting from the one
	// that does not have it is a no-op.
	errQ := s.call(ctx, url.Values{"mode": {"queue"}, "name": {"delete"}, "value": {id}, "del_files": {del}}, "", nil, nil)
	errH := s.call(ctx, url.Values{"mode": {"history"}, "name": {"delete"}, "value": {id}, "del_files": {del}}, "", nil, nil)
	if errQ != nil {
		return errQ
	}
	return errH
}
