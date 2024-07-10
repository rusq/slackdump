package viewer

import (
	"fmt"
	"net/http"
	"time"
)

// cacheMware is a middleware that sets cache control headers.
// [Mozilla reference].
//
// [Mozilla reference]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
func cacheMwareFunc(t time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			val := "no-cache, no-store, must-revalidate"
			if t > 0 {
				val = fmt.Sprintf("max-age=%d", int(t.Seconds()))
			}
			w.Header().Set("Cache-Control", val)
			next.ServeHTTP(w, r)
		})
	}
}
