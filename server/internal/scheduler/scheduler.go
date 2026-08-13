// Package scheduler runs the poll loop: one ticking goroutine per watched
// board, funnelling thread fetches through a small worker pool that all
// share the client's single global rate limiter. This is deliberately an
// in-process queue, not the Postgres SKIP LOCKED queue that's the known
// eventual destination — Store is the seam that will let that swap in
// later without touching this package.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jcl80/dredge4us/lib/detect"
	"github.com/jcl80/dredge4us/lib/diff"
	"github.com/jcl80/dredge4us/lib/fourchan"
	"github.com/jcl80/dredge4us/lib/general"
	libstore "github.com/jcl80/dredge4us/lib/store"
	"github.com/jcl80/dredge4us/server/internal/config"
)

// Scheduler runs one poll loop per watched board.
type Scheduler struct {
	Client    *fourchan.Client
	Store     libstore.Store
	Detectors []detect.Detector
	Boards    []config.Board
	Workers   int
}

// Run starts one ticking goroutine per board and blocks until ctx is
// cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, b := range s.Boards {
		wg.Add(1)
		go func(b config.Board) {
			defer wg.Done()
			s.watchBoard(ctx, b)
		}(b)
	}
	wg.Wait()
}

// watchBoard runs cycles for a single board on its own ticker. Cycles run
// synchronously within this goroutine, so a slow cycle simply delays the
// next tick rather than overlapping with it.
func (s *Scheduler) watchBoard(ctx context.Context, b config.Board) {
	ticker := time.NewTicker(b.Interval)
	defer ticker.Stop()

	var prev map[int]fourchan.Thread
	prev = s.runCycle(ctx, b.Name, prev)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			prev = s.runCycle(ctx, b.Name, prev)
		}
	}
}

func catalogURL(board string) string {
	return fmt.Sprintf("https://a.4cdn.org/%s/catalog.json", board)
}

func threadURL(board string, no int) string {
	return fmt.Sprintf("https://a.4cdn.org/%s/thread/%d.json", board, no)
}

// runCycle fetches the board's catalog, diffs it against prev, fetches
// and scans whatever's new or changed, persists findings and cycle
// stats, and returns the new snapshot to become next cycle's prev.
func (s *Scheduler) runCycle(ctx context.Context, board string, prev map[int]fourchan.Thread) map[int]fourchan.Thread {
	stats := libstore.PollCycle{Board: board, StartedAt: time.Now()}

	url := catalogURL(board)
	lastMod, _, err := s.Store.LastModified(ctx, url)
	if err != nil {
		slog.Error("lookup catalog last-modified failed", "board", board, "error", err)
	}

	catalog, newLastMod, err := s.Client.FetchCatalog(ctx, board, lastMod)
	stats.Requests++

	switch {
	case errors.Is(err, fourchan.ErrNotModified):
		stats.NotModified++
		s.finishCycle(ctx, stats)
		return prev
	case errors.Is(err, fourchan.ErrGone):
		stats.Errors++
		slog.Error("board catalog gone", "board", board)
		s.finishCycle(ctx, stats)
		return prev
	case err != nil:
		stats.Errors++
		slog.Error("fetch catalog failed", "board", board, "error", err)
		s.finishCycle(ctx, stats)
		return prev
	}

	if err := s.Store.SetLastModified(ctx, url, newLastMod); err != nil {
		slog.Error("persist catalog last-modified failed", "board", board, "error", err)
	}

	curr := diff.Snapshot(catalog)
	change := diff.Compute(prev, curr)

	stats.ThreadsSeen = len(curr)
	stats.ThreadsNew = len(change.New)
	stats.ThreadsChanged = len(change.Changed)
	stats.ThreadsGone = len(change.Gone)

	s.trackGenerals(ctx, board, change, prev, stats.StartedAt)

	toFetch := make([]fourchan.Thread, 0, len(change.New)+len(change.Changed))
	toFetch = append(toFetch, change.New...)
	toFetch = append(toFetch, change.Changed...)

	findings := s.fetchAndDetect(ctx, board, toFetch, &stats)

	if err := s.Store.SaveFindings(ctx, findings); err != nil {
		slog.Error("save findings failed", "board", board, "error", err)
		stats.Errors++
	}

	s.finishCycle(ctx, stats)
	return curr
}

// trackGenerals updates general_threads for any new/changed thread
// whose subject matches the general heuristic, and marks tracked
// generals as ended once their thread goes gone. Runs on catalog data
// alone — no extra HTTP requests beyond the catalog fetch already done.
func (s *Scheduler) trackGenerals(ctx context.Context, board string, change diff.Change, prev map[int]fourchan.Thread, seenAt time.Time) {
	candidates := make([]fourchan.Thread, 0, len(change.New)+len(change.Changed))
	candidates = append(candidates, change.New...)
	candidates = append(candidates, change.Changed...)

	for _, t := range candidates {
		if !general.IsGeneral(t.Sub) {
			continue
		}
		g := libstore.GeneralThread{
			Board:         board,
			SubjectKey:    general.NormalizeSubject(t.Sub),
			ThreadNo:      t.No,
			ThreadSubject: t.Sub,
			Replies:       t.Replies,
			SeenAt:        seenAt,
		}
		if err := s.Store.UpsertGeneralThread(ctx, g); err != nil {
			slog.Error("upsert general thread failed", "board", board, "thread", t.No, "error", err)
		}
	}

	for _, no := range change.Gone {
		t, ok := prev[no]
		if !ok || !general.IsGeneral(t.Sub) {
			continue
		}
		if err := s.Store.EndGeneralThread(ctx, board, no, seenAt); err != nil {
			slog.Error("end general thread failed", "board", board, "thread", no, "error", err)
		}
	}
}

// fetchAndDetect fans toFetch out across the worker pool. Every fetch
// still goes through the client's single shared rate limiter — the pool
// only lets the scheduler avoid blocking on each fetch in turn.
func (s *Scheduler) fetchAndDetect(ctx context.Context, board string, toFetch []fourchan.Thread, stats *libstore.PollCycle) []detect.Finding {
	workers := s.Workers
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan fourchan.Thread)
	var mu sync.Mutex
	var findings []detect.Finding
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				f, req, notMod, errored := s.fetchOneThread(ctx, board, t)
				mu.Lock()
				stats.Requests += req
				stats.NotModified += notMod
				if errored {
					stats.Errors++
				}
				findings = append(findings, f...)
				mu.Unlock()
			}
		}()
	}

	for _, t := range toFetch {
		jobs <- t
	}
	close(jobs)
	wg.Wait()

	return findings
}

// fetchOneThread fetches and scans a single thread. Posts are held only
// on this call stack — they're discarded the moment Detect returns.
func (s *Scheduler) fetchOneThread(ctx context.Context, board string, t fourchan.Thread) (findings []detect.Finding, requests, notModified int, errored bool) {
	url := threadURL(board, t.No)

	lastMod, _, err := s.Store.LastModified(ctx, url)
	if err != nil {
		slog.Error("lookup thread last-modified failed", "board", board, "thread", t.No, "error", err)
	}

	posts, newLastMod, err := s.Client.FetchThread(ctx, board, t.No, lastMod)
	requests = 1

	switch {
	case errors.Is(err, fourchan.ErrNotModified):
		return nil, requests, 1, false
	case errors.Is(err, fourchan.ErrGone):
		slog.Info("thread gone", "board", board, "thread", t.No)
		return nil, requests, 0, false
	case err != nil:
		slog.Error("fetch thread failed", "board", board, "thread", t.No, "error", err)
		return nil, requests, 0, true
	}

	if err := s.Store.SetLastModified(ctx, url, newLastMod); err != nil {
		slog.Error("persist thread last-modified failed", "board", board, "thread", t.No, "error", err)
	}

	for _, d := range s.Detectors {
		findings = append(findings, d.Detect(board, t, posts)...)
	}

	return findings, requests, 0, false
}

func (s *Scheduler) finishCycle(ctx context.Context, stats libstore.PollCycle) {
	slog.Info("poll cycle",
		"board", stats.Board,
		"threads_seen", stats.ThreadsSeen,
		"threads_new", stats.ThreadsNew,
		"threads_changed", stats.ThreadsChanged,
		"threads_gone", stats.ThreadsGone,
		"requests", stats.Requests,
		"not_modified", stats.NotModified,
		"errors", stats.Errors,
	)
	if err := s.Store.SavePollCycle(ctx, stats); err != nil {
		slog.Error("save poll cycle failed", "board", stats.Board, "error", err)
	}
}
