package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_emojiDownload(t *testing.T) {
	wantCode := 200
	wantData := []byte("fail")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(wantCode)
		w.Write(wantData)
	}))
	defer server.Close()

}
