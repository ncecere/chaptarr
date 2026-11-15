package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/service"
)

// ---------------------------------------------------------------- metadata search

func searchQuery(r *http.Request) (string, error) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		return "", badRequest("Enter at least two characters to search")
	}
	return q, nil
}

func (s *Server) searchBooks(w http.ResponseWriter, r *http.Request) (any, error) {
	q, err := searchQuery(r)
	if err != nil {
		return nil, err
	}
	return s.svc.SearchBooks(r.Context(), q)
}

func (s *Server) searchAuthors(w http.ResponseWriter, r *http.Request) (any, error) {
	q, err := searchQuery(r)
	if err != nil {
		return nil, err
	}
	return s.svc.SearchAuthors(r.Context(), q)
}

// ---------------------------------------------------------------- authors

func (s *Server) listAuthors(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.store.Authors()
}

func (s *Server) getAuthor(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return s.svc.AuthorDetail(id)
}

func (s *Server) addAuthor(w http.ResponseWriter, r *http.Request) (any, error) {
	var req service.AddAuthorRequest
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	return s.svc.AddAuthor(r.Context(), req)
}

func (s *Server) updateAuthor(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	var patch struct {
		Monitored       *bool `json:"monitored"`
		MonitorNewBooks *bool `json:"monitorNewBooks"`
	}
	if err := decode(r, &patch); err != nil {
		return nil, err
	}
	if _, err := s.store.Author(id); err != nil {
		return nil, err
	}
	if err := s.store.SetAuthorFlags(id, patch.Monitored, patch.MonitorNewBooks); err != nil {
		return nil, err
	}
	return s.store.Author(id)
}

func (s *Server) deleteAuthor(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return nil, s.svc.DeleteAuthor(id, queryBool(r, "deleteFiles"))
}

func (s *Server) refreshAuthor(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return s.svc.RefreshAuthor(r.Context(), id)
}

func (s *Server) searchAuthor(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.Author(id); err != nil {
		return nil, err
	}
	return map[string]int{"queued": s.svc.SearchMissingInBackground(id)}, nil
}

// ---------------------------------------------------------------- books

func (s *Server) listBooks(w http.ResponseWriter, r *http.Request) (any, error) {
	q := r.URL.Query()
	f := model.BookFilter{Status: q.Get("status"), Query: strings.TrimSpace(q.Get("q"))}
	switch f.Status {
	case "", model.StatusMissing, model.StatusUnreleased, model.StatusDownloading, model.StatusDownloaded:
	default:
		return nil, badRequest("Unknown status filter")
	}
	if v := q.Get("authorId"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, badRequest("authorId must be a number")
		}
		f.AuthorID = id
	}
	if v := q.Get("monitored"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, badRequest("monitored must be true or false")
		}
		f.Monitored = &b
	}
	return s.store.Books(f)
}

func (s *Server) getBook(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return s.svc.BookDetail(id)
}

func (s *Server) addBook(w http.ResponseWriter, r *http.Request) (any, error) {
	var req service.AddBookRequest
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	return s.svc.AddBook(r.Context(), req)
}

func (s *Server) updateBook(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	var patch struct {
		Monitored *bool `json:"monitored"`
	}
	if err := decode(r, &patch); err != nil {
		return nil, err
	}
	if _, err := s.store.Book(id); err != nil {
		return nil, err
	}
	if patch.Monitored != nil {
		if err := s.store.SetBooksMonitored([]int64{id}, *patch.Monitored); err != nil {
			return nil, err
		}
	}
	return s.store.Book(id)
}

func (s *Server) monitorBooks(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		IDs       []int64 `json:"ids"`
		Monitored bool    `json:"monitored"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	return nil, s.store.SetBooksMonitored(req.IDs, req.Monitored)
}

func (s *Server) deleteBook(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return nil, s.svc.DeleteBook(id, queryBool(r, "deleteFiles"))
}

func (s *Server) bookReleases(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return s.svc.BookReleases(r.Context(), id)
}

func (s *Server) grabRelease(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	var req struct {
		ReleaseID string `json:"releaseId"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	return s.svc.GrabRelease(r.Context(), id, req.ReleaseID)
}

func (s *Server) searchBook(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	grabbed, msg, err := s.svc.SearchBook(r.Context(), id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"grabbed": grabbed, "message": msg}, nil
}

// ---------------------------------------------------------------- series

func (s *Server) listSeries(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.store.SeriesList()
}

func (s *Server) getSeries(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return s.store.SeriesDetail(id)
}

// ---------------------------------------------------------------- activity

func (s *Server) listQueue(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.store.Queue()
}

func (s *Server) removeQueueItem(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return nil, s.svc.RemoveQueueItem(r.Context(), id, queryBool(r, "removeFromClient"))
}

func (s *Server) retryImport(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return nil, s.svc.RetryImport(r.Context(), id)
}

func (s *Server) listHistory(w http.ResponseWriter, r *http.Request) (any, error) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 50
	}
	return s.store.History(page, size)
}

// ---------------------------------------------------------------- settings

func (s *Server) listRootFolders(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.svc.RootFolders()
}

func (s *Server) addRootFolder(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	return s.svc.AddRootFolder(req.Path)
}

func (s *Server) deleteRootFolder(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return nil, s.svc.DeleteRootFolder(id)
}

func (s *Server) browse(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.svc.Browse(r.URL.Query().Get("path"))
}

// Secrets are write-only: responses carry hasApiKey/hasPassword instead.
func redactIndexer(i model.Indexer) model.Indexer {
	i.APIKey = ""
	return i
}

func redactClient(c model.DownloadClient) model.DownloadClient {
	c.Password, c.APIKey = "", ""
	return c
}

func (s *Server) listIndexers(w http.ResponseWriter, r *http.Request) (any, error) {
	all, err := s.store.Indexers()
	if err != nil {
		return nil, err
	}
	for i := range all {
		all[i] = redactIndexer(all[i])
	}
	return all, nil
}

// indexerInput is an Indexer whose id may arrive in the body (test route).
type indexerInput struct {
	model.Indexer
	ID *int64 `json:"id"`
}

func (s *Server) saveIndexer(w http.ResponseWriter, r *http.Request) (any, error) {
	var in indexerInput
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	in.Indexer.ID = 0
	if r.Method == http.MethodPut {
		id, err := pathID(r)
		if err != nil {
			return nil, err
		}
		in.Indexer.ID = id
	}
	saved, err := s.svc.SaveIndexer(in.Indexer)
	if err != nil {
		return nil, err
	}
	return redactIndexer(*saved), nil
}

func (s *Server) testIndexer(w http.ResponseWriter, r *http.Request) (any, error) {
	var in indexerInput
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	in.Indexer.ID = 0
	if in.ID != nil {
		in.Indexer.ID = *in.ID
	}
	return s.svc.TestIndexer(r.Context(), in.Indexer), nil
}

func (s *Server) deleteIndexer(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return nil, s.store.DeleteIndexer(id)
}

func (s *Server) listDownloadClients(w http.ResponseWriter, r *http.Request) (any, error) {
	all, err := s.store.DownloadClients()
	if err != nil {
		return nil, err
	}
	for i := range all {
		all[i] = redactClient(all[i])
	}
	return all, nil
}

type clientInput struct {
	model.DownloadClient
	ID *int64 `json:"id"`
}

func (s *Server) saveDownloadClient(w http.ResponseWriter, r *http.Request) (any, error) {
	var in clientInput
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	in.DownloadClient.ID = 0
	if r.Method == http.MethodPut {
		id, err := pathID(r)
		if err != nil {
			return nil, err
		}
		in.DownloadClient.ID = id
	}
	saved, err := s.svc.SaveDownloadClient(in.DownloadClient)
	if err != nil {
		return nil, err
	}
	return redactClient(*saved), nil
}

func (s *Server) testDownloadClient(w http.ResponseWriter, r *http.Request) (any, error) {
	var in clientInput
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	in.DownloadClient.ID = 0
	if in.ID != nil {
		in.DownloadClient.ID = *in.ID
	}
	return s.svc.TestDownloadClient(r.Context(), in.DownloadClient), nil
}

func (s *Server) deleteDownloadClient(w http.ResponseWriter, r *http.Request) (any, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	return nil, s.store.DeleteDownloadClient(id)
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.store.Settings()
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	// Start from the stored values so a client that omits a field does not
	// reset it.
	set, err := s.store.Settings()
	if err != nil {
		return nil, err
	}
	if err := decode(r, &set); err != nil {
		return nil, err
	}
	return s.svc.UpdateSettings(set)
}

// ---------------------------------------------------------------- system

func (s *Server) status(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.svc.Status(currentUser(r).APIKey)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) (any, error) {
	return s.svc.Tasks()
}

func (s *Server) runTask(w http.ResponseWriter, r *http.Request) (any, error) {
	return nil, s.svc.RunTask(r.PathValue("name"))
}
