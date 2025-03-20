-- +goose Up
-- +goose StatementBegin
-- DISTINCT IS NECESSARY DUE TO LATER CHUNKS MAY CONTAIN SAME MESSAGES.
CREATE VIEW IF NOT EXISTS V_LATEST_MESSAGE AS
SELECT M.CHANNEL_ID, M.TS, M.ID
FROM MESSAGE M,
     CHUNK C
WHERE M.CHUNK_ID = C.ID
  AND C.TYPE_ID = 0
  AND (M.CHANNEL_ID, M.ID) IN (SELECT M.CHANNEL_ID, MAX(M.ID) MAX_ID
                               FROM MESSAGE M,
                                    CHUNK C
                               WHERE C.ID = M.CHUNK_ID
                                 AND C.TYPE_ID = 0
                               GROUP BY M.CHANNEL_ID);

CREATE VIEW IF NOT EXISTS V_LATEST_THREAD AS
SELECT DISTINCT M.CHANNEL_ID, M.THREAD_TS, M.TS, M.PARENT_ID, M.ID
FROM MESSAGE M,
     CHUNK C
WHERE M.CHUNK_ID = C.ID
  AND C.TYPE_ID = 1
  AND (M.CHANNEL_ID, M.THREAD_TS, M.ID) IN (SELECT M.CHANNEL_ID, M.THREAD_TS, MAX(M.ID) MAX_ID
                                        FROM MESSAGE M,
                                             CHUNK C
                                        WHERE C.ID = M.CHUNK_ID
                                          AND C.TYPE_ID = 1
                                        GROUP BY M.CHANNEL_ID, M.THREAD_TS);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS V_LATEST_THREAD;
DROP VIEW IF EXISTS V_LATEST_MESSAGE;
-- +goose StatementEnd
