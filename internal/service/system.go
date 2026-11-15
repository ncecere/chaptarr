package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ncecere/chaptarr/internal/metadata"
	"github.com/ncecere/chaptarr/internal/model"
)

// ---------------------------------------------------------------- status

func (s *Service) Status(apiKey string) (*model.SystemStatus, error) {
	c, err := s.Store.Counts()
	if err != nil {
		return nil, err
	}
	roots, err := s.Store.RootFolders()
	if err != nil {
		return nil, err
	}
	indexers, err := s.Store.Indexers()
	if err != nil {
		return nil, err
	}
	clients, err := s.Store.DownloadClients()
	if err != nil {
		return nil, err
	}
	return &model.SystemStatus{
		Version: s.Version, DataDir: s.DataDir, StartedAt: s.StartedAt,
		AuthorCount: c.Authors, BookCount: c.Books, DownloadedCount: c.Downloaded, MissingCount: c.Missing,
		QueueCount: c.Queue, LibrarySizeBytes: c.LibrarySize,
		HasRootFolder: len(roots) > 0, HasIndexer: len(indexers) > 0, HasDownloadClient: len(clients) > 0,
		APIKey: apiKey,
	}, nil
}

// ---------------------------------------------------------------- settings

func (s *Service) UpdateSettings(set model.Settings) (model.Settings, error) {
	stored, err := s.Store.Settings()
	if err != nil {
		return set, err
	}
	keys := map[string]string{}
	for _, src := range stored.MetadataSources {
		keys[src.Type] = src.APIKey
	}
	set.MetadataSources = model.NormalizeSources(set.MetadataSources)
	for i := range set.MetadataSources {
		src := &set.MetadataSources[i]
		// An empty key on save means "unchanged": responses never carry it.
		if src.APIKey = strings.TrimSpace(src.APIKey); src.APIKey == "" {
			src.APIKey = keys[src.Type]
		}
		src.HasAPIKey = src.APIKey != ""
		if src.URL, err = cleanServerURL(src.URL); err != nil {
			return set, badRequest("%s server URL: %v", metadata.SourceName(src.Type), err)
		}
		if src.Type == model.SourceGoogleBooks && src.Enabled && src.APIKey == "" {
			return set, badRequest("Google Books needs an API key before it can be switched on")
		}
	}
	if set.AudnexusURL, err = cleanServerURL(set.AudnexusURL); err != nil {
		return set, badRequest("Audnexus server URL: %v", err)
	}
	set.AudibleRegion = strings.ToLower(strings.TrimSpace(set.AudibleRegion))
	if !metadata.ValidRegion(set.AudibleRegion) {
		return set, badRequest("Unknown Audible region %q", set.AudibleRegion)
	}
	set.Language = strings.ToLower(strings.TrimSpace(set.Language))
	set.FolderFormat = strings.TrimSpace(set.FolderFormat)
	if !strings.Contains(set.FolderFormat, "{Title}") && !strings.Contains(set.FolderFormat, "{ASIN}") {
		return set, badRequest("The folder format must include {Title} or {ASIN}, or different books would share a folder")
	}
	for _, seg := range strings.Split(strings.ReplaceAll(set.FolderFormat, `\`, "/"), "/") {
		if seg == ".." {
			return set, badRequest("The folder format cannot contain \"..\"")
		}
	}
	switch set.ImportMode {
	case "hardlink", "copy", "move":
	default:
		return set, badRequest("importMode must be hardlink, copy or move")
	}
	if set.MinSeeders < 0 || set.MissingSearchIntervalMinutes < 0 || set.AuthorRefreshIntervalHours < 0 {
		return set, badRequest("Intervals and minimum seeders cannot be negative")
	}
	if set.MissingSearchIntervalMinutes > 0 && set.MissingSearchIntervalMinutes < 15 {
		return set, badRequest("Search for missing books at most every 15 minutes, or indexers will rate-limit you")
	}
	formats := []string{}
	for _, f := range set.PreferredFormats {
		if f = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(f, "."))); f != "" {
			formats = append(formats, f)
		}
	}
	set.PreferredFormats = formats
	if set.DefaultRootFolderID != nil {
		roots, err := s.Store.RootFolders()
		if err != nil {
			return set, err
		}
		found := false
		for _, r := range roots {
			found = found || r.ID == *set.DefaultRootFolderID
		}
		if !found {
			return set, badRequest("The default root folder no longer exists")
		}
	}
	if err := s.Store.SaveSettings(set); err != nil {
		return set, err
	}
	s.configureMetadata(set)
	return set, nil
}

// configureMetadata points the Audible and Audnexus lookups at the user's
// custom servers, if any.
func (s *Service) configureMetadata(set model.Settings) {
	audible := ""
	for _, src := range set.MetadataSources {
		if src.Type == model.SourceAudible {
			audible = src.URL
		}
	}
	s.Meta.Configure(audible, set.AudnexusURL)
}

// cleanServerURL accepts an empty value or an http(s) URL, without a
// trailing slash.
func cleanServerURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("enter a full address starting with http:// or https://")
	}
	return raw, nil
}

type TestMetadataSourceRequest struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	APIKey string `json:"apiKey"`
}

// TestMetadataSource runs a small search against a source with the given,
// possibly unsaved, settings.
func (s *Service) TestMetadataSource(ctx context.Context, r TestMetadataSourceRequest) (model.TestResult, error) {
	set, err := s.Store.Settings()
	if err != nil {
		return model.TestResult{}, err
	}
	base, err := cleanServerURL(r.URL)
	if err != nil {
		return model.TestResult{Message: capitalize(err.Error())}, nil
	}
	const query = "the hobbit tolkien"
	var found []model.BookResult
	switch r.Type {
	case model.SourceAudible:
		// A throwaway client, so the test does not disturb live lookups.
		c := metadata.New()
		c.Configure(base, "")
		found, err = c.SearchBooks(ctx, set.AudibleRegion, query)
	case model.SourceITunes, model.SourceOpenLibrary, model.SourceGoogleBooks:
		key := strings.TrimSpace(r.APIKey)
		if key == "" {
			for _, src := range set.MetadataSources {
				if src.Type == r.Type {
					key = src.APIKey
				}
			}
		}
		found, err = s.Meta.SearchSource(ctx, metadata.SourceConfig{Type: r.Type, URL: base, APIKey: key}, set.AudibleRegion, set.Language, query)
	default:
		return model.TestResult{}, badRequest("Unknown metadata source %q", r.Type)
	}
	if err != nil {
		return model.TestResult{Message: capitalize(err.Error())}, nil
	}
	if len(found) == 0 {
		return model.TestResult{Message: "The server answered, but a search for “The Hobbit” found nothing"}, nil
	}
	return model.TestResult{OK: true, Message: fmt.Sprintf("Found %d results for “The Hobbit”", len(found))}, nil
}

// ---------------------------------------------------------------- root folders

func (s *Service) RootFolders() ([]model.RootFolder, error) {
	folders, err := s.Store.RootFolders()
	if err != nil {
		return nil, err
	}
	for i := range folders {
		fillDisk(&folders[i])
	}
	return folders, nil
}

func fillDisk(f *model.RootFolder) {
	info, err := os.Stat(f.Path)
	if err != nil || !info.IsDir() {
		return
	}
	f.Accessible = writable(f.Path)
	f.FreeBytes, f.TotalBytes = diskUsage(f.Path)
}

func writable(dir string) bool {
	tmp, err := os.CreateTemp(dir, ".chaptarr-write-test-*")
	if err != nil {
		return false
	}
	tmp.Close()
	os.Remove(tmp.Name())
	return true
}

func (s *Service) AddRootFolder(path string) (*model.RootFolder, error) {
	path = strings.TrimSpace(path)
	if !filepath.IsAbs(path) {
		return nil, badRequest("The root folder must be an absolute path")
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, badRequest("%s does not exist", path)
	}
	if err != nil || !info.IsDir() {
		return nil, badRequest("%s is not a readable directory", path)
	}
	if !writable(path) {
		return nil, badRequest("Chaptarr cannot write to %s", path)
	}
	existing, err := s.Store.RootFolders()
	if err != nil {
		return nil, err
	}
	for _, e := range existing {
		if e.Path == path {
			return nil, conflict("%s is already a root folder", path)
		}
		if strings.HasPrefix(path+string(filepath.Separator), e.Path+string(filepath.Separator)) ||
			strings.HasPrefix(e.Path+string(filepath.Separator), path+string(filepath.Separator)) {
			return nil, conflict("%s overlaps the existing root folder %s", path, e.Path)
		}
	}
	id, err := s.Store.AddRootFolder(path)
	if err != nil {
		return nil, err
	}
	// The first root folder becomes the default so imports work without
	// another trip to settings.
	if set, err := s.Store.Settings(); err == nil && set.DefaultRootFolderID == nil {
		set.DefaultRootFolderID = &id
		_ = s.Store.SaveSettings(set)
	}
	f := &model.RootFolder{ID: id, Path: path}
	fillDisk(f)
	go func() {
		if err := s.RunTask("scanLibrary"); err != nil {
			s.Log.Debug("library scan not started", "error", err)
		}
	}()
	return f, nil
}

func (s *Service) DeleteRootFolder(id int64) error {
	if err := s.Store.DeleteRootFolder(id); err != nil {
		return err
	}
	set, err := s.Store.Settings()
	if err != nil {
		return err
	}
	if set.DefaultRootFolderID != nil && *set.DefaultRootFolderID == id {
		set.DefaultRootFolderID = nil
		if roots, err := s.Store.RootFolders(); err == nil && len(roots) > 0 {
			set.DefaultRootFolderID = &roots[0].ID
		}
		return s.Store.SaveSettings(set)
	}
	return nil
}

// Browse lists the sub-directories of path for the folder picker.
func (s *Service) Browse(path string) (*model.DirectoryListing, error) {
	if path == "" {
		if home, err := os.UserHomeDir(); err == nil {
			path = home
		} else {
			path = string(filepath.Separator)
		}
	}
	if !filepath.IsAbs(path) {
		return nil, badRequest("The path must be absolute")
	}
	path = filepath.Clean(path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, badRequest("Cannot read %s", path)
	}
	l := &model.DirectoryListing{Path: path, Directories: []model.DirEntry{}}
	if parent := filepath.Dir(path); parent != path {
		l.Parent = &parent
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		full := filepath.Join(path, e.Name())
		isDir := e.IsDir()
		if e.Type()&fs.ModeSymlink != 0 {
			if info, err := os.Stat(full); err == nil {
				isDir = info.IsDir()
			}
		}
		if isDir {
			l.Directories = append(l.Directories, model.DirEntry{Name: e.Name(), Path: full})
		}
	}
	sort.Slice(l.Directories, func(i, j int) bool {
		return strings.ToLower(l.Directories[i].Name) < strings.ToLower(l.Directories[j].Name)
	})
	return l, nil
}

// ---------------------------------------------------------------- indexers & clients

func validateIndexer(ix *model.Indexer) error {
	ix.Name = strings.TrimSpace(ix.Name)
	ix.URL = strings.TrimSpace(ix.URL)
	ix.Categories = strings.ReplaceAll(ix.Categories, " ", "")
	if ix.Name == "" || ix.URL == "" {
		return badRequest("An indexer needs a name and a URL")
	}
	if ix.Type != "torznab" && ix.Type != "newznab" {
		return badRequest("Indexer type must be torznab or newznab")
	}
	if ix.Priority < 1 || ix.Priority > 50 {
		ix.Priority = 25
	}
	return nil
}

// SaveIndexer creates (ID 0) or updates an indexer. An empty API key on
// update keeps the stored one.
func (s *Service) SaveIndexer(ix model.Indexer) (*model.Indexer, error) {
	if err := validateIndexer(&ix); err != nil {
		return nil, err
	}
	if ix.ID != 0 {
		old, err := s.Store.Indexer(ix.ID)
		if err != nil {
			return nil, notFound(err)
		}
		if ix.APIKey == "" {
			ix.APIKey = old.APIKey
		}
	}
	if err := s.Store.SaveIndexer(&ix); err != nil {
		return nil, err
	}
	return s.Store.Indexer(ix.ID)
}

func (s *Service) TestIndexer(ctx context.Context, ix model.Indexer) model.TestResult {
	if err := validateIndexer(&ix); err != nil {
		return model.TestResult{Message: err.Error()}
	}
	if ix.APIKey == "" && ix.ID != 0 {
		if old, err := s.Store.Indexer(ix.ID); err == nil {
			ix.APIKey = old.APIKey
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := s.Indexer.Test(ctx, ix); err != nil {
		return model.TestResult{Message: capitalize(err.Error())}
	}
	return model.TestResult{OK: true, Message: "Connected to " + ix.Name}
}

func validateClient(c *model.DownloadClient) error {
	c.Name = strings.TrimSpace(c.Name)
	c.URL = strings.TrimSpace(c.URL)
	c.Category = strings.TrimSpace(c.Category)
	if c.Name == "" || c.URL == "" {
		return badRequest("A download client needs a name and a URL")
	}
	if c.Type != "qbittorrent" && c.Type != "sabnzbd" {
		return badRequest("Download client type must be qbittorrent or sabnzbd")
	}
	return nil
}

func (s *Service) SaveDownloadClient(c model.DownloadClient) (*model.DownloadClient, error) {
	if err := validateClient(&c); err != nil {
		return nil, err
	}
	if c.ID != 0 {
		old, err := s.Store.DownloadClient(c.ID)
		if err != nil {
			return nil, notFound(err)
		}
		if c.Password == "" {
			c.Password = old.Password
		}
		if c.APIKey == "" {
			c.APIKey = old.APIKey
		}
	}
	if err := s.Store.SaveDownloadClient(&c); err != nil {
		return nil, err
	}
	return s.Store.DownloadClient(c.ID)
}

func (s *Service) TestDownloadClient(ctx context.Context, c model.DownloadClient) model.TestResult {
	if err := validateClient(&c); err != nil {
		return model.TestResult{Message: err.Error()}
	}
	if c.ID != 0 {
		if old, err := s.Store.DownloadClient(c.ID); err == nil {
			if c.Password == "" {
				c.Password = old.Password
			}
			if c.APIKey == "" {
				c.APIKey = old.APIKey
			}
		}
	}
	// Build a throwaway connection: unsaved form values must not replace
	// the cached client.
	c.ID = 0
	client, err := s.client(c)
	if err != nil {
		return model.TestResult{Message: capitalize(err.Error())}
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := client.Test(ctx); err != nil {
		return model.TestResult{Message: capitalize(err.Error())}
	}
	return model.TestResult{OK: true, Message: "Connected to " + c.Name}
}
