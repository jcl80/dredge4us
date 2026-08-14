package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jcl80/dredge4us/lib/fourchan"
)

// watchedBoards is hardcoded for now — it should track POLLER_BOARDS, but
// api and poller are separate deployments with no shared config yet.
// Revisit once that needs to stay in sync automatically (e.g. by reading
// distinct boards out of the findings table instead). Keep this in sync
// by hand with the poller worker's POLLER_BOARDS env var (.do/app.yaml
// documents the intended value; the live value is set in the DO
// console — see that file's header comment).
var watchedBoards = []string{"g", "biz", "his", "k", "int", "news"}

func isWatched(board string) bool {
	for _, b := range watchedBoards {
		if b == board {
			return true
		}
	}
	return false
}

func boardsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(watchedBoards); err != nil {
			slog.Error("encode boards response failed", "error", err)
		}
	}
}

// BoardStatus is one entry in the full board index, tagged with whether
// this poller currently watches it.
type BoardStatus struct {
	Board   string `json:"board"`
	Title   string `json:"title"`
	Watched bool   `json:"watched"`
}

// allBoardsHandler serves every board 4chan currently has, each tagged
// watched/unwatched — unlike boardsHandler, which only lists the ones
// this poller watches.
func allBoardsHandler(fc *fourchan.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		boards, err := fc.FetchBoards(r.Context())
		if err != nil {
			slog.Error("fetch boards failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		out := make([]BoardStatus, len(boards))
		for i, b := range boards {
			out[i] = BoardStatus{Board: b.Board, Title: b.Title, Watched: isWatched(b.Board)}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(out); err != nil {
			slog.Error("encode all-boards response failed", "error", err)
		}
	}
}
