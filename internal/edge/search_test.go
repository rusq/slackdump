package edge

import (
	_ "embed"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed assets/search.module.channels.json
var searchChannelsJSON []byte

func TestClient_SearchChannels(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		srv := testServer(http.StatusOK, searchChannelsJSON)
		defer srv.Close()

		cl := Client{
			cl:           http.DefaultClient,
			edgeAPI:      srv.URL + "/",
			webclientAPI: srv.URL + "/",
		}
		r, err := cl.SearchChannels(t.Context(), "test")
		require.NoError(t, err)
		assert.Len(t, r, 6)
	})
	t.Run("500", func(t *testing.T) {
		srv := testServer(http.StatusInternalServerError, nil)
		defer srv.Close()

		cl := Client{
			cl:           http.DefaultClient,
			edgeAPI:      srv.URL + "/",
			webclientAPI: srv.URL + "/",
		}
		_, err := cl.SearchChannels(t.Context(), "test")
		require.Error(t, err)
	})
}
