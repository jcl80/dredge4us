package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// RawPost is one post's full text, as pulled by a backfill — the
// findings table's opposite: everything, not just what a detector
// flagged. See migrations/0005_raw_posts.sql for why this table exists
// at all. ID is unset on insert (DB-assigned); Source records which
// archive (or "live") it came from, informational only.
type RawPost struct {
	ID       int64
	Board    string
	Source   string
	ThreadNo int
	PostNo   int
	PostTime time.Time
	Sub      string
	Com      string
	Sticky   bool
	Closed   bool
	Archived bool
}

// SaveRawPosts inserts posts, skipping any (board, post_no) already
// stored — a re-run of the same backfill is a no-op past the first time.
func (p *Postgres) SaveRawPosts(ctx context.Context, posts []RawPost) error {
	if len(posts) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, rp := range posts {
		batch.Queue(`
			INSERT INTO raw_posts
				(board, source, thread_no, post_no, post_time, sub, com, sticky, closed, archived)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (board, post_no) DO NOTHING
		`, rp.Board, rp.Source, rp.ThreadNo, rp.PostNo, rp.PostTime, nullableText(rp.Sub), rp.Com, rp.Sticky, rp.Closed, rp.Archived)
	}

	br := p.pool.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()

	for range posts {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert raw_post: %w", err)
		}
	}
	return nil
}

// UnclassifiedRawPosts returns every raw post no classify pass has
// touched yet, ordered so callers can group consecutive rows into
// threads without a second pass. board filters to one board, or every
// board when "".
func (p *Postgres) UnclassifiedRawPosts(ctx context.Context, board string) ([]RawPost, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, board, source, thread_no, post_no, post_time, sub, com, sticky, closed, archived
		FROM raw_posts
		WHERE classified_at IS NULL AND ($1 = '' OR board = $1)
		ORDER BY board, thread_no, post_no
	`, board)
	if err != nil {
		return nil, fmt.Errorf("query raw_posts: %w", err)
	}
	defer rows.Close()

	var posts []RawPost
	for rows.Next() {
		var rp RawPost
		var sub *string
		if err := rows.Scan(&rp.ID, &rp.Board, &rp.Source, &rp.ThreadNo, &rp.PostNo, &rp.PostTime,
			&sub, &rp.Com, &rp.Sticky, &rp.Closed, &rp.Archived); err != nil {
			return nil, fmt.Errorf("scan raw_post: %w", err)
		}
		if sub != nil {
			rp.Sub = *sub
		}
		posts = append(posts, rp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate raw_posts: %w", err)
	}
	return posts, nil
}

// MarkClassified stamps classified_at on every id in ids, so a later
// classify pass doesn't reprocess them.
func (p *Postgres) MarkClassified(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := p.pool.Exec(ctx, `UPDATE raw_posts SET classified_at = now() WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("mark raw_posts classified: %w", err)
	}
	return nil
}
