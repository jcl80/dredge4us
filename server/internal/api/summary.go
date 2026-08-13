package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jcl80/dredge4us/server/internal/store"
)

func summaryHandler(finder Finder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		board := r.URL.Query().Get("board")

		windows, err := finder.Summary(r.Context(), board)
		if err != nil {
			slog.Error("summary failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if windows == nil {
			windows = []store.SummaryWindow{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(windows); err != nil {
			slog.Error("encode summary response failed", "error", err)
		}
	}
}

// narrativeSummaryHandler serves the latest generated prose summary per
// window (hour/day/week) — see server/cmd/summarizer, the only writer
// of narrative_summaries.
func narrativeSummaryHandler(finder Finder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summaries, err := finder.LatestNarrativeSummaries(r.Context())
		if err != nil {
			slog.Error("narrative summary failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if summaries == nil {
			summaries = []store.NarrativeSummary{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summaries); err != nil {
			slog.Error("encode narrative summary response failed", "error", err)
		}
	}
}
