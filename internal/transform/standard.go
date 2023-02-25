package transform

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v2/internal/event"
)

type Standard struct {
	fs fsadapter.FS
}

func NewStandard(fs fsadapter.FS) *Standard {
	return &Standard{fs: fs}
}

func (s *Standard) Transform(ei *EventsInfo) error {
	if ei == nil {
		return fmt.Errorf("nil events info")
	}
	var rs io.ReadSeeker
	f, err := os.Open(ei.File)
	if err != nil {
		return err
	}
	defer f.Close()
	if ei.IsCompressed {
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
	pl, err := event.NewPlayer(rs)
	if err != nil {
		return err
	}

	return nil
}

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
