-- +goose Up
-- +goose StatementBegin
ALTER TABLE CHUNK ADD COLUMN THREAD_ONLY BOOLEAN;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE CHUNK DROP COLUMN THREAD_ONLY;
-- +goose StatementEnd
