-- One row per physical thread instance recognized as a "general" (see
-- lib/general). A lineage is all rows sharing (board, subject_key),
-- ordered by first_seen_at — successive reposts of the same general
-- stitch together via that shared key.

CREATE TABLE IF NOT EXISTS general_threads (
    id              BIGSERIAL PRIMARY KEY,
    board           TEXT NOT NULL,
    subject_key     TEXT NOT NULL,
    thread_no       BIGINT NOT NULL,
    thread_subject  TEXT NOT NULL,
    replies         INT NOT NULL,
    first_seen_at   TIMESTAMPTZ NOT NULL,
    last_seen_at    TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ,
    UNIQUE (board, thread_no)
);

CREATE INDEX IF NOT EXISTS general_threads_board_subject_idx ON general_threads (board, subject_key);
