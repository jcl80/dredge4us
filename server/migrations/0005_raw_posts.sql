-- raw_posts holds full post text pulled by the archive backfill —
-- explicitly reversing findings' "no post text" boundary (see
-- 0001_init.sql's header) so classification can be re-run later,
-- against detectors added since, without re-fetching from the archive.
-- classified_at tracks whether a classify pass has processed a row yet;
-- NULL means it hasn't.
CREATE TABLE IF NOT EXISTS raw_posts (
    id             BIGSERIAL PRIMARY KEY,
    board          TEXT NOT NULL,
    source         TEXT NOT NULL,
    thread_no      BIGINT NOT NULL,
    post_no        BIGINT NOT NULL,
    post_time      TIMESTAMPTZ NOT NULL,
    sub            TEXT,
    com            TEXT NOT NULL,
    sticky         BOOLEAN NOT NULL DEFAULT false,
    closed         BOOLEAN NOT NULL DEFAULT false,
    archived       BOOLEAN NOT NULL DEFAULT false,
    fetched_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    classified_at  TIMESTAMPTZ,
    UNIQUE (board, post_no)
);

CREATE INDEX IF NOT EXISTS raw_posts_unclassified_idx
    ON raw_posts (board, thread_no) WHERE classified_at IS NULL;
