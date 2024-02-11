package export

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/chunktest"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/logger"
)

const (
	baseDir   = "../../../../"
	chunkDir  = baseDir + "tmp/2/"
	guestDir  = baseDir + "tmp/guest/"
	largeFile = chunkDir + "C0BBSGYFN.json.gz"
)

func Test_exportV3(t *testing.T) {
	// // TODO: this is manual
	// t.Run("large file", func(t *testing.T) {
	// 	srv := chunktest.NewDirServer(chunkDir)
	// 	defer srv.Close()
	// 	cl := slack.New("", slack.OptionAPIURL(srv.URL()))

	// 	lg := dlog.New(os.Stderr, "test ", log.LstdFlags, true)
	// 	ctx := logger.NewContext(context.Background(), lg)
	// 	prov := &chunktest.TestAuth{
	// 		FakeToken:      "xoxp-1234567890-1234567890-1234567890-1234567890",
	// 		WantHTTPClient: http.DefaultClient,
	// 	}
	// 	sess, err := slackdump.New(ctx, prov, slackdump.WithSlackClient(cl), slackdump.WithLimits(slackdump.NoLimits))
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	output := filepath.Join(baseDir, "output.zip")
	// 	fsa, err := fsadapter.New(output)
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	defer fsa.Close()

	// 	list := &structures.EntityList{Include: []string{"C0BBSGYFN"}}
	// 	if err := exportV3(ctx, sess, fsa, list, export.Config{List: list}); err != nil {
	// 		t.Fatal(err)
	// 	}
	// })
	t.Run("guest user", func(t *testing.T) {
		cd, err := chunk.OpenDir(guestDir)
		if err != nil {
			t.Fatal(err)
		}
		defer cd.Close()
		srv := chunktest.NewDirServer(cd)
		defer srv.Close()
		cl := slack.New("", slack.OptionAPIURL(srv.URL()))

		lg := dlog.New(os.Stderr, "test ", log.LstdFlags, true)
		ctx := logger.NewContext(context.Background(), lg)
		prov := &chunktest.TestAuth{
			FakeToken:      "xoxp-1234567890-1234567890-1234567890-1234567890",
			WantHTTPClient: http.DefaultClient,
		}
		sess, err := slackdump.New(ctx, prov, slackdump.WithSlackClient(cl), slackdump.WithLimits(slackdump.NoLimits))
		if err != nil {
			t.Fatal(err)
		}
		output := filepath.Join(baseDir, "output.zip")
		fsa, err := fsadapter.New(output)
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()

		list := &structures.EntityList{}
		if err := exportV3(ctx, sess, fsa, list, exportFlags{}); err != nil {
			t.Fatal(err)
		}
	})
}

func load(t *testing.T, filename string) io.ReadSeeker {
	absPath, err := filepath.Abs(largeFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("test file", absPath)
	f, err := os.Open(absPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}
