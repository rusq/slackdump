package transform

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime/trace"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/chunk/state"
)

// Reconstruct reconstructs the conversation files from the temporary directory
// with state files and chunk records.
func Reconstruct(ctx context.Context, fsa fsadapter.FS, tmpdir string, tf Interface) error {
	_, task := trace.NewTask(ctx, "reconstruct")
	defer task.End()

	return filepath.WalkDir(tmpdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != tmpdir {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".state" {
			return nil
		}

		st, err := state.Load(path)
		if err != nil {
			return fmt.Errorf("failed to load state file: %w", err)
		}

		dlog.Printf("reconstructing %s", st.ChunkFilename)
		return tf.Transform(ctx, tmpdir, st)
	})
}
