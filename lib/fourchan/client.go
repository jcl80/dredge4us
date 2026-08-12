package fourchan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const baseURL = "https://a.4cdn.org"

// ErrGone indicates the resource returned 404 — expected when a thread has
// been pruned or deleted, not a retryable failure.
var ErrGone = errors.New("fourchan: resource gone (404)")

// ErrNotModified indicates a 304 — the caller already has the latest data
// for the If-Modified-Since value it sent.
var ErrNotModified = errors.New("fourchan: not modified (304)")

// Client is a rate-limited client for the 4chan read API. Every request —
// catalog or thread, any board — goes through the same shared Limiter.
type Client struct {
	HTTPClient *http.Client
	Limiter    *Limiter
	BaseURL    string
}

// NewClient returns a Client backed by limiter. limiter must be shared
// across every Client the process creates; see Limiter's doc comment.
func NewClient(limiter *Limiter) *Client {
	return &Client{
		HTTPClient: http.DefaultClient,
		Limiter:    limiter,
		BaseURL:    baseURL,
	}
}

func (c *Client) get(ctx context.Context, url, ifModifiedSince string) (*http.Response, error) {
	if err := c.Limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if ifModifiedSince != "" {
		req.Header.Set("If-Modified-Since", ifModifiedSince)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	return resp, nil
}

// FetchCatalog fetches a board's catalog. ifModifiedSince should be the
// Last-Modified value stored from the previous successful fetch of this
// exact URL, or "" on first fetch. Returns ErrNotModified on 304 and
// ErrGone on 404 (an invalid or dead board).
func (c *Client) FetchCatalog(ctx context.Context, board, ifModifiedSince string) (Catalog, string, error) {
	url := fmt.Sprintf("%s/%s/catalog.json", c.BaseURL, board)
	resp, err := c.get(ctx, url, ifModifiedSince)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return nil, "", ErrNotModified
	case http.StatusNotFound:
		return nil, "", ErrGone
	case http.StatusOK:
	default:
		return nil, "", fmt.Errorf("fourchan: catalog %s: unexpected status %d", board, resp.StatusCode)
	}

	var catalog Catalog
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, "", fmt.Errorf("decode catalog: %w", err)
	}

	return catalog, resp.Header.Get("Last-Modified"), nil
}

// FetchThread fetches a single thread's posts. ifModifiedSince should be
// the Last-Modified value stored from the previous successful fetch of
// this exact URL, or "" on first fetch. Returns ErrNotModified on 304 and
// ErrGone on 404 (thread pruned or deleted — expected, not retryable).
func (c *Client) FetchThread(ctx context.Context, board string, threadNo int, ifModifiedSince string) ([]Post, string, error) {
	url := fmt.Sprintf("%s/%s/thread/%d.json", c.BaseURL, board, threadNo)
	resp, err := c.get(ctx, url, ifModifiedSince)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return nil, "", ErrNotModified
	case http.StatusNotFound:
		return nil, "", ErrGone
	case http.StatusOK:
	default:
		return nil, "", fmt.Errorf("fourchan: thread %s/%d: unexpected status %d", board, threadNo, resp.StatusCode)
	}

	var tr ThreadResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, "", fmt.Errorf("decode thread: %w", err)
	}

	return tr.Posts, resp.Header.Get("Last-Modified"), nil
}
