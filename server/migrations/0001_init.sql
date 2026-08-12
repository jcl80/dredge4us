-- Three tables total. findings holds detector output only — no post
-- text, no plaintext, nothing an image URL could be built from.
-- poll_cycles is liveness observability. fetch_state is the
-- If-Modified-Since cache keyed by exact request URL.

CREATE TABLE findings (
    id              BIGSERIAL PRIMARY KEY,
    board           TEXT NOT NULL,
    thread_no       BIGINT NOT NULL,
    post_no         BIGINT NOT NULL,
    post_time       TIMESTAMPTZ NOT NULL,
    detector        TEXT NOT NULL,
    kind            TEXT NOT NULL,
    matched_value   TEXT NOT NULL,
    thread_subject  TEXT,
    thread_replies  INT NOT NULL,
    found_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (board, post_no, kind, matched_value)
);

CREATE INDEX findings_board_found_at_idx ON findings (board, found_at DESC);

CREATE TABLE poll_cycles (
    id               BIGSERIAL PRIMARY KEY,
    board            TEXT NOT NULL,
    started_at       TIMESTAMPTZ NOT NULL,
    threads_seen     INT NOT NULL,
    threads_new      INT NOT NULL,
    threads_changed  INT NOT NULL,
    threads_gone     INT NOT NULL,
    requests         INT NOT NULL,
    not_modified     INT NOT NULL,
    errors           INT NOT NULL
);

CREATE INDEX poll_cycles_board_started_at_idx ON poll_cycles (board, started_at DESC);

CREATE TABLE fetch_state (
    url            TEXT PRIMARY KEY,
    last_modified  TEXT NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
