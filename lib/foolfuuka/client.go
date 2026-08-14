package foolfuuka

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/jcl80/dredge4us/lib/fourchan"
)

// DefaultCatalogPages is how many pages of the board index FetchCatalog
// walks per call. FoolFuuka's index has no end-of-catalog signal — deep
// pages keep returning old threads indefinitely instead of emptying out
// — so this is a deliberate cap, not a detected boundary. New/changed
// threads worth scanning are bumped-recent, which is what these leading
// pages hold; older activity isn't what live monitoring needs anyway.
const DefaultCatalogPages = 3

// DefaultUserAgent identifies this project and a contact address on every
// request. This isn't optional politeness: these archives sit behind
// Cloudflare, and an unidentified client (bare Go/curl UA) gets served a
// bot challenge page where a descriptive one gets normal responses. See
// docs/archive-sources.md.
const DefaultUserAgent = "dredge4us/0.1 (+mailto:jcambrac@gmail.com; 4chan findings monitor)"

// Client is a rate-limited client for one FoolFuuka archive host. Every
// request goes through the shared Limiter passed to NewClient — as with
// fourchan.Client, callers must not construct more than one Limiter per
// host across a process.
type Client struct {
	HTTPClient   *http.Client
	Limiter      *fourchan.Limiter
	BaseURL      string
	UserAgent    string
	CatalogPages int
}

// NewClient returns a Client for the archive at baseURL (e.g.
// "https://desuarchive.org"), backed by limiter.
func NewClient(baseURL string, limiter *fourchan.Limiter) *Client {
	return &Client{
		HTTPClient:   http.DefaultClient,
		Limiter:      limiter,
		BaseURL:      baseURL,
		UserAgent:    DefaultUserAgent,
		CatalogPages: DefaultCatalogPages,
	}
}

func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	if err := c.Limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	return resp, nil
}

// FetchCatalog fetches a board's leading index pages (see
// DefaultCatalogPages) and translates them into a fourchan.Catalog, so
// diff.Snapshot/diff.Compute work unchanged against archive-sourced
// boards. ifModifiedSince is accepted for interface parity with
// fourchan.Client but is not sent: this endpoint doesn't honor
// conditional GET (verified against desuarchive — see
// docs/archive-sources.md), so every call returns fresh data. The
// returned string is always "".
func (c *Client) FetchCatalog(ctx context.Context, board, ifModifiedSince string) (fourchan.Catalog, string, error) {
	pages := c.CatalogPages
	if pages < 1 {
		pages = DefaultCatalogPages
	}

	catalog := make(fourchan.Catalog, 0, pages)
	for p := 1; p <= pages; p++ {
		url := fmt.Sprintf("%s/_/api/chan/index/?board=%s&page=%d", c.BaseURL, board, p)
		resp, err := c.get(ctx, url)
		if err != nil {
			return nil, "", err
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, "", fmt.Errorf("foolfuuka: index %s page %d: unexpected status %d", board, p, resp.StatusCode)
		}

		var idx apiIndexResponse
		err = json.NewDecoder(resp.Body).Decode(&idx)
		_ = resp.Body.Close()
		if err != nil {
			return nil, "", fmt.Errorf("decode index: %w", err)
		}

		if len(idx) == 0 {
			break
		}

		threads, err := convertIndex(idx)
		if err != nil {
			return nil, "", err
		}
		catalog = append(catalog, fourchan.Page{Page: p, Threads: threads})
	}

	return catalog, "", nil
}

// FetchThread fetches a single thread's posts and translates them into
// []fourchan.Post, ordered like 4chan's own thread endpoint (OP first,
// then replies in ascending post order). ifModifiedSince is accepted for
// interface parity with fourchan.Client but is not sent, for the same
// reason as FetchCatalog. Returns fourchan.ErrGone for a thread that's
// been pruned — FoolFuuka signals this as HTTP 200 with an {"error":...}
// body rather than a 404, so that shape is checked explicitly.
func (c *Client) FetchThread(ctx context.Context, board string, threadNo int, ifModifiedSince string) ([]fourchan.Post, string, error) {
	url := fmt.Sprintf("%s/_/api/chan/thread/?board=%s&num=%d", c.BaseURL, board, threadNo)
	resp, err := c.get(ctx, url)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("foolfuuka: thread %s/%d: unexpected status %d", board, threadNo, resp.StatusCode)
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, "", fmt.Errorf("decode thread: %w", err)
	}
	if _, isError := raw["error"]; isError {
		return nil, "", fourchan.ErrGone
	}

	var entry apiThreadEntry
	for _, v := range raw {
		if err := json.Unmarshal(v, &entry); err != nil {
			return nil, "", fmt.Errorf("decode thread entry: %w", err)
		}
		break
	}

	posts, err := convertThread(entry)
	if err != nil {
		return nil, "", err
	}

	return posts, resp.Header.Get("Last-Modified"), nil
}

// Search fetches one page (25 posts) of a board's search results,
// newest-first, board-wide — not limited to currently bumped threads
// the way FetchCatalog is. Unfiltered search caps at 5000 total results
// (per Meta.TotalFound's max_results, itself returned by the API), so
// page 1..200 covers the reachable range; requesting beyond that returns
// an empty page rather than erroring. Returns the page's posts translated
// to fourchan.Post (grouped by ThreadNo via Post.Resto, same as any
// other posts slice) and the archive's total matching post count.
func (c *Client) Search(ctx context.Context, board string, page int) ([]fourchan.Post, int, error) {
	url := fmt.Sprintf("%s/_/api/chan/search/?boards=%s&page=%d", c.BaseURL, board, page)
	resp, err := c.get(ctx, url)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("foolfuuka: search %s page %d: unexpected status %d", board, page, resp.StatusCode)
	}

	var sr apiSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, 0, fmt.Errorf("decode search: %w", err)
	}

	posts := make([]fourchan.Post, 0, len(sr.Page0.Posts))
	for _, p := range sr.Page0.Posts {
		post, err := convertPost(p)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, post)
	}

	return posts, sr.Meta.TotalFound, nil
}

func convertIndex(idx apiIndexResponse) ([]fourchan.Thread, error) {
	threads := make([]fourchan.Thread, 0, len(idx))
	for _, entry := range idx {
		t, err := convertOP(entry.Op)
		if err != nil {
			return nil, err
		}

		t.Replies = entry.Omitted + len(entry.Posts)

		lastModified := t.Time
		for _, p := range entry.Posts {
			if p.Timestamp > lastModified {
				lastModified = p.Timestamp
			}
		}
		t.LastModified = lastModified

		threads = append(threads, t)
	}
	return threads, nil
}

func convertOP(op apiPost) (fourchan.Thread, error) {
	no, err := strconv.Atoi(op.Num)
	if err != nil {
		return fourchan.Thread{}, fmt.Errorf("foolfuuka: post num %q: %w", op.Num, err)
	}

	return fourchan.Thread{
		No:       no,
		Sub:      op.Title,
		Com:      op.Comment,
		Time:     op.Timestamp,
		Sticky:   parseFlag(op.Sticky),
		Closed:   parseFlag(op.Locked),
		Archived: parseFlag(op.TimestampExpired),
	}, nil
}

func convertThread(entry apiThreadEntry) ([]fourchan.Post, error) {
	all := make([]apiPost, 0, len(entry.Posts)+1)
	all = append(all, entry.Op)
	for _, p := range entry.Posts {
		all = append(all, p)
	}

	posts := make([]fourchan.Post, 0, len(all))
	for _, p := range all {
		post, err := convertPost(p)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	// Sort numerically (post numbers are strings on the wire, so a
	// string sort would misorder once digit counts differ) so OP lands
	// first and replies follow in post order, matching 4chan's own
	// thread endpoint.
	sort.Slice(posts, func(i, j int) bool { return posts[i].No < posts[j].No })

	return posts, nil
}

func convertPost(p apiPost) (fourchan.Post, error) {
	no, err := strconv.Atoi(p.Num)
	if err != nil {
		return fourchan.Post{}, fmt.Errorf("foolfuuka: post num %q: %w", p.Num, err)
	}

	resto := 0
	if parseFlag(p.IsOP) == 0 {
		resto, err = strconv.Atoi(p.ThreadNum)
		if err != nil {
			return fourchan.Post{}, fmt.Errorf("foolfuuka: thread_num %q: %w", p.ThreadNum, err)
		}
	}

	return fourchan.Post{
		No:       no,
		Resto:    resto,
		Time:     p.Timestamp,
		Sub:      p.Title,
		Com:      p.Comment,
		Sticky:   parseFlag(p.Sticky),
		Closed:   parseFlag(p.Locked),
		Archived: parseFlag(p.TimestampExpired),
	}, nil
}

func parseFlag(s string) int {
	if s == "1" {
		return 1
	}
	return 0
}
