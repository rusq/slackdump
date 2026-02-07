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
package chunktest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
)

func Test_handleUsersList(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		dir := t.TempDir()
		cd, err := chunk.OpenDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		f, err := cd.Create(chunk.FWorkspace)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(fixtures.ChunkWorkspace)); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()

		ds := DirServer{
			cd: cd,
		}

		h := ds.chunkfileWrapper(chunk.FWorkspace, handleAuthTest)
		srv := httptest.NewServer(h)
		defer srv.Close()

		type slackresp struct {
			Ok    bool   `json:"ok"`
			Error string `json:"error"`
		}

		for i := 0; i < 3; i++ {
			resp, _, err := tRequest[slackresp](srv.URL + "/api/auth.test")
			if err != nil {
				t.Fatal(err)
			}
			if resp.Ok != true {
				t.Errorf("got %v, want true", resp.Ok)
			}
		}
	})
}

func tRequest[T any](uri string) (T, int, error) {
	var ret T
	resp, err := http.Get(uri)
	if err != nil {
		return ret, 0, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return ret, resp.StatusCode, err
	}
	return ret, resp.StatusCode, err
}
