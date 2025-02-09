-- vim: ft=sql ts=4
-- +goose Up
-- +goose StatementBegin
CREATE TABLE SESSION
(
    ID              INTEGER PRIMARY KEY,                          -- UNIQUE SESSION ID
    CREATED_AT      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- CREATION TIMESTAMP
    UPDATED_AT      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, -- LAST UPDATED TIMESTAMP
    PAR_SESSION_ID  INTEGER,                                      -- PARENT SESSION ID
    FROM_TS         TIMESTAMP,
    TO_TS           TIMESTAMP,
    FINISHED        SMALLINT  NOT NULL DEFAULT FALSE,             -- IF TRUE, SESSION COMPLETED SUCCESSFULLY
    FILES_ENABLED   SMALLINT  NOT NULL DEFAULT FALSE,
    AVATARS_ENABLED SMALLINT  NOT NULL DEFAULT FALSE,
    MODE            TEXT      NOT NULL,                           -- MODE OF OPERATION, I.E. ARCHIVE/RESUME
    ARGS            TEXT                                          -- COMMAND LINE ARGUMENTS
);

CREATE TABLE CHUNK
(
    ID         INTEGER PRIMARY KEY,
    UNIX_TS    INTEGER  NOT NULL,           -- UNIX TIMESTAMP OF WHEN CHUNK WAS ADDED TO THE DATABASE.
    SESSION_ID INTEGER  NOT NULL,
    TYPE_ID    SMALLINT NOT NULL,           -- CHUNK TYPE ID (chunk.ChunkType)
    NUM_REC    INTEGER  NOT NULL DEFAULT 0, -- NUMBER OF RECORDS IN THE CHUNK (I.E. LEN OF A MESSAGE SLICE)
    FINAL      SMALLINT NOT NULL DEFAULT FALSE,
    FOREIGN KEY (SESSION_ID) REFERENCES SESSION (ID) ON DELETE RESTRICT
);

CREATE TABLE MESSAGE
(
    ID         INTEGER   NOT NULL,               -- TIMESTAMP IN NUMERIC FORM
    CHUNK_ID   INTEGER   NOT NULL,
    LOAD_DTTM  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHANNEL_ID TEXT      NOT NULL,               -- CHANNEL ID
    TS         TEXT      NOT NULL,               -- TIMESTAMP OF THE MESSAGE
    PARENT_ID  INTEGER,                          -- PARENT MESSAGE ID, SET FOR THREAD CHILDREN
    THREAD_TS  TEXT,                             -- SET TO TS FOR THREAD PARENT, SET TO THREAD MSG ID FOR THREAD CHILD
    IS_PARENT  SMALLINT  NOT NULL DEFAULT FALSE, -- SET TO TRUE FOR THREAD LEAD MESSAGES,
    IDX        INTEGER   NOT NULL,               -- INDEX OF THE MESSAGE IN THE CHUNK
    NUM_FILES  INTEGER   NOT NULL DEFAULT 0,     -- IF MESSAGE HAS FILES, THIS IS THEIR COUNT
    TXT        TEXT,                             -- MESSAGE TEXT
    DATA       BLOB      NOT NULL,               -- MESSAGE JSON
    PRIMARY KEY (ID, CHUNK_ID),
    FOREIGN KEY (CHUNK_ID) REFERENCES CHUNK (ID) ON DELETE CASCADE
);

CREATE TABLE CHANNEL
(
    ID        TEXT      NOT NULL, -- CHANNEL ID
    CHUNK_ID  INTEGER   NOT NULL,
    LOAD_DTTM TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    NAME      TEXT,
    IDX       INTEGER   NOT NULL,
    DATA      BLOB      NOT NULL,
    PRIMARY KEY (ID, CHUNK_ID),
    FOREIGN KEY (CHUNK_ID) REFERENCES CHUNK (ID) ON DELETE CASCADE
);

CREATE TABLE FILE
(
    ID         TEXT      NOT NULL, -- CHANNEL ID
    CHUNK_ID   INTEGER   NOT NULL,
    LOAD_DTTM  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHANNEL_ID TEXT      NOT NULL,
    MESSAGE_ID INTEGER   NOT NULL, -- PARENT MESSAGE ID
    THREAD_ID  INTEGER,            -- POPULATED IF FILE BELONG TO THE THREAD
    IDX        INTEGER   NOT NULL,
    FILENAME   TEXT      NOT NULL,
    URL        TEXT      NOT NULL,
    DATA       BLOB      NOT NULL,
    PRIMARY KEY (ID, CHUNK_ID),
    FOREIGN KEY (CHUNK_ID) REFERENCES CHUNK (ID) ON DELETE CASCADE
);

CREATE TABLE WORKSPACE
(
    ID            INTEGER PRIMARY KEY,
    CHUNK_ID      INTEGER   NOT NULL,
    LOAD_DTTM     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    TEAM          TEXT      NOT NULL,
    USERNAME      TEXT      NOT NULL,
    TEAM_ID       TEXT      NOT NULL,
    USER_ID       TEXT      NOT NULL,
    ENTERPRISE_ID TEXT,               -- NULL ON NON-ENTERPRISE INSTANCES
    URL           TEXT      NOT NULL, -- WORKSPACE URL
    DATA          BLOB      NOT NULL,
    FOREIGN KEY (CHUNK_ID) REFERENCES CHUNK (ID) ON DELETE CASCADE
);

CREATE TABLE S_USER
(
    ID        TEXT      NOT NULL,
    CHUNK_ID  INTEGER   NOT NULL,
    LOAD_DTTM TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    IDX       INTEGER   NOT NULL,
    DATA      BLOB      NOT NULL,
    PRIMARY KEY (ID, CHUNK_ID),
    FOREIGN KEY (CHUNK_ID) REFERENCES CHUNK (ID) ON DELETE CASCADE
);

CREATE TABLE CHANNEL_USER
(
    ID         TEXT      NOT NULL, -- SLACK USER ID
    CHUNK_ID   INTEGER   NOT NULL,
    LOAD_DTTM  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHANNEL_ID TEXT      NOT NULL,
    IDX        INTEGER   NOT NULL,
    PRIMARY KEY (ID, CHANNEL_ID, CHUNK_ID),
    FOREIGN KEY (CHUNK_ID) REFERENCES CHUNK (ID) ON DELETE CASCADE
);

CREATE TABLE SEARCH_MESSAGE
(
    ID           INTEGER PRIMARY KEY,
    CHUNK_ID     INTEGER   NOT NULL,
    LOAD_DTTM    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHANNEL_ID   TEXT      NOT NULL,
    CHANNEL_NAME TEXT,
    TS           TEXT      NOT NULL,
    TXT          TEXT,
    IDX          INTEGER   NOT NULL, -- INDEX OF THE MESSAGE WITHIN THE CHUNK
    DATA         BLOB      NOT NULL,
    FOREIGN KEY (CHUNK_ID) REFERENCES CHUNK (ID) ON DELETE CASCADE
);

CREATE TABLE SEARCH_FILE
(
    ID        INTEGER PRIMARY KEY,
    CHUNK_ID  INTEGER   NOT NULL,
    LOAD_DTTM TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FILE_ID   TEXT      NOT NULL,
    IDX       INTEGER   NOT NULL,
    DATA      BLOB      NOT NULL,
    FOREIGN KEY (CHUNK_ID) REFERENCES CHUNK (ID) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE SEARCH_FILE;
DROP TABLE SEARCH_MESSAGE;
DROP TABLE CHANNEL_USER;
DROP TABLE CHANNEL_INFO;
DROP TABLE S_USER;
DROP TABLE WORKSPACE;
DROP TABLE FILE;
DROP TABLE CHANNEL;
DROP TABLE MESSAGE;
DROP TABLE CHUNK;
DROP TABLE SESSION;
-- +goose StatementEnd
