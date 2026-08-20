package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jcl80/dredge4us/lib/fourchan"
	"github.com/jcl80/dredge4us/server/internal/config"
)

// archiveLagThreshold: an archive-sourced board with no successful poll
// cycle in this long is flagged "archive lag" rather than judged on
// yield — a stale archive and a genuinely quiet board look the same in
// the numbers but call for different fixes.
const archiveLagThreshold = 2 * time.Hour

// lowYieldThreshold is findings per 1k posts read; below it a board is
// "low yield" regardless of how active it is.
const lowYieldThreshold = 1.0

// archiveSourcedBoards mirrors frontend/app/findings.ts's archiveBoards
// — kept in sync by hand, same as watchedBoards in boards.go, since api
// and poller share no config yet.
var archiveSourcedBoards = map[string]bool{
	"his":  true,
	"k":    true,
	"int":  true,
	"news": true,
}

// CoverageBoard is one watched board's row on the Coverage screen.
type CoverageBoard struct {
	Board           string  `json:"board"`
	Title           string  `json:"title"`
	YieldPer1k      float64 `json:"yieldPer1k"`
	FindingsPerWeek int     `json:"findingsPerWeek"`
	EstCostPerWeek  float64 `json:"estCostPerWeek"`
	Health          string  `json:"health"`
	LiveGenerals    int     `json:"liveGenerals"`
	TotalGenerals   int     `json:"totalGenerals"`
}

// CoverageResponse is /coverage's full payload.
type CoverageResponse struct {
	BoardsWatched int             `json:"boardsWatched"`
	BoardsServed  int             `json:"boardsServed"`
	Boards        []CoverageBoard `json:"boards"`
}

// coverageHandler replaces the old Boards page's N+1 (getGenerals +
// getKinds per board, times ~74 boards) with one store query plus the
// single 4chan board-index call every /boards/all request already made.
func coverageHandler(finder Finder, fc *fourchan.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allBoards, err := fc.FetchBoards(r.Context())
		if err != nil {
			slog.Error("coverage: fetch boards failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		titles := make(map[string]string, len(allBoards))
		for _, b := range allBoards {
			titles[b.Board] = b.Title
		}

		stats, err := finder.CoverageStats(r.Context(), watchedBoards)
		if err != nil {
			slog.Error("coverage: stats query failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		boards := make([]CoverageBoard, len(stats))
		for i, s := range stats {
			yield := 0.0
			if s.PostsSeenPerWeek > 0 {
				yield = float64(s.FindingsPerWeek) / float64(s.PostsSeenPerWeek) * 1000
			}
			boards[i] = CoverageBoard{
				Board:           s.Board,
				Title:           titles[s.Board],
				YieldPer1k:      yield,
				FindingsPerWeek: s.FindingsPerWeek,
				EstCostPerWeek:  float64(s.PostsSeenPerWeek) / 1000 * config.EstLLMCostPerThousandPostsUSD,
				Health:          boardHealth(s.Board, s.LastCycleAt, yield),
				LiveGenerals:    s.LiveGenerals,
				TotalGenerals:   s.TotalGenerals,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(CoverageResponse{
			BoardsWatched: len(watchedBoards),
			BoardsServed:  len(allBoards),
			Boards:        boards,
		}); err != nil {
			slog.Error("encode coverage response failed", "error", err)
		}
	}
}

// boardHealth classifies a watched board per README's Coverage spec:
// healthy, archive lag (archive-sourced, no recent successful cycle),
// or low yield. Archive lag is checked first — a stale archive source
// explains low yield on its own and calling it out separately points at
// the right fix.
func boardHealth(board string, lastCycleAt *time.Time, yieldPer1k float64) string {
	if archiveSourcedBoards[board] && (lastCycleAt == nil || time.Since(*lastCycleAt) > archiveLagThreshold) {
		return "archive lag"
	}
	if yieldPer1k < lowYieldThreshold {
		return "low yield"
	}
	return "healthy"
}
