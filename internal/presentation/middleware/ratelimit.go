package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

func RateLimiter(requestsPerSecond int) func(http.Handler) http.Handler {
	return httprate.LimitByIP(requestsPerSecond, time.Second)
}
