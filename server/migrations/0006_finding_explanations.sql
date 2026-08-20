-- Explanation/provenance fields for findings, populated by the LLM
-- classifier (lib/detect/llm.go, added in a later commit — see
-- design_handoff_dredge4us_ui/COMMIT_PLAN.md commit 4). Nullable: rows
-- written before that commit, and regex-only findings, have none of
-- these; the UI falls back to matched_value/note.
ALTER TABLE findings ADD COLUMN IF NOT EXISTS headline TEXT;
ALTER TABLE findings ADD COLUMN IF NOT EXISTS rationale TEXT;
ALTER TABLE findings ADD COLUMN IF NOT EXISTS confidence REAL;
ALTER TABLE findings ADD COLUMN IF NOT EXISTS rule TEXT;
ALTER TABLE findings ADD COLUMN IF NOT EXISTS model TEXT;
