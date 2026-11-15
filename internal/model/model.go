// Package model holds the types shared by the store, the services and the
// HTTP API. JSON shapes here mirror web/src/lib/api.ts, which is the contract
// with the UI.
package model

import (
	"strings"
	"time"
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type AuthStatus struct {
	SetupRequired bool  `json:"setupRequired"`
	Authenticated bool  `json:"authenticated"`
	User          *User `json:"user"`
}

const (
	StatusMissing     = "missing"
	StatusUnreleased  = "unreleased"
	StatusDownloading = "downloading"
	StatusDownloaded  = "downloaded"
)

type Author struct {
	ID              int64      `json:"id"`
	ASIN            string     `json:"asin"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	ImageURL        string     `json:"imageUrl"`
	Monitored       bool       `json:"monitored"`
	MonitorNewBooks bool       `json:"monitorNewBooks"`
	AddedAt         time.Time  `json:"addedAt"`
	LastSyncAt      *time.Time `json:"lastSyncAt"`
	BookCount       int        `json:"bookCount"`
	DownloadedCount int        `json:"downloadedCount"`
	MissingCount    int        `json:"missingCount"`
}

type AuthorDetail struct {
	Author
	Books []Book `json:"books"`
}

type SeriesRef struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Position string `json:"position"`
}

type Book struct {
	ID             int64      `json:"id"`
	ASIN           string     `json:"asin"`
	Title          string     `json:"title"`
	Subtitle       string     `json:"subtitle"`
	AuthorID       int64      `json:"authorId"`
	AuthorName     string     `json:"authorName"`
	Authors        []string   `json:"authors"`
	Narrators      []string   `json:"narrators"`
	Series         *SeriesRef `json:"series"`
	Description    string     `json:"description"`
	CoverURL       string     `json:"coverUrl"`
	ReleaseDate    string     `json:"releaseDate"`
	RuntimeMinutes int        `json:"runtimeMinutes"`
	Language       string     `json:"language"`
	Publisher      string     `json:"publisher"`
	Monitored      bool       `json:"monitored"`
	Status         string     `json:"status"`
	Path           string     `json:"path"`
	SizeBytes      int64      `json:"sizeBytes"`
	Format         string     `json:"format"`
	AddedAt        time.Time  `json:"addedAt"`
}

type BookFile struct {
	ID        int64  `json:"id"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	Format    string `json:"format"`
}

type BookDetail struct {
	Book
	Files   []BookFile     `json:"files"`
	History []HistoryEvent `json:"history"`
}

type Series struct {
	ID              int64  `json:"id"`
	ASIN            string `json:"asin"`
	Name            string `json:"name"`
	BookCount       int    `json:"bookCount"`
	DownloadedCount int    `json:"downloadedCount"`
	CoverURL        string `json:"coverUrl"`
	AuthorName      string `json:"authorName"`
}

type SeriesDetail struct {
	Series
	Books []Book `json:"books"`
}

type BookFilter struct {
	Status    string
	AuthorID  int64
	SeriesID  int64
	Monitored *bool
	Query     string
}

// ---- metadata search results

type AuthorRef struct {
	ASIN string `json:"asin"`
	Name string `json:"name"`
}

type BookResult struct {
	// ASIN is the catalogue id. For Audible it is the ASIN; other sources
	// prefix theirs ("it:1442041162", "ol:OL5734647W", "gb:abc123").
	ASIN string `json:"asin"`
	// Source is the metadata source the entry came from (SourceAudible, ...).
	Source         string      `json:"source"`
	Title          string      `json:"title"`
	Subtitle       string      `json:"subtitle"`
	Authors        []AuthorRef `json:"authors"`
	Narrators      []string    `json:"narrators"`
	SeriesName     string      `json:"seriesName"`
	SeriesPosition string      `json:"seriesPosition"`
	Description    string      `json:"description"`
	CoverURL       string      `json:"coverUrl"`
	ReleaseDate    string      `json:"releaseDate"`
	RuntimeMinutes int         `json:"runtimeMinutes"`
	Language       string      `json:"language"`
	Publisher      string      `json:"publisher"`
	LibraryBookID  *int64      `json:"libraryBookId"`

	// Not part of the API: used when importing a catalogue.
	SeriesASIN string `json:"-"`
	Abridged   bool   `json:"-"`
	NumRatings int    `json:"-"`
}

type AuthorResult struct {
	ASIN            string `json:"asin"`
	Name            string `json:"name"`
	ImageURL        string `json:"imageUrl"`
	Description     string `json:"description"`
	LibraryAuthorID *int64 `json:"libraryAuthorId"`
}

// ---- releases, queue, history

const (
	ProtocolTorrent = "torrent"
	ProtocolUsenet  = "usenet"
)

type Release struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Indexer    string   `json:"indexer"`
	Protocol   string   `json:"protocol"`
	SizeBytes  int64    `json:"sizeBytes"`
	Seeders    *int     `json:"seeders"`
	Leechers   *int     `json:"leechers"`
	AgeDays    int      `json:"ageDays"`
	Format     string   `json:"format"`
	Score      int      `json:"score"`
	Rejections []string `json:"rejections"`
	InfoURL    string   `json:"infoUrl"`

	// Not part of the API.
	GUID        string `json:"-"`
	DownloadURL string `json:"-"`
	IndexerID   int64  `json:"-"`
	Priority    int    `json:"-"`
}

const (
	QueueQueued      = "queued"
	QueueDownloading = "downloading"
	QueuePaused      = "paused"
	QueueCompleted   = "completed"
	QueueImporting   = "importing"
	QueueFailed      = "failed"
)

type QueueItem struct {
	ID             int64     `json:"id"`
	BookID         int64     `json:"bookId"`
	BookTitle      string    `json:"bookTitle"`
	AuthorName     string    `json:"authorName"`
	CoverURL       string    `json:"coverUrl"`
	ReleaseTitle   string    `json:"releaseTitle"`
	Indexer        string    `json:"indexer"`
	Protocol       string    `json:"protocol"`
	DownloadClient string    `json:"downloadClient"`
	Status         string    `json:"status"`
	Progress       float64   `json:"progress"`
	SizeBytes      int64     `json:"sizeBytes"`
	ETASeconds     *int64    `json:"etaSeconds"`
	Error          string    `json:"error"`
	AddedAt        time.Time `json:"addedAt"`

	// Not part of the API.
	ClientID     int64  `json:"-"`
	ClientItemID string `json:"-"`
	GUID         string `json:"-"`
}

const (
	EventGrabbed        = "grabbed"
	EventImported       = "imported"
	EventDownloadFailed = "downloadFailed"
	EventImportFailed   = "importFailed"
	EventFileDeleted    = "fileDeleted"
)

type HistoryEvent struct {
	ID          int64     `json:"id"`
	BookID      int64     `json:"bookId"`
	BookTitle   string    `json:"bookTitle"`
	AuthorName  string    `json:"authorName"`
	Type        string    `json:"type"`
	SourceTitle string    `json:"sourceTitle"`
	Message     string    `json:"message"`
	Date        time.Time `json:"date"`
}

type HistoryPage struct {
	Items    []HistoryEvent `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

// ---- settings

type RootFolder struct {
	ID         int64  `json:"id"`
	Path       string `json:"path"`
	Accessible bool   `json:"accessible"`
	FreeBytes  uint64 `json:"freeBytes"`
	TotalBytes uint64 `json:"totalBytes"`
}

type Indexer struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	APIKey     string `json:"apiKey"`
	HasAPIKey  bool   `json:"hasApiKey"`
	Categories string `json:"categories"`
	Enabled    bool   `json:"enabled"`
	Priority   int    `json:"priority"`
}

type DownloadClient struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	HasPassword bool   `json:"hasPassword"`
	APIKey      string `json:"apiKey"`
	HasAPIKey   bool   `json:"hasApiKey"`
	Category    string `json:"category"`
	Enabled     bool   `json:"enabled"`
}

type TestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// Metadata sources. Audible is the primary catalogue: authors, series and
// narrators come only from it. The others add books it does not list and
// fill in descriptions and covers it lacks.
const (
	SourceAudible     = "audible"
	SourceITunes      = "itunes"
	SourceOpenLibrary = "openlibrary"
	SourceGoogleBooks = "googlebooks"
)

type MetadataSource struct {
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	// URL replaces the source's public endpoint, for a mirror or a
	// self-hosted server. Empty uses the default.
	URL string `json:"url"`
	// APIKey is write-only in the API: responses carry HasAPIKey instead,
	// and an empty value on save keeps the stored key.
	APIKey    string `json:"apiKey,omitempty"`
	HasAPIKey bool   `json:"hasApiKey"`
}

// SourceOfID tells which source a catalogue id belongs to.
func SourceOfID(id string) string {
	switch {
	case strings.HasPrefix(id, "it:"):
		return SourceITunes
	case strings.HasPrefix(id, "ol:"):
		return SourceOpenLibrary
	case strings.HasPrefix(id, "gb:"):
		return SourceGoogleBooks
	}
	return SourceAudible
}

func defaultSources() []MetadataSource {
	return []MetadataSource{
		{Type: SourceAudible, Enabled: true},
		{Type: SourceITunes},
		{Type: SourceOpenLibrary},
		{Type: SourceGoogleBooks},
	}
}

// NormalizeSources returns the list in the user's priority order with
// Audible first and always enabled, unknown and repeated entries dropped,
// and sources added in later versions appended, disabled.
func NormalizeSources(in []MetadataSource) []MetadataSource {
	known := map[string]bool{}
	for _, d := range defaultSources() {
		known[d.Type] = true
	}
	seen := map[string]bool{}
	out := []MetadataSource{{Type: SourceAudible, Enabled: true}}
	for _, m := range in {
		if !known[m.Type] || seen[m.Type] {
			continue
		}
		seen[m.Type] = true
		m.HasAPIKey = m.APIKey != ""
		if m.Type == SourceAudible {
			m.Enabled = true
			out[0] = m
			continue
		}
		out = append(out, m)
	}
	seen[SourceAudible] = true
	for _, d := range defaultSources() {
		if !seen[d.Type] {
			out = append(out, d)
		}
	}
	return out
}

type Settings struct {
	MetadataSources []MetadataSource `json:"metadataSources"`
	// AudnexusURL replaces the public Audnexus server (author photos and
	// biographies). Empty uses the default.
	AudnexusURL                  string   `json:"audnexusUrl"`
	AudibleRegion                string   `json:"audibleRegion"`
	Language                     string   `json:"language"`
	FolderFormat                 string   `json:"folderFormat"`
	PreferredFormats             []string `json:"preferredFormats"`
	ImportMode                   string   `json:"importMode"`
	MinSeeders                   int      `json:"minSeeders"`
	MissingSearchIntervalMinutes int      `json:"missingSearchIntervalMinutes"`
	AuthorRefreshIntervalHours   int      `json:"authorRefreshIntervalHours"`
	DefaultRootFolderID          *int64   `json:"defaultRootFolderId"`
}

func DefaultSettings() Settings {
	return Settings{
		MetadataSources:              defaultSources(),
		AudibleRegion:                "us",
		Language:                     "english",
		FolderFormat:                 "{Author}/{Series}/{Title}",
		PreferredFormats:             []string{"m4b", "mp3", "m4a", "mp4", "flac", "opus"},
		ImportMode:                   "hardlink",
		MinSeeders:                   1,
		MissingSearchIntervalMinutes: 360,
		AuthorRefreshIntervalHours:   24,
	}
}

// ---- system

type SystemStatus struct {
	Version           string    `json:"version"`
	DataDir           string    `json:"dataDir"`
	StartedAt         time.Time `json:"startedAt"`
	AuthorCount       int       `json:"authorCount"`
	BookCount         int       `json:"bookCount"`
	DownloadedCount   int       `json:"downloadedCount"`
	MissingCount      int       `json:"missingCount"`
	QueueCount        int       `json:"queueCount"`
	LibrarySizeBytes  int64     `json:"librarySizeBytes"`
	HasRootFolder     bool      `json:"hasRootFolder"`
	HasIndexer        bool      `json:"hasIndexer"`
	HasDownloadClient bool      `json:"hasDownloadClient"`
	APIKey            string    `json:"apiKey"`
}

type Task struct {
	Name            string     `json:"name"`
	Label           string     `json:"label"`
	IntervalSeconds int        `json:"intervalSeconds"`
	Running         bool       `json:"running"`
	LastRunAt       *time.Time `json:"lastRunAt"`
	LastDurationMs  int64      `json:"lastDurationMs"`
	LastError       string     `json:"lastError"`
	NextRunAt       *time.Time `json:"nextRunAt"`
}

type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

type DirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type DirectoryListing struct {
	Path        string     `json:"path"`
	Parent      *string    `json:"parent"`
	Directories []DirEntry `json:"directories"`
}
