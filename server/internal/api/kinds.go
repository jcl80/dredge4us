package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// kindsHandler serves the distinct finding kinds present, optionally
// scoped to ?board=. Unlike boardsHandler this is a live query, not a
// hardcoded list — kinds span multiple detectors and grow over time.
func kindsHandler(finder Finder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kinds, err := finder.ListKinds(r.Context(), r.URL.Query().Get("board"))
		if err != nil {
			slog.Error("list kinds failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if kinds == nil {
			kinds = []string{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(kinds); err != nil {
			slog.Error("encode kinds response failed", "error", err)
		}
	}
}
