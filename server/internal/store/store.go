// Package store is the Postgres implementation of lib/store.Store — the
// only place in this repo allowed to import a Postgres driver, per
// CLAUDE.md's module boundary.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jcl80/dredge4us/lib/detect"
	libstore "github.com/jcl80/dredge4us/lib/store"
)

// Postgres implements libstore.Store.
type Postgres struct {
	pool *pgxpool.Pool
}

var _ libstore.Store = (*Postgres)(nil)

// NewPostgres connects to databaseURL and returns a ready Postgres store.
func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

// Pool exposes the underlying pool for the migration runner.
func (p *Postgres) Pool() *pgxpool.Pool { return p.pool }

// Close releases the connection pool.
func (p *Postgres) Close() { p.pool.Close() }

// LastModified implements libstore.Store.
func (p *Postgres) LastModified(ctx context.Context, url string) (string, bool, error) {
	var value string
	err := p.pool.QueryRow(ctx,
		`SELECT last_modified FROM fetch_state WHERE url = $1`, url,
	).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("query fetch_state: %w", err)
	}
	return value, true, nil
}

// SetLastModified implements libstore.Store.
func (p *Postgres) SetLastModified(ctx context.Context, url, lastModified string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO fetch_state (url, last_modified, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (url) DO UPDATE SET last_modified = $2, updated_at = now()
	`, url, lastModified)
	if err != nil {
		return fmt.Errorf("upsert fetch_state: %w", err)
	}
	return nil
}

// SaveFindings implements libstore.Store. Dupe findings — same board,
// post, kind, and matched value from a re-fetched thread — are silently
// skipped rather than erroring.
func (p *Postgres) SaveFindings(ctx context.Context, findings []detect.Finding) error {
	if len(findings) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, f := range findings {
		batch.Queue(`
			INSERT INTO findings
				(board, thread_no, post_no, post_time, detector, kind, matched_value, thread_subject, thread_replies)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (board, post_no, kind, matched_value) DO NOTHING
		`, f.Board, f.ThreadNo, f.PostNo, f.PostTime, f.Detector, f.Kind, f.MatchedValue, f.ThreadSubject, f.ThreadReplies)
	}

	br := p.pool.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()

	for range findings {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert finding: %w", err)
		}
	}
	return nil
}

// SavePollCycle implements libstore.Store.
func (p *Postgres) SavePollCycle(ctx context.Context, c libstore.PollCycle) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO poll_cycles
			(board, started_at, threads_seen, threads_new, threads_changed, threads_gone, requests, not_modified, errors)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.Board, c.StartedAt, c.ThreadsSeen, c.ThreadsNew, c.ThreadsChanged, c.ThreadsGone, c.Requests, c.NotModified, c.Errors)
	if err != nil {
		return fmt.Errorf("insert poll_cycle: %w", err)
	}
	return nil
}
