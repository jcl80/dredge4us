// Package store declares the persistence boundary lib depends on but
// never implements. lib holds no state itself — see CLAUDE.md's module
// split — so this is only an interface; server provides the Postgres
// implementation.
package store

import (
	"context"
	"time"

	"github.com/jcl80/dredge4us/lib/detect"
)

// PollCycle mirrors one row of the poll_cycles table — coarse liveness
// stats for a single board's poll iteration.
type PollCycle struct {
	Board          string
	StartedAt      time.Time
	ThreadsSeen    int
	ThreadsNew     int
	ThreadsChanged int
	ThreadsGone    int
	Requests       int
	NotModified    int
	Errors         int
}

// Store is everything the poll loop needs persisted.
type Store interface {
	// LastModified returns the previously stored Last-Modified header for
	// url, and whether one was found.
	LastModified(ctx context.Context, url string) (value string, ok bool, err error)

	// SetLastModified persists the Last-Modified header returned for url
	// so restarts don't re-fetch unchanged resources.
	SetLastModified(ctx context.Context, url, lastModified string) error

	// SaveFindings persists detector output. Implementations must dedupe
	// on (board, post_no, kind, matched_value) so re-fetching a thread
	// doesn't duplicate rows.
	SaveFindings(ctx context.Context, findings []detect.Finding) error

	// SavePollCycle persists one cycle's stats.
	SavePollCycle(ctx context.Context, cycle PollCycle) error
}
