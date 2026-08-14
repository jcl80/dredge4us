// Package api implements the poller's read-only HTTP API: a single
// endpoint serving recent findings as JSON. Assumed caller is a trusted
// server-side process (e.g. a Next.js route handler), not a browser
// directly — there's no CORS or auth here yet.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jcl80/dredge4us/lib/foolfuuka"
	"github.com/jcl80/dredge4us/lib/fourchan"
	"github.com/jcl80/dredge4us/server/internal/store"
)

const (
	defaultLimit = 100
	maxLimit     = 500
)

// Finder is the read surface this package needs from storage.
type Finder interface {
	ListFindings(ctx context.Context, q store.FindingsQuery) ([]store.FindingRecord, error)
	ListGenerals(ctx context.Context, board string) ([]store.GeneralLineage, error)
	ListKinds(ctx context.Context, board string) ([]string, error)
	Summary(ctx context.Context, board string) ([]store.SummaryWindow, error)
	LatestNarrativeSummaries(ctx context.Context) ([]store.NarrativeSummary, error)
}

var _ Finder = (*store.Postgres)(nil)

// New builds the API's HTTP handler. fc serves the /boards/all board
// index — the only route that talks to 4chan directly instead of Store.
func New(finder Finder, fc *fourchan.Client) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /findings", findingsHandler(finder))
	mux.HandleFunc("GET /boards", boardsHandler())
	mux.HandleFunc("GET /boards/all", allBoardsHandler(fc))
	mux.HandleFunc("GET /generals", generalsHandler(finder))
	mux.HandleFunc("GET /kinds", kindsHandler(finder))
	mux.HandleFunc("GET /summary", summaryHandler(finder))
	mux.HandleFunc("GET /summary/narrative", narrativeSummaryHandler(finder))
	// TEMP: verifies desuarchive/palanq are reachable (not Cloudflare-
	// challenged) from this app's real DO egress, vs. the phone hotspot
	// docs/archive-sources.md was drafted against. Remove once checked.
	mux.HandleFunc("GET /debug/archive-check", archiveCheckHandler())
	return withLogging(mux)
}

func archiveCheckHandler() http.HandlerFunc {
	// board must be one each archive actually carries — an unmapped
	// board 422s at the FoolFuuka app itself, which looks like a
	// failure here but has nothing to do with Cloudflare reachability.
	hosts := []struct{ base, board string }{
		{"https://desuarchive.org", "g"},
		{"https://archive.palanq.win", "news"},
		{"https://archive.4plebs.org", "pol"},
		{"https://archived.moe", "biz"},
	}
	return func(w http.ResponseWriter, r *http.Request) {
		type result struct {
			Host        string `json:"host"`
			Status      int    `json:"status"`
			CFMitigated string `json:"cf_mitigated,omitempty"`
			Error       string `json:"error,omitempty"`
		}

		results := make([]result, 0, len(hosts))
		for _, h := range hosts {
			res := result{Host: h.base}

			req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
				h.base+"/_/api/chan/index/?board="+h.board+"&page=1", nil)
			if err != nil {
				res.Error = err.Error()
				results = append(results, res)
				continue
			}
			req.Header.Set("User-Agent", foolfuuka.DefaultUserAgent)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				res.Error = err.Error()
				results = append(results, res)
				continue
			}
			res.Status = resp.StatusCode
			res.CFMitigated = resp.Header.Get("cf-mitigated")
			_ = resp.Body.Close()

			results = append(results, res)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(results)
	}
}

func findingsHandler(finder Finder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := store.FindingsQuery{
			Board: r.URL.Query().Get("board"),
			Kind:  r.URL.Query().Get("kind"),
			Limit: parseLimit(r.URL.Query().Get("limit")),
		}

		findings, err := finder.ListFindings(r.Context(), q)
		if err != nil {
			slog.Error("list findings failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if findings == nil {
			findings = []store.FindingRecord{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(findings); err != nil {
			slog.Error("encode findings response failed", "error", err)
		}
	}
}

func parseLimit(raw string) int {
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
	})
}
