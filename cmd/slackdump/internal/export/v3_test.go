// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package export

import (
	"path/filepath"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/chunktest"
	"github.com/rusq/slackdump/v3/internal/client"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/structures"
)

var (
	baseDir   = filepath.Join("..", "..", "..", "..")
	chunkDir  = filepath.Join(baseDir, "tmp", "2")
	guestDir  = filepath.Join(baseDir, "tmp", "guest")
	largeFile = filepath.Join(chunkDir, "C0BBSGYFN.json.gz")
)

func Test_exportV3(t *testing.T) {
	fixtures.SkipInCI(t)
	fixtures.SkipOnWindows(t)
	// // TODO: this is manual
	// t.Run("large file", func(t *testing.T) {
	// 	srv := chunktest.NewDirServer(chunkDir)
	// 	defer srv.Close()
	// 	cl := slack.New("", slack.OptionAPIURL(srv.URL()))

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
		fixtures.SkipIfNotExist(t, guestDir)
		cd, err := chunk.OpenDir(guestDir)
		if err != nil {
			t.Fatal(err)
		}
		defer cd.Close()
		srv := chunktest.NewDirServer(cd)
		defer srv.Close()
		cl := client.Wrap(slack.New("", slack.OptionAPIURL(srv.URL())))

		ctx := t.Context()
		dir := t.TempDir()
		output := filepath.Join(dir, "output.zip")
		fsa, err := fsadapter.New(output)
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()

		list := &structures.EntityList{}
		if err := exportWithDir(ctx, cl, fsa, list, exportFlags{}); err != nil {
			t.Fatal(err)
		}
	})
}
