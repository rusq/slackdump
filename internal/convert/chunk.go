package convert

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/slack-go/slack"
)

var (
	ErrNotADir    = errors.New("not a directory")
	ErrEmptyChunk = errors.New("missing chunk")
)

// ChunkToExport is a converter between Chunk and Export formats.
type ChunkToExport struct {
	// Src is the source directory with chunks
	Src *chunk.Directory
	// Trg is the target FS for the export
	Trg fsadapter.FS
	// UploadDir is the upload directory name (relative to Src)
	UploadDir    string
	IncludeFiles bool

	// FindFile should return the path to the file within the upload directory
	SrcFileLoc func(*slack.File) string
}

// Validate validates the input parameters.
func (c *ChunkToExport) Validate() error {
	if c.Src == nil || c.Trg == nil {
		return errors.New("source and target must be set")
	}
	if c.UploadDir != "" && c.IncludeFiles {
		stat, err := os.Stat(filepath.Join(c.Src.Name(), c.UploadDir))
		if err != nil {
			return fmt.Errorf("invalid uploads directory %q: %s", c.UploadDir, err)
		}
		if !stat.IsDir() {
			return fmt.Errorf("invalid uploads directory %q: %w", c.UploadDir, ErrNotADir)
		}
	}
	// users chunk is required
	if fi, err := c.Src.Stat(chunk.FUsers); err != nil {
		return fmt.Errorf("users chunk: %w", err)
	} else if fi.Size() == 0 {
		return fmt.Errorf("users chunk: %w", ErrEmptyChunk)
	}
	// we are not checking for channels chunk because channel information will
	// be stored in each of the chunk files, and we can collect it from there.

	return nil
}

func (c *ChunkToExport) Convert() error {
	if err := c.Validate(); err != nil {
		return err
	}

	return nil
}
