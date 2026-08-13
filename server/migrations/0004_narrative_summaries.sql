-- One row per generated LLM narrative summary of a trailing time
-- window (hour/day/week). Append-only, like poll_cycles — the API
-- reads the most recent row per window, older rows are just history.

-- window_label, not window — WINDOW is a reserved word in Postgres.
CREATE TABLE IF NOT EXISTS narrative_summaries (
    id             BIGSERIAL PRIMARY KEY,
    window_label   TEXT NOT NULL,
    period_start   TIMESTAMPTZ NOT NULL,
    period_end     TIMESTAMPTZ NOT NULL,
    finding_count  INT NOT NULL,
    summary        TEXT NOT NULL,
    generated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS narrative_summaries_window_generated_at_idx
    ON narrative_summaries (window_label, generated_at DESC);
