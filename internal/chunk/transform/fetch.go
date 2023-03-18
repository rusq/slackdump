package transform

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/slack-go/slack"
)

type Parameters struct {
	Oldest    time.Time
	Latest    time.Time
	List      *structures.EntityList
	DumpFiles bool
}

type streamer interface {
	Client() *slack.Client
	Stream(opt ...slackdump.StreamOption) *slackdump.Stream
}

var replacer = strings.NewReplacer("/", "-", ":", "-")

// Fetch dumps a single conversation or thread into a directory,
// returning the name of the state file that was created.  State file contains
// the information about the filename of the chunk recording file, as well as
// paths to downloaded files.
func Fetch(ctx context.Context, sess streamer, dir string, link string, p *Parameters) (string, error) {
	fileprefix := replacer.Replace(link)
	var pattern = fmt.Sprintf("%s-*.jsonl.gz", fileprefix)
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	pr, err := processor.NewStandard(ctx, gz, sess.Client(), dir, processor.DumpFiles(p.DumpFiles))
	if err != nil {
		return "", err
	}
	defer pr.Close()

	state, err := pr.State()
	if err != nil {
		return "", err
	}
	state.
		SetChunkFilename(filepath.Base(f.Name())).
		SetIsCompressed(true)
	if p.DumpFiles {
		state.SetFilesDir(fileprefix)
	}
	statefile := filepath.Join(dir, fileprefix+".state")
	defer func() {
		// we are deferring this so that it would execute even if the error
		// has occurred to have a consistent state.
		if err := state.Save(statefile); err != nil {
			dlog.Print(err)
			return
		}
	}()
	if err := sess.Stream(
		slackdump.OptLatest(p.Latest),
		slackdump.OptOldest(p.Oldest),
	).Conversations(ctx, pr, link); err != nil {
		return statefile, err
	}
	if ctx.Err() != nil {
		return statefile, ctx.Err()
	}
	state.SetIsComplete(true)
	return statefile, nil
}
