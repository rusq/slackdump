-- +goose Up
-- +goose StatementBegin
CREATE VIEW IF NOT EXISTS V_LATEST_SAVED_ITEM AS
SELECT S.*
FROM SAVED_ITEM S
WHERE S.ID IN (SELECT MAX(ID)
               FROM SAVED_ITEM
               GROUP BY ITEM_ID, COALESCE(TS, ''));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS V_LATEST_SAVED_ITEM;
-- +goose StatementEnd
