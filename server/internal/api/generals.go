package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jcl80/dredge4us/server/internal/store"
)

func generalsHandler(finder Finder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		board := r.URL.Query().Get("board")
		if board == "" {
			http.Error(w, "board is required", http.StatusBadRequest)
			return
		}

		generals, err := finder.ListGenerals(r.Context(), board)
		if err != nil {
			slog.Error("list generals failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if generals == nil {
			generals = []store.GeneralLineage{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(generals); err != nil {
			slog.Error("encode generals response failed", "error", err)
		}
	}
}
