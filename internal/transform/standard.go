package transform

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/state"
)

type Standard struct {
	fs fsadapter.FS
}

func NewStandard(fs fsadapter.FS) *Standard {
	return &Standard{fs: fs}
}

func (s *Standard) Transform(st *state.State) error {
	if st == nil {
		return fmt.Errorf("nil state")
	}
	var rs io.ReadSeeker
	f, err := os.Open(st.Filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if st.IsCompressed {
		tf, err := uncompress(f)
		if err != nil {
			return err
		}
		defer os.Remove(tf.Name())
		defer tf.Close()
		rs = tf
	} else {
		rs = f
	}
	pl, err := chunk.NewPlayer(rs)
	if err != nil {
		return err
	}
	_ = pl

	return nil
}

// uncompress decompresses a gzip file and returns a temporary file handler.
// it must be removed after use.
func uncompress(r io.Reader) (*os.File, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	f, err := os.CreateTemp("", "fsadapter-*")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(f, gr)
	if err != nil {
		return nil, err
	}
	return f, nil
}
