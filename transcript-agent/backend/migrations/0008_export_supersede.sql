-- 0008: export supersede tracking (PRD 13.2 r5). Re-approval marks every
-- prior export superseded inside the approve transaction; superseded exports
-- stay downloadable but responses carry X-Superseded: true.

ALTER TABLE exports ADD COLUMN IF NOT EXISTS superseded BOOLEAN NOT NULL DEFAULT FALSE;
