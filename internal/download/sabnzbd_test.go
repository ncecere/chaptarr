package download

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ncecere/chaptarr/internal/model"
)

func TestSABnzbd(t *testing.T) {
	stage := "queue"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("apikey") != "key" {
			json.NewEncoder(w).Encode(map[string]any{"status": false, "error": "API Key Incorrect"})
			return
		}
		switch q.Get("mode") {
		case "addfile":
			f, hdr, err := r.FormFile("name")
			if err != nil {
				t.Errorf("no upload: %v", err)
				return
			}
			data, _ := io.ReadAll(f)
			if !strings.Contains(string(data), "<nzb") || hdr.Filename != "My Book.nzb" || q.Get("cat") != "audiobooks" {
				t.Errorf("unexpected upload %q cat=%q", hdr.Filename, q.Get("cat"))
			}
			json.NewEncoder(w).Encode(map[string]any{"status": true, "nzo_ids": []string{"SABnzbd_nzo_1"}})
		case "queue":
			slots := []map[string]string{}
			if stage == "queue" {
				slots = append(slots, map[string]string{"nzo_id": "SABnzbd_nzo_1", "status": "Downloading", "percentage": "25", "mb": "100.0", "timeleft": "0:01:30"})
			}
			json.NewEncoder(w).Encode(map[string]any{"queue": map[string]any{"slots": slots}})
		case "history":
			slots := []map[string]any{}
			switch stage {
			case "unpacking":
				slots = append(slots, map[string]any{"nzo_id": "SABnzbd_nzo_1", "status": "Extracting"})
			case "done":
				slots = append(slots, map[string]any{"nzo_id": "SABnzbd_nzo_1", "status": "Completed", "storage": "/downloads/My Book", "bytes": 1234})
			case "failed":
				slots = append(slots, map[string]any{"nzo_id": "SABnzbd_nzo_1", "status": "Failed", "fail_message": "Unpacking failed"})
			}
			json.NewEncoder(w).Encode(map[string]any{"history": map[string]any{"slots": slots}})
		}
	}))
	defer srv.Close()
	ctx := context.Background()

	bad, _ := New(model.DownloadClient{Type: "sabnzbd", URL: srv.URL, APIKey: "wrong"})
	if err := bad.Test(ctx); err == nil || !strings.Contains(err.Error(), "API Key Incorrect") {
		t.Fatalf("wrong key: %v", err)
	}
	c, err := New(model.DownloadClient{Type: "sabnzbd", URL: srv.URL + "/", APIKey: "key", Category: "audiobooks"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Test(ctx); err != nil {
		t.Fatal(err)
	}
	id, err := c.Add(ctx, Payload{Title: "My Book", Filename: "My Book.nzb", File: []byte(`<nzb xmlns="x"/>`)})
	if err != nil || id != "SABnzbd_nzo_1" {
		t.Fatalf("add: %q %v", id, err)
	}

	it, err := c.Status(ctx, id)
	if err != nil || it.State != StateDownloading || it.Progress != 0.25 || it.ETASeconds == nil || *it.ETASeconds != 90 {
		t.Fatalf("downloading: %+v %v", it, err)
	}
	stage = "unpacking"
	if it, _ = c.Status(ctx, id); it.State != StateDownloading {
		t.Fatalf("post-processing must not count as complete: %+v", it)
	}
	stage = "done"
	if it, _ = c.Status(ctx, id); it.State != StateCompleted || it.Path != "/downloads/My Book" {
		t.Fatalf("completed: %+v", it)
	}
	stage = "failed"
	if it, _ = c.Status(ctx, id); it.State != StateFailed || it.Error != "Unpacking failed" {
		t.Fatalf("failed: %+v", it)
	}
	stage = "gone"
	if _, err := c.Status(ctx, id); !errors.Is(err, ErrGone) {
		t.Fatalf("gone: %v", err)
	}
}
