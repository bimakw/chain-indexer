package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimiter creates a rate limiting middleware
func RateLimiter(requestsPerSecond int) func(http.Handler) http.Handler {
	return httprate.LimitByIP(requestsPerSecond, time.Second)
}
