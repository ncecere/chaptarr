// Package metadata looks up audiobooks and authors. Book data comes from
// Audible's public catalogue API (no account needed); author photos and bios
// come from Audnexus, which Audible's catalogue does not expose.
package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ncecere/chaptarr/internal/model"
	"github.com/ncecere/chaptarr/internal/textutil"
)

var regionHosts = map[string]string{
	"us": "api.audible.com",
	"uk": "api.audible.co.uk",
	"ca": "api.audible.ca",
	"au": "api.audible.com.au",
	"de": "api.audible.de",
	"fr": "api.audible.fr",
	"es": "api.audible.es",
	"it": "api.audible.it",
	"in": "api.audible.in",
	"jp": "api.audible.co.jp",
}

func ValidRegion(r string) bool { _, ok := regionHosts[r]; return ok }

const responseGroups = "contributors,product_desc,product_attrs,media,series,product_extended_attrs,rating"

type Client struct {
	HTTP *http.Client
	// AudibleBase and AudnexusBase override the real endpoints in tests.
	AudibleBase  string
	AudnexusBase string

	// The user's custom servers (Settings → Metadata), set by Configure.
	mu             sync.RWMutex
	customAudible  string
	customAudnexus string
}

// Configure sets the user's custom Audible and Audnexus servers; an empty
// URL means the public one.
func (c *Client) Configure(audibleURL, audnexusURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.customAudible = strings.TrimRight(audibleURL, "/")
	c.customAudnexus = strings.TrimRight(audnexusURL, "/")
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) audibleURL(region string) string {
	c.mu.RLock()
	custom := c.customAudible
	c.mu.RUnlock()
	if custom != "" {
		return custom
	}
	if c.AudibleBase != "" {
		return c.AudibleBase
	}
	host, ok := regionHosts[region]
	if !ok {
		host = regionHosts["us"]
	}
	return "https://" + host
}

func (c *Client) audnexusURL() string {
	c.mu.RLock()
	custom := c.customAudnexus
	c.mu.RUnlock()
	if custom != "" {
		return custom
	}
	if c.AudnexusBase != "" {
		return c.AudnexusBase
	}
	return "https://api.audnex.us"
}

func (c *Client) getJSON(ctx context.Context, u string, out any) error {
	return c.getJSONWith(ctx, u, nil, out)
}

// getJSONWith sends extra headers, which is where API keys go: a key in
// the URL would end up in error messages and logs.
func (c *Client) getJSONWith(ctx context.Context, u string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Chaptarr")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 300))
		return fmt.Errorf("%s returned %s: %s", req.URL.Host, res.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(res.Body).Decode(out)
}

type product struct {
	ASIN    string `json:"asin"`
	Title   string `json:"title"`
	Sub     string `json:"subtitle"`
	Authors []struct {
		ASIN string `json:"asin"`
		Name string `json:"name"`
	} `json:"authors"`
	Narrators []struct {
		Name string `json:"name"`
	} `json:"narrators"`
	Series []struct {
		ASIN     string `json:"asin"`
		Title    string `json:"title"`
		Sequence string `json:"sequence"`
	} `json:"series"`
	Summary      string            `json:"publisher_summary"`
	Images       map[string]string `json:"product_images"`
	ReleaseDate  string            `json:"release_date"`
	Runtime      int               `json:"runtime_length_min"`
	Language     string            `json:"language"`
	Publisher    string            `json:"publisher_name"`
	FormatType   string            `json:"format_type"`
	DeliveryType string            `json:"content_delivery_type"`
	ContentType  string            `json:"content_type"`
	Rating       struct {
		Overall struct {
			NumRatings int `json:"num_ratings"`
		} `json:"overall_distribution"`
	} `json:"rating"`
}

// isBook filters out podcasts, periodicals, lectures-as-episodes and the
// like, which share the catalogue with audiobooks.
func (p product) isBook() bool {
	if p.ASIN == "" || p.Title == "" {
		return false
	}
	switch p.DeliveryType {
	case "SinglePartBook", "MultiPartBook", "":
		return true
	}
	return false
}

func (p product) toResult() model.BookResult {
	r := model.BookResult{
		ASIN:           p.ASIN,
		Source:         model.SourceAudible,
		Title:          strings.TrimSpace(p.Title),
		Subtitle:       strings.TrimSpace(p.Sub),
		Authors:        []model.AuthorRef{},
		Narrators:      []string{},
		Description:    p.Summary,
		ReleaseDate:    p.ReleaseDate,
		RuntimeMinutes: p.Runtime,
		Language:       strings.ToLower(p.Language),
		Publisher:      p.Publisher,
		Abridged:       strings.EqualFold(p.FormatType, "abridged"),
		NumRatings:     p.Rating.Overall.NumRatings,
	}
	for _, a := range p.Authors {
		if name := strings.TrimSpace(a.Name); name != "" {
			r.Authors = append(r.Authors, model.AuthorRef{ASIN: a.ASIN, Name: name})
		}
	}
	for _, n := range p.Narrators {
		if name := strings.TrimSpace(n.Name); name != "" {
			r.Narrators = append(r.Narrators, name)
		}
	}
	// A book can sit in several series (a saga and its sub-series). Prefer
	// one that says where the book goes.
	for _, s := range p.Series {
		if r.SeriesName == "" || (r.SeriesPosition == "" && s.Sequence != "") {
			r.SeriesName, r.SeriesASIN, r.SeriesPosition = s.Title, s.ASIN, s.Sequence
		}
	}
	// Largest image available.
	best := 0
	for size, u := range p.Images {
		if n, _ := strconv.Atoi(size); n >= best {
			best, r.CoverURL = n, u
		}
	}
	return r
}

func (c *Client) products(ctx context.Context, region string, q url.Values) ([]product, int, error) {
	q.Set("response_groups", responseGroups)
	q.Set("image_sizes", "500")
	var out struct {
		Products []product `json:"products"`
		Total    int       `json:"total_results"`
	}
	err := c.getJSON(ctx, c.audibleURL(region)+"/1.0/catalog/products?"+q.Encode(), &out)
	return out.Products, out.Total, err
}

// SearchBooks runs a keyword search.
func (c *Client) SearchBooks(ctx context.Context, region, query string) ([]model.BookResult, error) {
	ps, _, err := c.products(ctx, region, url.Values{
		"keywords":         {query},
		"num_results":      {"30"},
		"products_sort_by": {"Relevance"},
	})
	if err != nil {
		return nil, err
	}
	out := []model.BookResult{}
	for _, p := range ps {
		if p.isBook() {
			out = append(out, p.toResult())
		}
	}
	return out, nil
}

// Book fetches one book by ASIN.
func (c *Client) Book(ctx context.Context, region, asin string) (*model.BookResult, error) {
	q := url.Values{"response_groups": {responseGroups}, "image_sizes": {"500"}}
	var out struct {
		Product product `json:"product"`
	}
	if err := c.getJSON(ctx, c.audibleURL(region)+"/1.0/catalog/products/"+url.PathEscape(asin)+"?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	if out.Product.Title == "" {
		return nil, fmt.Errorf("book %s not found in the %s catalogue", asin, region)
	}
	r := out.Product.toResult()
	return &r, nil
}

// AuthorBooks returns the author's catalogue in the given language ("" for
// any), with one entry per work: Audible lists re-recordings, abridgements
// and dramatisations of the same title separately, and following an author
// should not queue all of them.
//
// The catalogue's author listing is lossy: any single walk silently omits
// titles, and which ones depends on the sort order. Several differently
// sorted walks are merged, which in practice recovers nearly everything;
// the periodic author refresh picks up stragglers, since books are only
// ever added.
func (c *Client) AuthorBooks(ctx context.Context, region, authorASIN, authorName, language string) ([]model.BookResult, error) {
	sorts := []string{"-ReleaseDate", "Title", "AvgRating"}
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		byASIN  = map[string]model.BookResult{}
		lastErr error
		okWalks int
	)
	for _, sortBy := range sorts {
		wg.Add(1)
		go func(sortBy string) {
			defer wg.Done()
			found, err := c.walkAuthor(ctx, region, authorName, sortBy)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				lastErr = err
				return
			}
			okWalks++
			for _, r := range found {
				byASIN[r.ASIN] = r
			}
		}(sortBy)
	}
	wg.Wait()
	if okWalks == 0 {
		return nil, lastErr
	}
	var all []model.BookResult
	for _, r := range byASIN {
		if !creditedTo(r, authorASIN, authorName) {
			continue
		}
		if language != "" && r.Language != "" && r.Language != language {
			continue
		}
		// Full-cast dramatisations are separate products of books that are
		// also listed in their own right. They can still be added one by
		// one from book search.
		if dramatizedRe.MatchString(r.Title) {
			continue
		}
		all = append(all, r)
	}
	// Map iteration is random; fix the order so edition choice is stable.
	sort.Slice(all, func(i, j int) bool { return all[i].ASIN < all[j].ASIN })
	return dedupeEditions(all), nil
}

// SeriesBooks returns the books of a series in order, in the given language
// ("" for any), one entry per work: a series lists every recording and
// translation of each of its books.
func (c *Client) SeriesBooks(ctx context.Context, region, seriesASIN, seriesName, language string) ([]model.BookResult, error) {
	var out struct {
		Product struct {
			Relationships []struct {
				ASIN     string `json:"asin"`
				To       string `json:"relationship_to_product"`
				Type     string `json:"relationship_type"`
				Sequence string `json:"sequence"`
				Sort     string `json:"sort"`
			} `json:"relationships"`
		} `json:"product"`
	}
	u := c.audibleURL(region) + "/1.0/catalog/products/" + url.PathEscape(seriesASIN) + "?response_groups=relationships"
	if err := c.getJSON(ctx, u, &out); err != nil {
		return nil, err
	}
	type child struct {
		sequence string
		sort     float64
	}
	children := map[string]child{}
	var asins []string
	for _, r := range out.Product.Relationships {
		if r.To != "child" || r.Type != "series" || r.ASIN == "" {
			continue
		}
		if _, dup := children[r.ASIN]; dup {
			continue
		}
		n, _ := strconv.ParseFloat(r.Sort, 64)
		children[r.ASIN] = child{r.Sequence, n}
		asins = append(asins, r.ASIN)
	}

	var all []model.BookResult
	const batch = 50
	for i := 0; i < len(asins); i += batch {
		ps, _, err := c.products(ctx, region, url.Values{
			"asins":       {strings.Join(asins[i:min(i+batch, len(asins))], ",")},
			"num_results": {strconv.Itoa(batch)},
		})
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			if !p.isBook() {
				continue
			}
			r := p.toResult()
			if language != "" && r.Language != "" && r.Language != language {
				continue
			}
			if dramatizedRe.MatchString(r.Title) {
				continue
			}
			// The book may sit in several series; file it under this one.
			r.SeriesASIN, r.SeriesName, r.SeriesPosition = seriesASIN, seriesName, children[r.ASIN].sequence
			all = append(all, r)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ASIN < all[j].ASIN })
	all = dedupeEditions(all)
	// Re-recordings are often retitled ("Old Man's War: Book 1"), so within
	// a series the position identifies the work as well.
	byPos := map[string]int{}
	kept := all[:0]
	for _, r := range all {
		i, dup := byPos[r.SeriesPosition]
		switch {
		case r.SeriesPosition == "" || !dup:
			byPos[r.SeriesPosition] = len(kept)
			kept = append(kept, r)
		case betterEdition(r, kept[i]):
			kept[i] = r
		}
	}
	all = kept
	sort.SliceStable(all, func(i, j int) bool {
		a, b := children[all[i].ASIN].sort, children[all[j].ASIN].sort
		if a != b {
			return a < b
		}
		return all[i].ReleaseDate < all[j].ReleaseDate
	})
	return all, nil
}

var dramatizedRe = regexp.MustCompile(`(?i)\bdramati[sz]ed adaptation\b`)

func (c *Client) walkAuthor(ctx context.Context, region, authorName, sortBy string) ([]model.BookResult, error) {
	const pageSize = 50
	var all []model.BookResult
	for page := 1; page <= 20; page++ {
		ps, total, err := c.products(ctx, region, url.Values{
			"author":           {authorName},
			"num_results":      {strconv.Itoa(pageSize)},
			"page":             {strconv.Itoa(page)},
			"products_sort_by": {sortBy},
		})
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			if p.isBook() {
				all = append(all, p.toResult())
			}
		}
		if len(ps) < pageSize || page*pageSize >= total {
			break
		}
	}
	return all, nil
}

// creditedTo reports whether the book lists the author. The author search
// is by name, so it also returns namesakes; the ASIN tells them apart.
func creditedTo(r model.BookResult, asin, name string) bool {
	norm := textutil.Normalize(name)
	for _, a := range r.Authors {
		if a.ASIN != "" && asin != "" {
			if a.ASIN == asin {
				return true
			}
			continue
		}
		if textutil.Normalize(a.Name) == norm {
			return true
		}
	}
	return false
}

// dedupeEditions keeps one edition per title: unabridged over abridged, then
// the most-rated, which is nearly always the canonical recording.
func dedupeEditions(books []model.BookResult) []model.BookResult {
	best := map[string]model.BookResult{}
	var order []string
	for _, b := range books {
		key := textutil.Normalize(b.Title)
		cur, ok := best[key]
		if !ok {
			order = append(order, key)
			best[key] = b
			continue
		}
		if betterEdition(b, cur) {
			best[key] = b
		}
	}
	out := make([]model.BookResult, 0, len(order))
	for _, k := range order {
		out = append(out, best[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ReleaseDate > out[j].ReleaseDate })
	return out
}

func betterEdition(a, b model.BookResult) bool {
	if a.Abridged != b.Abridged {
		return !a.Abridged
	}
	if a.NumRatings != b.NumRatings {
		return a.NumRatings > b.NumRatings
	}
	return a.ReleaseDate > b.ReleaseDate
}

// ---------------------------------------------------------------- authors (Audnexus)

type audnexusAuthor struct {
	ASIN        string `json:"asin"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

// Author fetches an author's bio and photo.
func (c *Client) Author(ctx context.Context, region, asin string) (*model.AuthorResult, error) {
	var a audnexusAuthor
	u := c.audnexusURL() + "/authors/" + url.PathEscape(asin) + "?region=" + url.QueryEscape(region)
	if err := c.getJSON(ctx, u, &a); err != nil {
		return nil, err
	}
	return &model.AuthorResult{ASIN: a.ASIN, Name: a.Name, Description: a.Description, ImageURL: a.Image}, nil
}

// SearchAuthors finds authors by name. The search endpoint returns only names
// and ASINs (with duplicates), so the top hits are enriched concurrently;
// an author whose detail lookup fails is still returned, without a photo.
func (c *Client) SearchAuthors(ctx context.Context, region, query string) ([]model.AuthorResult, error) {
	var hits []audnexusAuthor
	u := c.audnexusURL() + "/authors?name=" + url.QueryEscape(query) + "&region=" + url.QueryEscape(region)
	if err := c.getJSON(ctx, u, &hits); err != nil {
		return nil, err
	}
	const max = 12
	out := []model.AuthorResult{}
	seen := map[string]bool{}
	for _, h := range hits {
		if h.ASIN == "" || seen[h.ASIN] {
			continue
		}
		seen[h.ASIN] = true
		out = append(out, model.AuthorResult{ASIN: h.ASIN, Name: h.Name})
		if len(out) == max {
			break
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := range out {
		wg.Add(1)
		go func(r *model.AuthorResult) {
			defer wg.Done()
			if full, err := c.Author(ctx, region, r.ASIN); err == nil {
				r.Description, r.ImageURL = full.Description, full.ImageURL
				if full.Name != "" {
					r.Name = full.Name
				}
			}
		}(&out[i])
	}
	wg.Wait()
	return out, nil
}
