package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bimakw/chain-indexer/internal/testutil"
)

func TestNewHealthHandler(t *testing.T) {
	db := testutil.NewMockHealthChecker(true)
	cache := testutil.NewMockHealthChecker(true)

	handler := NewHealthHandler(db, cache)
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHealthHandler_Health_AllHealthy(t *testing.T) {
	db := testutil.NewMockHealthChecker(true)
	cache := testutil.NewMockHealthChecker(true)
	handler := NewHealthHandler(db, cache)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("expected status healthy, got %s", response.Status)
	}
	if response.Services["database"] != "healthy" {
		t.Errorf("expected database healthy, got %s", response.Services["database"])
	}
	if response.Services["cache"] != "healthy" {
		t.Errorf("expected cache healthy, got %s", response.Services["cache"])
	}
	if response.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestHealthHandler_Health_DatabaseUnhealthy(t *testing.T) {
	db := testutil.NewMockHealthChecker(false)
	cache := testutil.NewMockHealthChecker(true)
	handler := NewHealthHandler(db, cache)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	var response HealthResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Status != "unhealthy" {
		t.Errorf("expected status unhealthy, got %s", response.Status)
	}
	if response.Services["database"] == "healthy" {
		t.Error("expected database to be unhealthy")
	}
}

func TestHealthHandler_Health_CacheUnhealthy(t *testing.T) {
	db := testutil.NewMockHealthChecker(true)
	cache := testutil.NewMockHealthChecker(false)
	handler := NewHealthHandler(db, cache)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	// Cache unhealthy should result in "degraded" status, not "unhealthy"
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for degraded, got %d", rec.Code)
	}

	var response HealthResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Status != "degraded" {
		t.Errorf("expected status degraded, got %s", response.Status)
	}
	if response.Services["cache"] == "healthy" {
		t.Error("expected cache to be unhealthy")
	}
}

func TestHealthHandler_Health_NoCache(t *testing.T) {
	db := testutil.NewMockHealthChecker(true)
	handler := NewHealthHandler(db, nil) // No cache

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response HealthResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Status != "healthy" {
		t.Errorf("expected status healthy, got %s", response.Status)
	}
	// Cache should not be in services
	if _, exists := response.Services["cache"]; exists {
		t.Error("cache should not be in services when nil")
	}
}

func TestHealthHandler_Health_ContentType(t *testing.T) {
	db := testutil.NewMockHealthChecker(true)
	handler := NewHealthHandler(db, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestHealthHandler_Ready_Healthy(t *testing.T) {
	db := testutil.NewMockHealthChecker(true)
	handler := NewHealthHandler(db, nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body != "ready" {
		t.Errorf("expected body 'ready', got '%s'", body)
	}
}

func TestHealthHandler_Ready_Unhealthy(t *testing.T) {
	db := testutil.NewMockHealthChecker(false)
	handler := NewHealthHandler(db, nil)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
}

func TestHealthHandler_Live(t *testing.T) {
	db := testutil.NewMockHealthChecker(true)
	handler := NewHealthHandler(db, nil)

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()

	handler.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body != "alive" {
		t.Errorf("expected body 'alive', got '%s'", body)
	}
}

func TestHealthHandler_Live_AlwaysAlive(t *testing.T) {
	// Even when DB is unhealthy, liveness should pass
	db := testutil.NewMockHealthChecker(false)
	handler := NewHealthHandler(db, nil)

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()

	handler.Live(rec, req)

	// Liveness should always return 200
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHealthResponse_Structure(t *testing.T) {
	db := testutil.NewMockHealthChecker(true)
	cache := testutil.NewMockHealthChecker(true)
	handler := NewHealthHandler(db, cache)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	var response map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	// Check required fields
	requiredFields := []string{"status", "timestamp", "services"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Check services structure
	services, ok := response["services"].(map[string]interface{})
	if !ok {
		t.Fatal("services should be a map")
	}
	if _, exists := services["database"]; !exists {
		t.Error("missing database in services")
	}
}
