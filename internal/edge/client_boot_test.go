package edge

import (
	"context"
	_ "embed"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed assets/client.userBoot.json
var clientUserBootJSON []byte

func TestClient_ClientUserBoot(t *testing.T) {
	srv := testServer(http.StatusOK, clientUserBootJSON)
	defer srv.Close()

	cl := Client{
		cl:           http.DefaultClient,
		edgeAPI:      srv.URL + "/",
		webclientAPI: srv.URL + "/",
	}
	r, err := cl.ClientUserBoot(context.Background())
	require.NoError(t, err)
	assert.True(t, r.Ok)
	assert.Equal(t, 3, len(r.Channels))
	assert.Equal(t, 4, len(r.IMs))
}
