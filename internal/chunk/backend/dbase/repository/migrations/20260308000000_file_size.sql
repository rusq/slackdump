-- +goose Up
-- +goose StatementBegin
-- Add SIZE column to FILE table for deduplication
-- Using NOT NULL DEFAULT 0 for backward compatibility with existing archives
ALTER TABLE FILE ADD COLUMN SIZE INTEGER NOT NULL DEFAULT 0;

-- Backfill SIZE from the stored Slack file JSON for pre-existing rows.
UPDATE FILE
SET SIZE = COALESCE(CAST(JSON_EXTRACT(DATA, '$.size') AS INTEGER), 0);

-- Index for fast ID+Size lookup
CREATE INDEX IF NOT EXISTS FILE_ID_SIZE_IDX ON FILE (ID, SIZE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
PRAGMA writable_schema=ON;
DROP INDEX IF EXISTS FILE_ID_SIZE_IDX;
ALTER TABLE FILE DROP COLUMN SIZE;
PRAGMA writable_schema=OFF;
-- +goose StatementEnd
