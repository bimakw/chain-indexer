package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// HealthChecker defines the interface for health checking components
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthHandler handles health check requests
type HealthHandler struct {
	db    HealthChecker
	cache HealthChecker
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db, cache HealthChecker) *HealthHandler {
	return &HealthHandler{
		db:    db,
		cache: cache,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// Health handles GET /health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  make(map[string]string),
	}

	// Check database
	if err := h.db.HealthCheck(ctx); err != nil {
		response.Status = "unhealthy"
		response.Services["database"] = "unhealthy: " + err.Error()
	} else {
		response.Services["database"] = "healthy"
	}

	// Check cache
	if h.cache != nil {
		if err := h.cache.HealthCheck(ctx); err != nil {
			response.Status = "degraded"
			response.Services["cache"] = "unhealthy: " + err.Error()
		} else {
			response.Services["cache"] = "healthy"
		}
	}

	status := http.StatusOK
	if response.Status == "unhealthy" {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// Ready handles GET /ready (Kubernetes readiness probe)
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.HealthCheck(ctx); err != nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

// Live handles GET /live (Kubernetes liveness probe)
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("alive"))
}
