package viewer

import (
	"net"
	"net/http"
	"strings"
)

const viewerCSP = "default-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self' http: https: data:; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'none'; frame-src 'self'; frame-ancestors 'self'"

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", viewerCSP)
		next.ServeHTTP(w, r)
	})
}

func hostGuard(allowed map[string]struct{}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, ok := requestHost(r.Host)
		if !ok {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if _, ok := allowed[host]; !ok {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestHost(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	} else if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
	} else if strings.HasPrefix(value, "[") || strings.Count(value, ":") == 1 {
		return "", false
	}
	value = strings.Trim(strings.ToLower(value), "[]")
	return value, value != ""
}

func allowedHosts(addr string, extra []string) map[string]struct{} {
	allowed := map[string]struct{}{"localhost": {}, "127.0.0.1": {}, "::1": {}}
	add := func(value string) {
		host, ok := requestHost(value)
		if !ok {
			return
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
			return
		}
		if host != "0.0.0.0" && host != "::" {
			allowed[host] = struct{}{}
		}
	}
	add(addr)
	for _, host := range extra {
		add(host)
	}
	return allowed
}
