-- 0010: transcript segment full-text search (library mode search).
-- Generated tsvector column + GIN index; queries use websearch_to_tsquery,
-- ts_headline (StartSel=<b>, StopSel=</b>) and ts_rank.

ALTER TABLE transcript_segments
    ADD COLUMN IF NOT EXISTS text_search tsvector
    GENERATED ALWAYS AS (to_tsvector('english', text)) STORED;

CREATE INDEX IF NOT EXISTS idx_transcript_segments_text_search
    ON transcript_segments USING GIN (text_search);
