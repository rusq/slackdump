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

package transform

import (
	"path/filepath"
	"testing"

	"github.com/rusq/slackdump/v4/source"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/nametmpl"
)

func Test_stdConvert(t *testing.T) {
	testNames := []chunk.FileID{
		"CHYLGDP0D-1682335799.257359",
		"CHYLGDP0D-1682375167.836499",
		"CHM82GF99",
	}
	t.Run("manual", func(t *testing.T) {
		testDir := filepath.Join("..", "..", "..", "tmp", "3")
		fixtures.SkipIfNotExist(t, testDir)

		ctx := t.Context()

		src, err := source.Load(ctx, testDir)
		if err != nil {
			t.Fatal(err)
		}
		defer src.Close()
		tmp := t.TempDir()
		fsa, err := fsadapter.New(filepath.Join(tmp, "output-dump.zip"))
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()
		cvt := DumpConverter{
			src:  src,
			fsa:  fsa,
			tmpl: nametmpl.NewDefault(),
		}

		for i, name := range testNames {
			id, thread := chunk.FileID(name).Split()
			if err := cvt.Convert(ctx, id, thread); err != nil {
				t.Fatalf("failed on i=%d, name=%s: %s", i, name, err)
			}
		}
	})
}
