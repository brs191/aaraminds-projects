-- 0007: direct uploads (PRD R1) are staged as media_artifacts rows before a
-- job exists, so job_id becomes nullable. A submit with an upload:// source
-- URI links the staged bytes to the new job via a job-owned source_media row.

ALTER TABLE media_artifacts ALTER COLUMN job_id DROP NOT NULL;
