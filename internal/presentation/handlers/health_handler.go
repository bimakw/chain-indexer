package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type HealthHandler struct {
	db    HealthChecker
	cache HealthChecker
}

func NewHealthHandler(db, cache HealthChecker) *HealthHandler {
	return &HealthHandler{
		db:    db,
		cache: cache,
	}
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  make(map[string]string),
	}

	if err := h.db.HealthCheck(ctx); err != nil {
		response.Status = "unhealthy"
		response.Services["database"] = "unhealthy: " + err.Error()
	} else {
		response.Services["database"] = "healthy"
	}

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
	_ = json.NewEncoder(w).Encode(response)
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
	_, _ = w.Write([]byte("ready"))
}

// Live handles GET /live (Kubernetes liveness probe)
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("alive"))
}
