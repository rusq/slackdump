package state

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/rusq/slackdump/v3/internal/osext"
)

// State holds the state of a chunk recording. It contains the filename of the
// chunk recording file, as well as the path to the downloaded files.

var ErrNoChunkFile = errors.New("no linked chunk file")

// OpenChunks attempts to open the chunk file linked in the State. If the
// chunk is compressed, it will be decompressed and a temporary file will be
// created. The temporary file will be removed when the OpenChunks is
// closed.
func (st *State) OpenChunks(basePath string) (io.ReadSeekCloser, error) {
	if st.ChunkFilename == "" {
		return nil, ErrNoChunkFile
	}
	f, err := os.Open(filepath.Join(basePath, st.ChunkFilename))
	if err != nil {
		return nil, err
	}
	if st.IsCompressed {
		tf, err := osext.UnGZIP(f)
		if err != nil {
			return nil, err
		}
		return osext.RemoveOnClose(tf), nil
	}
	return f, nil
}
