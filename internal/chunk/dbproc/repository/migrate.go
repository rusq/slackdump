package repository

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func init() {
	goose.SetBaseFS(migrationsFS)
	goose.SetDialect("sqlite3")
}

func Migrate(ctx context.Context, db *sql.DB, verbose bool) error {
	if !verbose {
		goose.SetLogger(goose.NopLogger())
	} else {
		goose.SetLogger(log.Default())
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
