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
	"sync/atomic"
	"time"

	"github.com/jcl80/dredge4us/lib/detect"
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

// FindingsSaver is the write surface the backfill handler needs —
// separate from Finder since every other route is read-only.
type FindingsSaver interface {
	SaveFindings(ctx context.Context, findings []detect.Finding) error
}

var _ FindingsSaver = (*store.Postgres)(nil)

// BackfillBoard is one board /debug/backfill pulls, and the archive
// client to pull it through. Client must be shared across every
// BackfillBoard on the same host — one Limiter per host, not per board,
// same rule as the scheduler's Sources. See docs/archive-sources.md.
type BackfillBoard struct {
	Board  string
	Client *foolfuuka.Client
}

// New builds the API's HTTP handler. fc serves the /boards/all board
// index — the only route that talks to 4chan directly instead of Store.
// backfillBoards/detectors feed /debug/backfill; pass nil/empty to
// disable it.
func New(finder Finder, saver FindingsSaver, fc *fourchan.Client, backfillBoards []BackfillBoard, detectors []detect.Detector) http.Handler {
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
	// TEMP: one-shot bulk pull via the archive search API (full board
	// history, not just the currently bumped catalog) for a quick data
	// volume burst. Remove once done.
	mux.HandleFunc("GET /debug/backfill", backfillHandler(saver, backfillBoards, detectors))
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

const (
	backfillDuration = 15 * time.Minute
	// backfillMaxPages caps pagination at the search API's own reachable
	// range for an unfiltered query (25 posts/page, ~5000 max — see
	// Meta.TotalFound in lib/foolfuuka). Requesting past it just returns
	// an empty page, so this is a safety cap, not a real limit hit in
	// practice.
	backfillMaxPages = 200
)

// backfillHandler triggers a one-shot pull of each board's recent
// history via the archive search API (not just the currently bumped
// catalog) and runs it through detectors, same as the live poller would
// — a burst for "get more data now", not something meant to run
// continuously. Runs in the background so the request returns
// immediately; a second call while one's in flight is rejected rather
// than running two at once against the same hosts.
func backfillHandler(saver FindingsSaver, boards []BackfillBoard, detectors []detect.Detector) http.HandlerFunc {
	var running atomic.Bool

	return func(w http.ResponseWriter, r *http.Request) {
		if len(boards) == 0 {
			http.Error(w, "no backfill boards configured", http.StatusNotImplemented)
			return
		}
		if !running.CompareAndSwap(false, true) {
			http.Error(w, "backfill already running", http.StatusConflict)
			return
		}

		go func() {
			defer running.Store(false)
			ctx, cancel := context.WithTimeout(context.Background(), backfillDuration)
			defer cancel()
			runBackfill(ctx, saver, boards, detectors)
		}()

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("backfill started — check logs for progress\n"))
	}
}

func runBackfill(ctx context.Context, saver FindingsSaver, boards []BackfillBoard, detectors []detect.Detector) {
	started := time.Now()
	for _, b := range boards {
		if ctx.Err() != nil {
			slog.Info("backfill deadline hit, stopping early", "remaining_boards", b.Board)
			break
		}
		backfillBoard(ctx, saver, b, detectors)
	}
	slog.Info("backfill complete", "elapsed", time.Since(started))
}

// backfillBoard pages through board's search results until the archive's
// own result cap, ctx's deadline, or an error stops it, groups posts by
// thread, runs detectors per thread (same shape the live poller uses —
// see scheduler.fetchOneThread), and saves whatever's found.
func backfillBoard(ctx context.Context, saver FindingsSaver, b BackfillBoard, detectors []detect.Detector) {
	threads := make(map[int][]fourchan.Post)
	threadSub := make(map[int]string)
	fetched := 0

	for page := 1; page <= backfillMaxPages; page++ {
		if ctx.Err() != nil {
			break
		}

		posts, totalFound, err := b.Client.Search(ctx, b.Board, page)
		if err != nil {
			slog.Error("backfill search failed", "board", b.Board, "page", page, "error", err)
			break
		}
		if len(posts) == 0 {
			break
		}

		for _, p := range posts {
			threadNo := p.Resto
			if threadNo == 0 {
				threadNo = p.No
			}
			threads[threadNo] = append(threads[threadNo], p)
			if p.Sub != "" {
				threadSub[threadNo] = p.Sub
			}
		}

		fetched += len(posts)
		if fetched >= totalFound {
			break
		}
	}

	var findings []detect.Finding
	for threadNo, posts := range threads {
		th := fourchan.Thread{No: threadNo, Sub: threadSub[threadNo], Replies: len(posts)}
		for _, d := range detectors {
			findings = append(findings, d.Detect(b.Board, th, posts)...)
		}
	}

	// Independent of ctx and its own short timeout: ctx is the overall
	// backfill deadline, and if it expired mid-page-loop above (the
	// common case for whichever board is running when time runs out),
	// reusing it here would drop that board's findings on the floor
	// instead of saving whatever was gathered before the cutoff.
	saveCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := saver.SaveFindings(saveCtx, findings); err != nil {
		slog.Error("backfill save findings failed", "board", b.Board, "error", err)
		return
	}
	slog.Info("backfill board done", "board", b.Board, "posts", fetched, "threads", len(threads), "findings", len(findings))
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
