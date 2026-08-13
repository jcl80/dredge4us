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

	"github.com/jcl80/dredge4us/server/internal/store"
)

const (
	defaultLimit = 100
	maxLimit     = 500
)

// Finder is the read surface this package needs from storage.
type Finder interface {
	ListFindings(ctx context.Context, q store.FindingsQuery) ([]store.FindingRecord, error)
}

var _ Finder = (*store.Postgres)(nil)

// New builds the API's HTTP handler.
func New(finder Finder) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /findings", findingsHandler(finder))
	mux.HandleFunc("GET /boards", boardsHandler())
	return withLogging(mux)
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
