package redownload

import (
	"log/slog"

	"github.com/dustin/go-humanize"
)

// FileStats contains the file statistics.
type FileStats struct {
	NumFiles uint
	NumBytes uint64
}

func (fs *FileStats) add(other FileStats) {
	fs.NumFiles += other.NumFiles
	fs.NumBytes += other.NumBytes
}

func (fs *FileStats) Attr() slog.Attr {
	return slog.Group("file_stats", slog.Uint64("num_files", uint64(fs.NumFiles)), slog.String("total_bytes", humanize.Bytes(fs.NumBytes)))
}
