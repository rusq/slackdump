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
