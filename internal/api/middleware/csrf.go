package middleware

import (
	"net/http"
	"strings"
)

// CSRF enforces double-submit CSRF protection for cookie-authenticated writes.
// Bearer-token requests are exempt.
func CSRF(csrfCookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			if hasBearerAuth(r) {
				next.ServeHTTP(w, r)
				return
			}
			if csrfCookieName == "" {
				http.Error(w, `{"error":"csrf cookie name not configured"}`, http.StatusForbidden)
				return
			}

			cookie, err := r.Cookie(csrfCookieName)
			if err != nil || strings.TrimSpace(cookie.Value) == "" {
				http.Error(w, `{"error":"missing csrf cookie"}`, http.StatusForbidden)
				return
			}
			header := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
			if header == "" || header != cookie.Value {
				http.Error(w, `{"error":"invalid csrf token"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func hasBearerAuth(r *http.Request) bool {
	return strings.HasPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")
}

