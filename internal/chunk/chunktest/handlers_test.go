package chunktest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/fixtures"
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
