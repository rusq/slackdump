package repository

import (
	"context"
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func init() {
	goose.SetBaseFS(migrationsFS)
	goose.SetDialect("sqlite3")
}

func Migrate(ctx context.Context, db *sql.DB) error {
	return goose.UpContext(ctx, db, "migrations")
}
