// Package api is the HTTP layer: JSON routes under /api/v1 (the contract is
// web/src/lib/api.ts) and the embedded single-page UI everywhere else.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ncecere/chaptarr/internal/logbuf"
	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/service"
	"github.com/ncecere/chaptarr/internal/store"
)

type Server struct {
	svc   *service.Service
	store *store.Store
	log   *slog.Logger
	logs  *logbuf.Buffer
	ui    fs.FS

	setupMu sync.Mutex
	limiter loginLimiter
}

func New(svc *service.Service, logs *logbuf.Buffer, ui fs.FS) *Server {
	return &Server{svc: svc, store: svc.Store, log: svc.Log, logs: logs, ui: ui}
}

type ctxKey int

const (
	userKey ctxKey = iota
	sessionKey
)

func currentUser(r *http.Request) *store.UserRecord {
	u, _ := r.Context().Value(userKey).(*store.UserRecord)
	return u
}

// handler is a route that returns a value to encode as JSON, or an error.
// A nil value with a nil error is 204 No Content.
type handler func(w http.ResponseWriter, r *http.Request) (any, error)

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// public routes need no session; everything else does.
	public := func(pattern string, h handler) { mux.Handle(pattern, s.wrap(h, false)) }
	authed := func(pattern string, h handler) { mux.Handle(pattern, s.wrap(h, true)) }

	public("GET /api/v1/auth/status", s.authStatus)
	public("POST /api/v1/auth/setup", s.setup)
	public("POST /api/v1/auth/login", s.login)
	public("POST /api/v1/auth/logout", s.logout)
	authed("POST /api/v1/auth/password", s.changePassword)

	authed("GET /api/v1/search/books", s.searchBooks)
	authed("GET /api/v1/search/authors", s.searchAuthors)

	authed("GET /api/v1/authors", s.listAuthors)
	authed("POST /api/v1/authors", s.addAuthor)
	authed("GET /api/v1/authors/{id}", s.getAuthor)
	authed("PUT /api/v1/authors/{id}", s.updateAuthor)
	authed("DELETE /api/v1/authors/{id}", s.deleteAuthor)
	authed("POST /api/v1/authors/{id}/refresh", s.refreshAuthor)
	authed("POST /api/v1/authors/{id}/search", s.searchAuthor)

	authed("GET /api/v1/books", s.listBooks)
	authed("POST /api/v1/books", s.addBook)
	authed("PUT /api/v1/books/monitor", s.monitorBooks)
	authed("GET /api/v1/books/{id}", s.getBook)
	authed("PUT /api/v1/books/{id}", s.updateBook)
	authed("DELETE /api/v1/books/{id}", s.deleteBook)
	authed("GET /api/v1/books/{id}/releases", s.bookReleases)
	authed("POST /api/v1/books/{id}/grab", s.grabRelease)
	authed("POST /api/v1/books/{id}/search", s.searchBook)

	authed("GET /api/v1/series", s.listSeries)
	authed("GET /api/v1/series/{id}", s.getSeries)
	authed("GET /api/v1/series/{id}/missing", s.seriesMissing)
	authed("POST /api/v1/series/{id}/books", s.addSeriesBooks)

	authed("GET /api/v1/queue", s.listQueue)
	authed("DELETE /api/v1/queue/{id}", s.removeQueueItem)
	authed("POST /api/v1/queue/{id}/import", s.retryImport)
	authed("GET /api/v1/history", s.listHistory)

	authed("GET /api/v1/rootfolders", s.listRootFolders)
	authed("POST /api/v1/rootfolders", s.addRootFolder)
	authed("DELETE /api/v1/rootfolders/{id}", s.deleteRootFolder)
	authed("GET /api/v1/filesystem", s.browse)

	authed("GET /api/v1/indexers", s.listIndexers)
	authed("POST /api/v1/indexers", s.saveIndexer)
	authed("POST /api/v1/indexers/test", s.testIndexer)
	authed("PUT /api/v1/indexers/{id}", s.saveIndexer)
	authed("DELETE /api/v1/indexers/{id}", s.deleteIndexer)

	authed("GET /api/v1/downloadclients", s.listDownloadClients)
	authed("POST /api/v1/downloadclients", s.saveDownloadClient)
	authed("POST /api/v1/downloadclients/test", s.testDownloadClient)
	authed("PUT /api/v1/downloadclients/{id}", s.saveDownloadClient)
	authed("DELETE /api/v1/downloadclients/{id}", s.deleteDownloadClient)

	authed("GET /api/v1/settings", s.getSettings)
	authed("PUT /api/v1/settings", s.putSettings)
	authed("POST /api/v1/settings/metadata/test", s.testMetadataSource)

	authed("GET /api/v1/system/status", s.status)
	authed("GET /api/v1/system/tasks", s.listTasks)
	authed("POST /api/v1/system/tasks/{name}/run", s.runTask)
	authed("GET /api/v1/system/logs", s.listLogs)
	authed("POST /api/v1/system/apikey", s.regenerateAPIKey)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "No such API route")
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.Handle("/", s.uiHandler())
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) wrap(h handler, needAuth bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The session cookie is SameSite=Lax, which already keeps it off
		// cross-site POSTs; refusing foreign Origins covers older browsers.
		if r.Method != http.MethodGet && !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "Cross-origin request refused")
			return
		}
		user, token := s.authenticate(r)
		if needAuth && user == nil {
			writeError(w, http.StatusUnauthorized, "Sign in required")
			return
		}
		ctx := context.WithValue(r.Context(), userKey, user)
		ctx = context.WithValue(ctx, sessionKey, token)
		r = r.WithContext(ctx)
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		v, err := h(w, r)
		if err != nil {
			var se *service.Error
			switch {
			case errors.As(err, &se):
				writeError(w, se.Status, se.Msg)
			case errors.Is(err, store.ErrNotFound):
				writeError(w, http.StatusNotFound, "Not found")
			case r.Context().Err() != nil:
				// The client went away; nothing useful to send.
			default:
				s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
				writeError(w, http.StatusInternalServerError, "Internal error. Check the log for details.")
			}
			return
		}
		if v == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, http.StatusOK, v)
	})
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // not a browser cross-origin request
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = fwd
	}
	origin = strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
	return strings.EqualFold(origin, host) || strings.EqualFold(origin, r.Host)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func badRequest(msg string) error { return &service.Error{Status: http.StatusBadRequest, Msg: msg} }

// decode reads a JSON body into v.
func decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return badRequest("A JSON body is required")
		}
		return badRequest("Invalid JSON body")
	}
	return nil
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, service.ErrNotFound
	}
	return id, nil
}

func queryBool(r *http.Request, name string) bool {
	v, _ := strconv.ParseBool(r.URL.Query().Get(name))
	return v
}

// ---------------------------------------------------------------- UI

func (s *Server) uiHandler() http.Handler {
	files := http.FileServerFS(s.ui)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" {
			if info, err := fs.Stat(s.ui, name); err == nil && !info.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					// Vite fingerprints everything under assets/.
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
			// A missing file with an extension is a real 404; anything else
			// is a client-side route.
			if path.Ext(name) != "" {
				http.NotFound(w, r)
				return
			}
		}
		index, err := fs.ReadFile(s.ui, "index.html")
		if err != nil {
			http.Error(w, "The web UI was not built into this binary. Run `make build`.", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, "index.html", time.Time{}, strings.NewReader(string(index)))
	})
}

// ---------------------------------------------------------------- misc handlers that need no service logic

func (s *Server) listLogs(w http.ResponseWriter, r *http.Request) (any, error) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 200
	}
	entries := s.logs.Recent(limit)
	if entries == nil {
		entries = []model.LogEntry{}
	}
	return entries, nil
}
