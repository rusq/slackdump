// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package repository

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sync"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql postgres_migrations/*.sql
var migrationsFS embed.FS

var migrationMu sync.Mutex

func Migrate(ctx context.Context, db *sql.DB, verbose bool) error {
	return MigrateDriver(ctx, db, Driver, verbose)
}

// MigrateDriver migrates the database using the schema for driver.  Goose
// keeps package-global migration configuration, so calls are serialised.
func MigrateDriver(ctx context.Context, db *sql.DB, driver string, verbose bool) error {
	migrationMu.Lock()
	defer migrationMu.Unlock()

	dialect := "sqlite3"
	dir := "migrations"
	if driver == PostgresDriver || driver == "pgx" {
		dialect = "postgres"
		dir = "postgres_migrations"
	}
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("migrate: set dialect: %w", err)
	}
	if !verbose {
		goose.SetLogger(goose.NopLogger())
	} else {
		goose.SetLogger(log.Default())
	}
	if err := goose.UpContext(ctx, db, dir); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
