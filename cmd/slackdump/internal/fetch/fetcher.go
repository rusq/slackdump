package fetch

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
	"github.com/rusq/slackdump/v2/internal/event/processor"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/slack-go/slack"
)

type Parameters struct {
	Oldest    time.Time
	Latest    time.Time
	List      *structures.EntityList
	DumpFiles bool
}

func Fetch(ctx context.Context, sess *slackdump.Session, dir string, p *Parameters) error {
	if p == nil {
		return fmt.Errorf("nil parameters")
	}

	dlog.Printf("using %s as temporary directory", dir)

	for _, link := range p.List.Include {
		if err := dumpOne(ctx, sess, dir, link, p); err != nil {
			return err
		}
	}
	return nil
}

type streamer interface {
	Client() *slack.Client
	Stream(context.Context, processor.Processor, string, time.Time, time.Time) error
}

var replacer = strings.NewReplacer("/", "-", ":", "-")

func dumpOne(ctx context.Context, sess streamer, dir string, link string, p *Parameters) error {
	fileprefix := replacer.Replace(link)
	var pattern = fmt.Sprintf("%s-*.jsonl.gz", fileprefix)
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	pr, err := processor.NewStandard(ctx, gz, sess.Client(), dir, processor.DumpFiles(p.DumpFiles))
	if err != nil {
		return err
	}
	defer pr.Close()
	if err := sess.Stream(ctx, pr, link, p.Oldest, p.Latest); err != nil {
		return err
	}
	s, err := pr.State()
	if err != nil {
		return err
	}
	s.SetFilename(filepath.Base(f.Name()))
	return s.Save(filepath.Join(dir, fileprefix+".state"))
}
