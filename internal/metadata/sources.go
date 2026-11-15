package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/textutil"
)

// The additional sources. Audible is the catalogue Chaptarr is built on;
// these list books it does not have and fill in details it lacks. None of
// them knows narrators, runtimes or series, so a book added from one is
// searched for by title and author alone.

// SourceConfig is one additional source's settings.
type SourceConfig struct {
	Type   string
	URL    string // empty for the public endpoint
	APIKey string
}

// DefaultURL is the public endpoint of a source, shown as the placeholder
// for its custom server setting.
func DefaultURL(sourceType string) string {
	switch sourceType {
	case model.SourceAudible:
		return "https://api.audible.com"
	case model.SourceITunes:
		return "https://itunes.apple.com"
	case model.SourceOpenLibrary:
		return "https://openlibrary.org"
	case model.SourceGoogleBooks:
		return "https://www.googleapis.com/books/v1"
	}
	return ""
}

// SourceName is the source's display name.
func SourceName(sourceType string) string {
	switch sourceType {
	case model.SourceAudible:
		return "Audible"
	case model.SourceITunes:
		return "iTunes"
	case model.SourceOpenLibrary:
		return "Open Library"
	case model.SourceGoogleBooks:
		return "Google Books"
	}
	return sourceType
}

func (cfg SourceConfig) base() string {
	if cfg.URL != "" {
		return strings.TrimRight(cfg.URL, "/")
	}
	return DefaultURL(cfg.Type)
}

// language is how each source names a language. Settings hold Audible's
// name for it ("english").
type language struct{ iso1, iso3 string }

var languages = map[string]language{
	"english": {"en", "eng"}, "german": {"de", "ger"}, "french": {"fr", "fre"}, "spanish": {"es", "spa"},
	"italian": {"it", "ita"}, "portuguese": {"pt", "por"}, "dutch": {"nl", "dut"}, "japanese": {"ja", "jpn"},
	"russian": {"ru", "rus"}, "swedish": {"sv", "swe"}, "danish": {"da", "dan"}, "norwegian": {"no", "nor"},
	"finnish": {"fi", "fin"}, "polish": {"pl", "pol"}, "hindi": {"hi", "hin"}, "mandarin_chinese": {"zh", "chi"},
	"korean": {"ko", "kor"}, "turkish": {"tr", "tur"}, "arabic": {"ar", "ara"}, "hebrew": {"he", "heb"},
}

func languageName(code string) string {
	code = strings.ToLower(code)
	for name, l := range languages {
		if l.iso1 == code || l.iso3 == code {
			return name
		}
	}
	return ""
}

// iTunes stores are per country; use the one matching the Audible region.
var itunesCountry = map[string]string{
	"us": "us", "uk": "gb", "ca": "ca", "au": "au", "de": "de", "fr": "fr", "es": "es", "it": "it", "in": "in", "jp": "jp",
}

// SearchSource runs a keyword search on an additional source.
func (c *Client) SearchSource(ctx context.Context, cfg SourceConfig, region, lang, query string) ([]model.BookResult, error) {
	switch cfg.Type {
	case model.SourceITunes:
		return c.itunesSearch(ctx, cfg, region, query)
	case model.SourceOpenLibrary:
		return c.openLibrarySearch(ctx, cfg, lang, query)
	case model.SourceGoogleBooks:
		return c.googleBooksSearch(ctx, cfg, lang, query)
	}
	return nil, fmt.Errorf("unknown metadata source %q", cfg.Type)
}

// SourceBook fetches one book from an additional source by its prefixed id.
func (c *Client) SourceBook(ctx context.Context, cfg SourceConfig, region, id string) (*model.BookResult, error) {
	_, raw, _ := strings.Cut(id, ":")
	if raw == "" {
		return nil, fmt.Errorf("malformed id %q", id)
	}
	var (
		r   *model.BookResult
		err error
	)
	switch cfg.Type {
	case model.SourceITunes:
		r, err = c.itunesBook(ctx, cfg, region, raw)
	case model.SourceOpenLibrary:
		r, err = c.openLibraryBook(ctx, cfg, raw)
	case model.SourceGoogleBooks:
		r, err = c.googleBooksBook(ctx, cfg, raw)
	default:
		err = fmt.Errorf("unknown metadata source %q", cfg.Type)
	}
	return r, err
}

// SameWork reports whether two entries are the same book: the same main
// title and a shared author. Sources disagree on subtitles and on how they
// credit co-authors, so both are compared loosely.
func SameWork(a, b model.BookResult) bool {
	if textutil.Normalize(textutil.MainTitle(a.Title)) != textutil.Normalize(textutil.MainTitle(b.Title)) {
		return false
	}
	for _, x := range a.Authors {
		for _, y := range b.Authors {
			if textutil.Normalize(x.Name) == textutil.Normalize(y.Name) {
				return true
			}
		}
	}
	return false
}

func authorRefs(names ...string) []model.AuthorRef {
	out := []model.AuthorRef{}
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, model.AuthorRef{Name: n})
		}
	}
	return out
}

// ---------------------------------------------------------------- iTunes

type itunesItem struct {
	Kind        string `json:"wrapperType"`
	ID          int64  `json:"collectionId"`
	Title       string `json:"collectionName"`
	Artist      string `json:"artistName"`
	Artwork     string `json:"artworkUrl100"`
	ReleaseDate string `json:"releaseDate"`
	Description string `json:"description"`
	Copyright   string `json:"copyright"`
}

var (
	itunesEditionRe = regexp.MustCompile(`(?i)\s*\((un)?abridged\)\s*$`)
	itunesArtRe     = regexp.MustCompile(`/\d+x\d+bb\.`)
	// Co-authors arrive as one string: "Terry Pratchett & Neil Gaiman".
	itunesAuthorSep = regexp.MustCompile(`\s*(?:,|&| and )\s*`)
)

func (it itunesItem) toResult() model.BookResult {
	r := model.BookResult{
		ASIN:        "it:" + strconv.FormatInt(it.ID, 10),
		Source:      model.SourceITunes,
		Title:       strings.TrimSpace(itunesEditionRe.ReplaceAllString(it.Title, "")),
		Authors:     authorRefs(itunesAuthorSep.Split(it.Artist, -1)...),
		Narrators:   []string{},
		Description: it.Description,
		CoverURL:    itunesArtRe.ReplaceAllString(it.Artwork, "/600x600bb."),
		Abridged:    strings.Contains(strings.ToLower(it.Title), "(abridged)"),
	}
	if len(it.ReleaseDate) >= 10 {
		r.ReleaseDate = it.ReleaseDate[:10]
	}
	return r
}

func (c *Client) itunesSearch(ctx context.Context, cfg SourceConfig, region, query string) ([]model.BookResult, error) {
	q := url.Values{"term": {query}, "media": {"audiobook"}, "entity": {"audiobook"}, "limit": {"25"}}
	if country, ok := itunesCountry[region]; ok {
		q.Set("country", country)
	}
	var out struct {
		Results []itunesItem `json:"results"`
	}
	if err := c.getJSON(ctx, cfg.base()+"/search?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	res := []model.BookResult{}
	for _, it := range out.Results {
		if it.ID != 0 && it.Title != "" {
			res = append(res, it.toResult())
		}
	}
	return res, nil
}

func (c *Client) itunesBook(ctx context.Context, cfg SourceConfig, region, id string) (*model.BookResult, error) {
	q := url.Values{"id": {id}, "entity": {"audiobook"}}
	if country, ok := itunesCountry[region]; ok {
		q.Set("country", country)
	}
	var out struct {
		Results []itunesItem `json:"results"`
	}
	if err := c.getJSON(ctx, cfg.base()+"/lookup?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	for _, it := range out.Results {
		if strconv.FormatInt(it.ID, 10) == id && it.Title != "" {
			r := it.toResult()
			return &r, nil
		}
	}
	return nil, fmt.Errorf("audiobook %s not found on iTunes", id)
}

// ---------------------------------------------------------------- Open Library

const openLibraryFields = "key,title,subtitle,author_name,first_publish_year,cover_i,language,publisher"

type openLibraryDoc struct {
	Key       string   `json:"key"` // "/works/OL5734647W"
	Title     string   `json:"title"`
	Subtitle  string   `json:"subtitle"`
	Authors   []string `json:"author_name"`
	Year      int      `json:"first_publish_year"`
	Cover     int64    `json:"cover_i"`
	Languages []string `json:"language"`
	Publisher []string `json:"publisher"`
}

func (d openLibraryDoc) toResult(lang string) model.BookResult {
	r := model.BookResult{
		ASIN:      "ol:" + strings.TrimPrefix(d.Key, "/works/"),
		Source:    model.SourceOpenLibrary,
		Title:     strings.TrimSpace(d.Title),
		Subtitle:  strings.TrimSpace(d.Subtitle),
		Authors:   authorRefs(d.Authors...),
		Narrators: []string{},
	}
	if d.Year > 0 {
		r.ReleaseDate = fmt.Sprintf("%04d-01-01", d.Year)
	}
	if d.Cover > 0 {
		r.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", d.Cover)
	}
	// A work lists the languages of all its editions; name the wanted one
	// if it is among them, otherwise the only one it has.
	want := languages[lang].iso3
	for _, l := range d.Languages {
		if l == want {
			r.Language = lang
		}
	}
	if r.Language == "" && len(d.Languages) == 1 {
		r.Language = languageName(d.Languages[0])
	}
	return r
}

func (c *Client) openLibraryDocs(ctx context.Context, cfg SourceConfig, q string, limit int) ([]openLibraryDoc, error) {
	v := url.Values{"q": {q}, "limit": {strconv.Itoa(limit)}, "fields": {openLibraryFields}}
	var out struct {
		Docs []openLibraryDoc `json:"docs"`
	}
	err := c.getJSON(ctx, cfg.base()+"/search.json?"+v.Encode(), &out)
	return out.Docs, err
}

func (c *Client) openLibrarySearch(ctx context.Context, cfg SourceConfig, lang, query string) ([]model.BookResult, error) {
	if l, ok := languages[lang]; ok {
		query += " language:" + l.iso3
	}
	docs, err := c.openLibraryDocs(ctx, cfg, query, 20)
	if err != nil {
		return nil, err
	}
	res := []model.BookResult{}
	for _, d := range docs {
		if strings.HasPrefix(d.Key, "/works/") && d.Title != "" && len(d.Authors) > 0 {
			res = append(res, d.toResult(lang))
		}
	}
	return res, nil
}

func (c *Client) openLibraryBook(ctx context.Context, cfg SourceConfig, id string) (*model.BookResult, error) {
	docs, err := c.openLibraryDocs(ctx, cfg, "key:/works/"+id, 1)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 || docs[0].Title == "" {
		return nil, fmt.Errorf("work %s not found on Open Library", id)
	}
	r := docs[0].toResult("")
	// The description lives on the work record, as a string or {value}.
	var work struct {
		Description json.RawMessage `json:"description"`
	}
	if err := c.getJSON(ctx, cfg.base()+"/works/"+url.PathEscape(id)+".json", &work); err == nil && len(work.Description) > 0 {
		var text string
		var obj struct {
			Value string `json:"value"`
		}
		if json.Unmarshal(work.Description, &text) == nil {
			r.Description = text
		} else if json.Unmarshal(work.Description, &obj) == nil {
			r.Description = obj.Value
		}
	}
	return &r, nil
}

// ---------------------------------------------------------------- Google Books

type googleVolume struct {
	ID   string `json:"id"`
	Info struct {
		Title       string            `json:"title"`
		Subtitle    string            `json:"subtitle"`
		Authors     []string          `json:"authors"`
		Publisher   string            `json:"publisher"`
		Published   string            `json:"publishedDate"` // "2005", "2005-01" or "2005-01-15"
		Description string            `json:"description"`
		Language    string            `json:"language"`
		Images      map[string]string `json:"imageLinks"`
	} `json:"volumeInfo"`
}

func (v googleVolume) toResult() model.BookResult {
	r := model.BookResult{
		ASIN:        "gb:" + v.ID,
		Source:      model.SourceGoogleBooks,
		Title:       strings.TrimSpace(v.Info.Title),
		Subtitle:    strings.TrimSpace(v.Info.Subtitle),
		Authors:     authorRefs(v.Info.Authors...),
		Narrators:   []string{},
		Description: v.Info.Description,
		Publisher:   v.Info.Publisher,
		Language:    languageName(v.Info.Language),
	}
	switch len(v.Info.Published) {
	case 4:
		r.ReleaseDate = v.Info.Published + "-01-01"
	case 7:
		r.ReleaseDate = v.Info.Published + "-01"
	case 10:
		r.ReleaseDate = v.Info.Published
	}
	for _, size := range []string{"extraLarge", "large", "medium", "small", "thumbnail", "smallThumbnail"} {
		if u := v.Info.Images[size]; u != "" {
			r.CoverURL = strings.Replace(u, "http://", "https://", 1)
			break
		}
	}
	return r
}

func googleKey(cfg SourceConfig) map[string]string {
	return map[string]string{"X-Goog-Api-Key": cfg.APIKey}
}

func (c *Client) googleBooksSearch(ctx context.Context, cfg SourceConfig, lang, query string) ([]model.BookResult, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Google Books needs an API key")
	}
	q := url.Values{"q": {query}, "maxResults": {"20"}, "printType": {"books"}}
	if l, ok := languages[lang]; ok {
		q.Set("langRestrict", l.iso1)
	}
	var out struct {
		Items []googleVolume `json:"items"`
	}
	if err := c.getJSONWith(ctx, cfg.base()+"/volumes?"+q.Encode(), googleKey(cfg), &out); err != nil {
		return nil, err
	}
	res := []model.BookResult{}
	for _, v := range out.Items {
		if v.ID != "" && v.Info.Title != "" && len(v.Info.Authors) > 0 {
			res = append(res, v.toResult())
		}
	}
	return res, nil
}

func (c *Client) googleBooksBook(ctx context.Context, cfg SourceConfig, id string) (*model.BookResult, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Google Books needs an API key")
	}
	var v googleVolume
	if err := c.getJSONWith(ctx, cfg.base()+"/volumes/"+url.PathEscape(id), googleKey(cfg), &v); err != nil {
		return nil, err
	}
	if v.Info.Title == "" {
		return nil, fmt.Errorf("volume %s not found on Google Books", id)
	}
	r := v.toResult()
	return &r, nil
}
