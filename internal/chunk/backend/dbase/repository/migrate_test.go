package repository

import (
	"database/sql"
	"testing"
)

func TestMigrate(t *testing.T) {
	t.Run("Migrate", func(t *testing.T) {
		db, err := sql.Open(Driver, ":memory:")
		if err != nil {
			t.Fatalf("sql.Open() err = %v; want nil", err)
		}
		defer db.Close()

		if err := Migrate(t.Context(), db, true); err != nil {
			t.Fatalf("Migrate() err = %v; want nil", err)
		}
	})
}
