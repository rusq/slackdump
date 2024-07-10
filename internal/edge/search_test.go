package edge

import (
	"context"
	_ "embed"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed assets/search.module.channels.json
var searchChannelsJSON []byte

func TestClient_SearchChannels(t *testing.T) {
	srv := testServer(http.StatusOK, searchChannelsJSON)
	defer srv.Close()

	cl := Client{
		cl:           http.DefaultClient,
		edgeAPI:      srv.URL + "/",
		webclientAPI: srv.URL + "/",
	}
	r, err := cl.SearchChannels(context.Background(), "test")
	require.NoError(t, err)
	assert.Len(t, r, 6)
}
