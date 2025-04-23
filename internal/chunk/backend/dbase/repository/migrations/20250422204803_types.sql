-- +goose Up
-- +goose StatementBegin
-- TYPES CONTAINS CHUNK TYPE NAMES
CREATE TABLE TYPES (
	ID INT,
	NAME TEXT
);

INSERT INTO TYPES (ID, NAME) VALUES
 (0, 'MESSAGES')
,(1, 'THREAD_MESSAGES')
,(2, 'FILES')
,(3, 'USERS')
,(4, 'CHANNELS')
,(5, 'CHANNEL_INFO')
,(6, 'WORKSPACE_INFO')
,(7, 'CHANNEL_USERS')
,(8, 'STARRED_ITEMS')
,(9, 'BOOKMARKS')
,(10, 'SEARCH_MESSAGES')
,(11, 'SEARCH_FILES')
;

CREATE UNIQUE INDEX idx_types_id ON TYPES (ID);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE TYPES;
-- +goose StatementEnd
