package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// LoginRateLimit restricts login attempts by IP.
func LoginRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitByIP(5, 15*time.Minute)
}

// AuthRateLimit limits authenticated traffic.
func AuthRateLimit() func(http.Handler) http.Handler {
	return httprate.LimitByIP(100, time.Minute)
}

