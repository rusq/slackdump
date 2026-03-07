-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ALIAS
(
    CHANNEL_ID TEXT      NOT NULL PRIMARY KEY,
    ALIAS      TEXT,
    CREATED_AT TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ALIAS;
-- +goose StatementEnd
