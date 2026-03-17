-- +goose Up
-- +goose StatementBegin
-- Add SIZE column to FILE table for deduplication
-- Using NOT NULL DEFAULT 0 for backward compatibility with existing archives
ALTER TABLE FILE ADD COLUMN SIZE INTEGER NOT NULL DEFAULT 0;

-- Index for fast ID+Size lookup
CREATE INDEX IF NOT EXISTS FILE_ID_SIZE_IDX ON FILE (ID, SIZE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS FILE_ID_SIZE_IDX;
-- Note: SQLite doesn't support dropping columns, so SIZE column remains
-- +goose StatementEnd