-- Optional detector rationale, distinct from matched_value (which stays
-- URL/hash only, per the original findings design). NULL for detectors
-- that don't have anything to say beyond the match itself.

ALTER TABLE findings ADD COLUMN IF NOT EXISTS note TEXT;
