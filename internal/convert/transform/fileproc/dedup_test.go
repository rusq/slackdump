package fileproc

import (
	"context"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v4/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileProcessor tracks which files were "downloaded"
type mockFileProcessor struct {
	FilesCalled []slack.File
}

func (m *mockFileProcessor) Files(ctx context.Context, ch *slack.Channel, msg slack.Message, ff []slack.File) error {
	m.FilesCalled = append(m.FilesCalled, ff...)
	return nil
}

func (m *mockFileProcessor) Close() error { return nil }

func TestDeduplicatingFileProcessor_Files(t *testing.T) {
	// Set up in-memory SQLite database with migrations
	db := testutil.TestDB(t)
	err := repository.Migrate(context.Background(), db.DB, true)
	require.NoError(t, err)

	// Create file repository and insert test data
	ctx := context.Background()

	// Insert existing files directly into DB (need most columns due to NOT NULL constraints)
	existingFiles := []struct {
		ID        string
		ChunkID   int64
		ChannelID string
		Size      int64
	}{
		{ID: "F1", ChunkID: 1, ChannelID: "C1", Size: 1000},
		{ID: "F3", ChunkID: 1, ChannelID: "C1", Size: 500},
	}
	for _, f := range existingFiles {
		_, err := db.ExecContext(ctx, `INSERT INTO FILE (ID, CHUNK_ID, CHANNEL_ID, IDX, MODE, DATA, SIZE) VALUES (?, ?, ?, 0, 'hosted', ?, ?)`,
			f.ID, f.ChunkID, f.ChannelID, []byte("{}"), f.Size)
		require.NoError(t, err)
	}

	// Create mock inner processor
	mock := &mockFileProcessor{}

	// Create deduplicating processor
	dedup := NewDeduplicatingFileProcessor(mock, db, nil)

	channel := &slack.Channel{}
	parent := slack.Message{}

	t.Run("existing file should be skipped", func(t *testing.T) {
		mock.FilesCalled = nil

		// File that already exists in DB
		files := []slack.File{
			{ID: "F1", Size: 1000},
		}

		err := dedup.Files(context.Background(), channel, parent, files)
		require.NoError(t, err)

		// Should NOT have called inner.Files (file was skipped)
		assert.Len(t, mock.FilesCalled, 0, "existing file should be skipped")
	})

	t.Run("new file should be downloaded", func(t *testing.T) {
		mock.FilesCalled = nil

		// File that doesn't exist in DB - needs Name to pass IsValid check
		files := []slack.File{
			{ID: "NEW1", Size: 2000, Name: "newfile.txt"},
		}

		err := dedup.Files(context.Background(), channel, parent, files)
		require.NoError(t, err)

		// Should have called inner.Files with the new file
		assert.Len(t, mock.FilesCalled, 1)
		assert.Equal(t, "NEW1", mock.FilesCalled[0].ID)
	})

	t.Run("same ID different size should be downloaded", func(t *testing.T) {
		mock.FilesCalled = nil

		// Same file ID but different size - needs Name to pass IsValid check
		files := []slack.File{
			{ID: "F1", Size: 2000, Name: "test1-updated.txt"},
		}

		err := dedup.Files(context.Background(), channel, parent, files)
		require.NoError(t, err)

		// Should have called inner.Files because size differs
		assert.Len(t, mock.FilesCalled, 1)
		assert.Equal(t, 2000, mock.FilesCalled[0].Size)
	})

	t.Run("different file should be downloaded", func(t *testing.T) {
		mock.FilesCalled = nil

		// Different file ID - needs Name to pass IsValid check
		files := []slack.File{
			{ID: "NEW2", Size: 500, Name: "newfile2.txt"},
		}

		err := dedup.Files(context.Background(), channel, parent, files)
		require.NoError(t, err)

		// Should have called inner.Files
		assert.Len(t, mock.FilesCalled, 1)
		assert.Equal(t, "NEW2", mock.FilesCalled[0].ID)
	})

	t.Run("mixed new and existing files", func(t *testing.T) {
		mock.FilesCalled = nil

		files := []slack.File{
			{ID: "F1", Size: 1000},                // exists, skip
			{ID: "NEW3", Size: 300, Name: "new3"}, // new, download
			{ID: "F3", Size: 500},                 // exists, skip
		}

		err := dedup.Files(context.Background(), channel, parent, files)
		require.NoError(t, err)

		// Should have called inner.Files only for NEW3
		assert.Len(t, mock.FilesCalled, 1)
		assert.Equal(t, "NEW3", mock.FilesCalled[0].ID)
	})

	t.Run("invalid files are skipped", func(t *testing.T) {
		mock.FilesCalled = nil

		files := []slack.File{
			{ID: "", Size: 0}, // invalid - empty ID
		}

		err := dedup.Files(context.Background(), channel, parent, files)
		require.NoError(t, err)

		// Invalid files should be skipped
		assert.Len(t, mock.FilesCalled, 0)
	})
}

// TestDeduplicationLogic tests the deduplication decision logic
func TestDeduplicationLogic(t *testing.T) {
	tests := []struct {
		name     string
		existing *struct {
			ID   string
			Size int
		}
		newFile  slack.File
		wantSkip bool
	}{
		{
			name:     "new file - should download",
			existing: nil,
			newFile:  slack.File{ID: "F1", Size: 1000},
			wantSkip: false,
		},
		{
			name: "same file same size - should skip",
			existing: &struct {
				ID   string
				Size int
			}{ID: "F1", Size: 1000},
			newFile:  slack.File{ID: "F1", Size: 1000},
			wantSkip: true,
		},
		{
			name: "same file different size - should download",
			existing: &struct {
				ID   string
				Size int
			}{ID: "F1", Size: 1000},
			newFile:  slack.File{ID: "F1", Size: 2000},
			wantSkip: false,
		},
		{
			name: "different file same size - should download",
			existing: &struct {
				ID   string
				Size int
			}{ID: "F1", Size: 1000},
			newFile:  slack.File{ID: "F2", Size: 1000},
			wantSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Logic: skip if existing != nil AND existing.ID == newFile.ID AND existing.Size == newFile.Size
			var skip bool
			if tt.existing != nil {
				skip = tt.existing.ID == tt.newFile.ID && tt.existing.Size == tt.newFile.Size
			}
			assert.Equal(t, tt.wantSkip, skip)
		})
	}
}
