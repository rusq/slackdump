-- +goose Up
-- +goose StatementBegin
CREATE TABLE SAVED_ITEM
(
    ID         INTEGER PRIMARY KEY,
    CHUNK_ID   INTEGER   NOT NULL,
    LOAD_DTTM  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ITEM_ID    TEXT      NOT NULL, -- channel ID the saved item belongs to
    ITEM_TYPE  TEXT      NOT NULL, -- e.g. "message"
    TS         TEXT,               -- message timestamp, if item_type is "message"
    STATE      TEXT,
    TODO_STATE TEXT,
    IDX        INTEGER   NOT NULL, -- INDEX OF THE ITEM WITHIN THE CHUNK
    DATA       BLOB      NOT NULL,
    FOREIGN KEY (CHUNK_ID) REFERENCES CHUNK (ID) ON DELETE CASCADE
);
CREATE INDEX SAVED_ITEM_CHUNK_ID_IDX ON SAVED_ITEM (CHUNK_ID);
CREATE INDEX SAVED_ITEM_I1 ON SAVED_ITEM (ITEM_ID, TS);

INSERT INTO TYPES (ID, NAME) VALUES (12, 'SAVED_ITEMS');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM TYPES WHERE ID = 12;
DROP TABLE SAVED_ITEM;
-- +goose StatementEnd
