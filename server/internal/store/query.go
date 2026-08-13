package store

import (
	"context"
	"fmt"
	"time"
)

// FindingRecord is a persisted findings row, as returned to the API. It's
// a separate type from detect.Finding because it carries DB-assigned
// fields (ID, FoundAt) that don't exist before a finding is saved.
type FindingRecord struct {
	ID            int64     `json:"id"`
	Board         string    `json:"board"`
	ThreadNo      int64     `json:"threadNo"`
	PostNo        int64     `json:"postNo"`
	PostTime      time.Time `json:"postTime"`
	Detector      string    `json:"detector"`
	Kind          string    `json:"kind"`
	MatchedValue  string    `json:"matchedValue"`
	Note          *string   `json:"note"`
	ThreadSubject string    `json:"threadSubject"`
	ThreadReplies int       `json:"threadReplies"`
	FoundAt       time.Time `json:"foundAt"`
}

// FindingsQuery filters ListFindings. Board and Kind are exact-match
// filters, ignored when empty. Limit is the caller's responsibility to
// clamp — this method does not enforce a maximum.
type FindingsQuery struct {
	Board string
	Kind  string
	Limit int
}

// ListFindings returns the most recent findings matching q, newest first.
func (p *Postgres) ListFindings(ctx context.Context, q FindingsQuery) ([]FindingRecord, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, board, thread_no, post_no, post_time, detector, kind, matched_value, note, thread_subject, thread_replies, found_at
		FROM findings
		WHERE ($1 = '' OR board = $1)
		  AND ($2 = '' OR kind = $2)
		ORDER BY found_at DESC, id DESC
		LIMIT $3
	`, q.Board, q.Kind, q.Limit)
	if err != nil {
		return nil, fmt.Errorf("query findings: %w", err)
	}
	defer rows.Close()

	var out []FindingRecord
	for rows.Next() {
		var f FindingRecord
		if err := rows.Scan(
			&f.ID, &f.Board, &f.ThreadNo, &f.PostNo, &f.PostTime,
			&f.Detector, &f.Kind, &f.MatchedValue, &f.Note, &f.ThreadSubject, &f.ThreadReplies, &f.FoundAt,
		); err != nil {
			return nil, fmt.Errorf("scan finding: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate findings: %w", err)
	}

	return out, nil
}

// ListKinds returns the distinct finding kinds present, optionally
// scoped to board ("" means all boards). Dynamic rather than hardcoded
// since kinds span multiple detectors and grow as detectors are added.
func (p *Postgres) ListKinds(ctx context.Context, board string) ([]string, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT DISTINCT kind FROM findings
		WHERE ($1 = '' OR board = $1)
		ORDER BY kind
	`, board)
	if err != nil {
		return nil, fmt.Errorf("query kinds: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("scan kind: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate kinds: %w", err)
	}

	return out, nil
}

// GeneralLineage is one general's current state: its latest known
// thread instance, plus how many instances that lineage has had.
type GeneralLineage struct {
	Board         string     `json:"board"`
	SubjectKey    string     `json:"subjectKey"`
	ThreadNo      int64      `json:"threadNo"`
	ThreadSubject string     `json:"threadSubject"`
	Replies       int        `json:"replies"`
	LastSeenAt    time.Time  `json:"lastSeenAt"`
	EndedAt       *time.Time `json:"endedAt"`
	InstanceCount int        `json:"instanceCount"`
	FirstSeenAt   time.Time  `json:"firstSeenAt"`
}

// ListGenerals returns one row per general lineage tracked for board —
// the most recent thread instance in each (board, subject_key) group —
// most recently active first.
func (p *Postgres) ListGenerals(ctx context.Context, board string) ([]GeneralLineage, error) {
	rows, err := p.pool.Query(ctx, `
		WITH ranked AS (
			SELECT
				board, subject_key, thread_no, thread_subject, replies, last_seen_at, ended_at,
				ROW_NUMBER() OVER (PARTITION BY subject_key ORDER BY first_seen_at DESC) AS rn,
				COUNT(*) OVER (PARTITION BY subject_key) AS instance_count,
				MIN(first_seen_at) OVER (PARTITION BY subject_key) AS lineage_first_seen_at
			FROM general_threads
			WHERE board = $1
		)
		SELECT board, subject_key, thread_no, thread_subject, replies, last_seen_at, ended_at, instance_count, lineage_first_seen_at
		FROM ranked
		WHERE rn = 1
		ORDER BY last_seen_at DESC
	`, board)
	if err != nil {
		return nil, fmt.Errorf("query general_threads: %w", err)
	}
	defer rows.Close()

	var out []GeneralLineage
	for rows.Next() {
		var g GeneralLineage
		if err := rows.Scan(
			&g.Board, &g.SubjectKey, &g.ThreadNo, &g.ThreadSubject, &g.Replies,
			&g.LastSeenAt, &g.EndedAt, &g.InstanceCount, &g.FirstSeenAt,
		); err != nil {
			return nil, fmt.Errorf("scan general_thread: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate general_threads: %w", err)
	}

	return out, nil
}
