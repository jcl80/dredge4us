-- posts_seen tracks how many posts a poll cycle actually fetched (across
-- every new/changed thread it read) — poll_cycles previously only
-- counted threads, which the Coverage screen's yield metric (findings
-- kept per thousand posts read) needs a real number for. Defaults to 0
-- and only starts being meaningful for cycles run after this deploys;
-- older rows have no way to backfill it.
ALTER TABLE poll_cycles ADD COLUMN IF NOT EXISTS posts_seen INT NOT NULL DEFAULT 0;
