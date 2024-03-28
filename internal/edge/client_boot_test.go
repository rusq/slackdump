package edge

import (
	"context"
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed assets/client.userBoot.json
var clientUserBootJSON []byte

func TestClient_ClientUserBoot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(clientUserBootJSON)
	}))
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
