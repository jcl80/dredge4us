package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// watchedBoards is hardcoded for now — it should track POLLER_BOARDS, but
// api and poller are separate deployments with no shared config yet.
// Revisit once that needs to stay in sync automatically (e.g. by reading
// distinct boards out of the findings table instead).
var watchedBoards = []string{"g", "biz"}

func boardsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(watchedBoards); err != nil {
			slog.Error("encode boards response failed", "error", err)
		}
	}
}
