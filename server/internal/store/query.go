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
	Headline      *string   `json:"headline"`
	Rationale     *string   `json:"rationale"`
	Confidence    *float32  `json:"confidence"`
	Rule          *string   `json:"rule"`
	Model         *string   `json:"model"`
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
		SELECT id, board, thread_no, post_no, post_time, detector, kind, matched_value, note, thread_subject, thread_replies, found_at,
		       headline, rationale, confidence, rule, model
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
			&f.Headline, &f.Rationale, &f.Confidence, &f.Rule, &f.Model,
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
// thread instance, how many instances that lineage has had, and the
// distinct finding kinds tied to its current thread (empty if none).
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
	FindingKinds  []string   `json:"findingKinds"`
}

// ListGenerals returns one row per general lineage tracked for board —
// the most recent thread instance in each (board, subject_key) group —
// most recently active first. FindingKinds only reflects findings tied
// to that current thread, not the lineage's earlier instances.
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
		),
		current AS (
			SELECT * FROM ranked WHERE rn = 1
		)
		SELECT
			c.board, c.subject_key, c.thread_no, c.thread_subject, c.replies, c.last_seen_at, c.ended_at,
			c.instance_count, c.lineage_first_seen_at,
			COALESCE(array_agg(DISTINCT f.kind) FILTER (WHERE f.kind IS NOT NULL), '{}')
		FROM current c
		LEFT JOIN findings f ON f.board = c.board AND f.thread_no = c.thread_no
		GROUP BY c.board, c.subject_key, c.thread_no, c.thread_subject, c.replies, c.last_seen_at, c.ended_at,
			c.instance_count, c.lineage_first_seen_at
		ORDER BY c.last_seen_at DESC
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
			&g.LastSeenAt, &g.EndedAt, &g.InstanceCount, &g.FirstSeenAt, &g.FindingKinds,
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

// KindCount is one finding kind's count within a SummaryWindow.
type KindCount struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// SummaryWindow is findings/generals activity within a trailing time
// window ending now — e.g. everything found in the last hour.
type SummaryWindow struct {
	Label         string      `json:"label"`
	TotalFindings int         `json:"totalFindings"`
	ByKind        []KindCount `json:"byKind"`
	NewGenerals   int         `json:"newGenerals"`
}

var summaryWindows = []struct {
	Label string
	Since time.Duration
}{
	{"Last hour", time.Hour},
	{"Last 24 hours", 24 * time.Hour},
	{"Last 7 days", 7 * 24 * time.Hour},
}

// Summary returns findings/generals activity for board (""=all boards)
// across three trailing windows: last hour, last 24 hours, last 7 days.
func (p *Postgres) Summary(ctx context.Context, board string) ([]SummaryWindow, error) {
	out := make([]SummaryWindow, 0, len(summaryWindows))
	for _, w := range summaryWindows {
		since := time.Now().Add(-w.Since)

		byKind, err := p.FindingKindCounts(ctx, board, since)
		if err != nil {
			return nil, err
		}

		total := 0
		for _, kc := range byKind {
			total += kc.Count
		}

		newGenerals, err := p.summaryNewGenerals(ctx, board, since)
		if err != nil {
			return nil, err
		}

		out = append(out, SummaryWindow{
			Label:         w.Label,
			TotalFindings: total,
			ByKind:        byKind,
			NewGenerals:   newGenerals,
		})
	}
	return out, nil
}

// FindingKindCounts returns findings counted by kind for board (""=all
// boards) since the given time. Also used directly by the narrative
// summarizer (server/cmd/summarizer), not just Summary.
func (p *Postgres) FindingKindCounts(ctx context.Context, board string, since time.Time) ([]KindCount, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT kind, count(*)
		FROM findings
		WHERE found_at >= $1 AND ($2 = '' OR board = $2)
		GROUP BY kind
		ORDER BY count(*) DESC
	`, since, board)
	if err != nil {
		return nil, fmt.Errorf("query finding kind counts: %w", err)
	}
	defer rows.Close()

	out := []KindCount{}
	for rows.Next() {
		var kc KindCount
		if err := rows.Scan(&kc.Kind, &kc.Count); err != nil {
			return nil, fmt.Errorf("scan kind count: %w", err)
		}
		out = append(out, kc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate kind counts: %w", err)
	}

	return out, nil
}

func (p *Postgres) summaryNewGenerals(ctx context.Context, board string, since time.Time) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM general_threads
		WHERE first_seen_at >= $1 AND ($2 = '' OR board = $2)
	`, since, board).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("query summary new generals: %w", err)
	}
	return count, nil
}

// FindingsSince returns the most recent findings (across every board)
// found at or after since, newest first, capped at limit. Used by the
// narrative summarizer to feed concrete examples into its prompt — see
// FindingKindCounts for the full-window breakdown that isn't capped.
func (p *Postgres) FindingsSince(ctx context.Context, since time.Time, limit int) ([]FindingRecord, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, board, thread_no, post_no, post_time, detector, kind, matched_value, note, thread_subject, thread_replies, found_at,
		       headline, rationale, confidence, rule, model
		FROM findings
		WHERE found_at >= $1
		ORDER BY found_at DESC, id DESC
		LIMIT $2
	`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("query findings since: %w", err)
	}
	defer rows.Close()

	out := []FindingRecord{}
	for rows.Next() {
		var f FindingRecord
		if err := rows.Scan(
			&f.ID, &f.Board, &f.ThreadNo, &f.PostNo, &f.PostTime,
			&f.Detector, &f.Kind, &f.MatchedValue, &f.Note, &f.ThreadSubject, &f.ThreadReplies, &f.FoundAt,
			&f.Headline, &f.Rationale, &f.Confidence, &f.Rule, &f.Model,
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

// GeneralActivity is one general thread's shape at the moment it
// started or ended, for narrative summary input.
type GeneralActivity struct {
	Board         string
	ThreadSubject string
	Replies       int
}

// NewGenerals returns general threads first seen at or after since.
func (p *Postgres) NewGenerals(ctx context.Context, since time.Time) ([]GeneralActivity, error) {
	return p.generalActivity(ctx, `
		SELECT board, thread_subject, replies FROM general_threads
		WHERE first_seen_at >= $1 ORDER BY first_seen_at DESC
	`, since)
}

// EndedGenerals returns general threads that ended (went gone from the
// catalog) at or after since.
func (p *Postgres) EndedGenerals(ctx context.Context, since time.Time) ([]GeneralActivity, error) {
	return p.generalActivity(ctx, `
		SELECT board, thread_subject, replies FROM general_threads
		WHERE ended_at >= $1 ORDER BY ended_at DESC
	`, since)
}

func (p *Postgres) generalActivity(ctx context.Context, query string, since time.Time) ([]GeneralActivity, error) {
	rows, err := p.pool.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("query general activity: %w", err)
	}
	defer rows.Close()

	out := []GeneralActivity{}
	for rows.Next() {
		var g GeneralActivity
		if err := rows.Scan(&g.Board, &g.ThreadSubject, &g.Replies); err != nil {
			return nil, fmt.Errorf("scan general activity: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate general activity: %w", err)
	}

	return out, nil
}

// NarrativeSummary is one generated prose summary of a trailing time
// window (window is "hour", "day", or "week").
type NarrativeSummary struct {
	Window       string    `json:"window"`
	PeriodStart  time.Time `json:"periodStart"`
	PeriodEnd    time.Time `json:"periodEnd"`
	FindingCount int       `json:"findingCount"`
	Summary      string    `json:"summary"`
	GeneratedAt  time.Time `json:"generatedAt"`
}

// narrativeWindowOrder is the canonical hour/day/week ordering for
// LatestNarrativeSummaries' response — not alphabetical, which "day" <
// "hour" < "week" would otherwise produce.
var narrativeWindowOrder = []string{"hour", "day", "week"}

// LatestNarrativeSummaries returns the most recently generated summary
// for each window that has at least one, in hour/day/week order.
func (p *Postgres) LatestNarrativeSummaries(ctx context.Context) ([]NarrativeSummary, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT DISTINCT ON (window_label) window_label, period_start, period_end, finding_count, summary, generated_at
		FROM narrative_summaries
		ORDER BY window_label, generated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query narrative summaries: %w", err)
	}
	defer rows.Close()

	byWindow := map[string]NarrativeSummary{}
	for rows.Next() {
		var s NarrativeSummary
		if err := rows.Scan(&s.Window, &s.PeriodStart, &s.PeriodEnd, &s.FindingCount, &s.Summary, &s.GeneratedAt); err != nil {
			return nil, fmt.Errorf("scan narrative summary: %w", err)
		}
		byWindow[s.Window] = s
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate narrative summaries: %w", err)
	}

	out := []NarrativeSummary{}
	for _, w := range narrativeWindowOrder {
		if s, ok := byWindow[w]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// SaveNarrativeSummary persists one generated summary. Append-only, like
// SavePollCycle — not part of libstore.Store since only the standalone
// summarizer binary calls this, never the poller.
func (p *Postgres) SaveNarrativeSummary(ctx context.Context, s NarrativeSummary) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO narrative_summaries (window_label, period_start, period_end, finding_count, summary)
		VALUES ($1, $2, $3, $4, $5)
	`, s.Window, s.PeriodStart, s.PeriodEnd, s.FindingCount, s.Summary)
	if err != nil {
		return fmt.Errorf("insert narrative_summary: %w", err)
	}
	return nil
}
