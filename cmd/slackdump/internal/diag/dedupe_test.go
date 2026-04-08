package diag

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequireExistingDatabase(t *testing.T) {
	t.Run("missing path returns error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.sqlite")

		err := requireExistingDatabase(path)
		if err == nil {
			t.Fatal("requireExistingDatabase() error = nil, want error")
		}
		want := `database "` + path + `" does not exist`
		if err.Error() != want {
			t.Fatalf("requireExistingDatabase() error = %q, want %q", err, want)
		}
	})

	t.Run("existing path succeeds", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "slackdump.sqlite")
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		if err := requireExistingDatabase(path); err != nil {
			t.Fatalf("requireExistingDatabase() error = %v, want nil", err)
		}
	})
}
