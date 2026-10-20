-- +goose Up
-- Restrict duplicate candidates by message identity before checking older chunks
-- and DATA. MESSAGE_I1 alone permits a channel-wide range probe (issue #750).
CREATE INDEX MESSAGE_DEDUPE_IDX ON MESSAGE (CHANNEL_ID, ID, CHUNK_ID);

-- +goose Down
DROP INDEX MESSAGE_DEDUPE_IDX;
