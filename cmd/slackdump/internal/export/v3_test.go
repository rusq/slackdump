package export

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk/chunktest"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/slack-go/slack"
)

const (
	baseDir   = "../../../../"
	chunkDir  = baseDir + "tmp/2/"
	largeFile = chunkDir + "C0BBSGYFN.json.gz"
)

func Test_exportV3(t *testing.T) {
	// TODO: this is manual
	t.Run("large file", func(t *testing.T) {
		srv := chunktest.NewDirServer(chunkDir, "U0BBSGYFN")
		defer srv.Close()
		cl := slack.New("", slack.OptionAPIURL(srv.URL()))

		ctx := context.Background()
		cl.AuthTestContext(ctx)
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

		list := &structures.EntityList{Include: []string{"C0BBSGYFN"}}
		if err := exportV3(ctx, sess, fsa, list, export.Config{List: list}); err != nil {
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
