package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/application/services"
	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/testutil"
)

func setupTokenHandlerTest() (*TokenHandler, *testutil.MockTokenRepository) {
	tokenRepo := testutil.NewMockTokenRepository()
	logger := zap.NewNop()

	service := services.NewTokenService(tokenRepo, nil, logger)
	handler := NewTokenHandler(service, logger)

	return handler, tokenRepo
}

func TestNewTokenHandler(t *testing.T) {
	handler, _ := setupTokenHandlerTest()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestTokenHandler_GetAllTokens_Success(t *testing.T) {
	handler, tokenRepo := setupTokenHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
		testutil.TokenWithSymbol("USDT"),
	))
	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDCAddress),
		testutil.TokenWithSymbol("USDC"),
	))

	req := httptest.NewRequest(http.MethodGet, "/tokens", nil)
	rec := httptest.NewRecorder()

	handler.GetAllTokens(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TokenListResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Pagination.Total != 2 {
		t.Errorf("expected 2 tokens, got %d", response.Pagination.Total)
	}
	if len(response.Data) != 2 {
		t.Errorf("expected 2 tokens in data, got %d", len(response.Data))
	}
}

func TestTokenHandler_GetAllTokens_WithQueryParams(t *testing.T) {
	handler, tokenRepo := setupTokenHandlerTest()

	// Add 5 tokens
	for i := 0; i < 5; i++ {
		addr := "0x" + string(rune('a'+i)) + "000000000000000000000000000000000000000"
		tokenRepo.AddToken(testutil.CreateTestToken(
			testutil.TokenWithAddress(addr),
		))
	}

	req := httptest.NewRequest(http.MethodGet, "/tokens?limit=3&offset=1&sort_by=symbol&sort_order=asc", nil)
	rec := httptest.NewRecorder()

	handler.GetAllTokens(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TokenListResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Pagination.Limit != 3 {
		t.Errorf("expected limit 3, got %d", response.Pagination.Limit)
	}
	if response.Pagination.Offset != 1 {
		t.Errorf("expected offset 1, got %d", response.Pagination.Offset)
	}
}

func TestTokenHandler_GetAllTokens_DefaultParams(t *testing.T) {
	handler, _ := setupTokenHandlerTest()

	req := httptest.NewRequest(http.MethodGet, "/tokens", nil)
	rec := httptest.NewRecorder()

	handler.GetAllTokens(rec, req)

	var response services.TokenListResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Pagination.Limit != 100 {
		t.Errorf("expected default limit 100, got %d", response.Pagination.Limit)
	}
	if response.Pagination.Offset != 0 {
		t.Errorf("expected default offset 0, got %d", response.Pagination.Offset)
	}
}

func TestTokenHandler_GetAllTokens_InvalidLimit(t *testing.T) {
	handler, _ := setupTokenHandlerTest()

	// Test with limit > 1000 (should use default)
	req := httptest.NewRequest(http.MethodGet, "/tokens?limit=5000", nil)
	rec := httptest.NewRecorder()

	handler.GetAllTokens(rec, req)

	var response services.TokenListResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Pagination.Limit != 100 {
		t.Errorf("expected default limit 100, got %d", response.Pagination.Limit)
	}
}

func TestTokenHandler_GetAllTokens_InvalidSortOrder(t *testing.T) {
	handler, _ := setupTokenHandlerTest()

	req := httptest.NewRequest(http.MethodGet, "/tokens?sort_order=INVALID", nil)
	rec := httptest.NewRecorder()

	handler.GetAllTokens(rec, req)

	// Should still succeed with default sort order
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestTokenHandler_GetAllTokens_ServiceError(t *testing.T) {
	handler, tokenRepo := setupTokenHandlerTest()

	tokenRepo.GetAllPaginatedFunc = func(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]*entities.Token, int64, error) {
		return nil, 0, errors.New("database error")
	}

	req := httptest.NewRequest(http.MethodGet, "/tokens", nil)
	rec := httptest.NewRecorder()

	handler.GetAllTokens(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Failed to get tokens" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestTokenHandler_GetByAddress_Success(t *testing.T) {
	handler, tokenRepo := setupTokenHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
		testutil.TokenWithName("Tether USD"),
		testutil.TokenWithSymbol("USDT"),
	))

	r := chi.NewRouter()
	r.Get("/tokens/{address}", handler.GetByAddress)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TokenResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Data.Address != testutil.USDTAddress {
		t.Errorf("expected address %s, got %s", testutil.USDTAddress, response.Data.Address)
	}
	if response.Data.Symbol != "USDT" {
		t.Errorf("expected symbol USDT, got %s", response.Data.Symbol)
	}
}

func TestTokenHandler_GetByAddress_NotFound(t *testing.T) {
	handler, _ := setupTokenHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{address}", handler.GetByAddress)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "token not found" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestTokenHandler_GetByAddress_InvalidAddress(t *testing.T) {
	handler, _ := setupTokenHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{address}", handler.GetByAddress)

	tests := []struct {
		name    string
		address string
	}{
		{"too short", "0x1234"},
		{"no prefix", "1111111111111111111111111111111111111111"},
		{"wrong prefix", "1x1111111111111111111111111111111111111111"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/tokens/"+tt.address, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", rec.Code)
			}

			var response map[string]string
			json.NewDecoder(rec.Body).Decode(&response)
			if response["error"] != "Invalid address format" {
				t.Errorf("unexpected error: %s", response["error"])
			}
		})
	}
}

func TestTokenHandler_GetByAddress_UppercaseAddress(t *testing.T) {
	handler, tokenRepo := setupTokenHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	r := chi.NewRouter()
	r.Get("/tokens/{address}", handler.GetByAddress)

	// Use uppercase address
	upperAddr := "0xDAC17F958D2EE523A2206206994597C13D831EC7"
	req := httptest.NewRequest(http.MethodGet, "/tokens/"+upperAddr, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TokenResponse
	json.NewDecoder(rec.Body).Decode(&response)

	// Should return lowercase address
	if response.Data.Address != testutil.USDTAddress {
		t.Errorf("expected lowercase address %s, got %s", testutil.USDTAddress, response.Data.Address)
	}
}

func TestTokenHandler_GetByAddress_ServiceError(t *testing.T) {
	handler, tokenRepo := setupTokenHandlerTest()

	tokenRepo.GetByAddressFunc = func(ctx context.Context, address string) (*entities.Token, error) {
		return nil, errors.New("database error")
	}

	r := chi.NewRouter()
	r.Get("/tokens/{address}", handler.GetByAddress)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Failed to get token" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestTokenHandler_RegisterRoutes(t *testing.T) {
	handler, tokenRepo := setupTokenHandlerTest()

	tokenRepo.AddToken(testutil.CreateTestToken(
		testutil.TokenWithAddress(testutil.USDTAddress),
	))

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// Verify routes are registered by making requests
	routes := []struct {
		method       string
		path         string
		expectStatus int
	}{
		{"GET", "/tokens", http.StatusOK},
		{"GET", "/tokens/" + testutil.USDTAddress, http.StatusOK},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		// Should not return 404 (route not found)
		if rec.Code == http.StatusNotFound {
			t.Errorf("route %s %s not registered", route.method, route.path)
		}
	}
}

func TestTokenHandler_ResponseContentType(t *testing.T) {
	handler, _ := setupTokenHandlerTest()

	req := httptest.NewRequest(http.MethodGet, "/tokens", nil)
	rec := httptest.NewRecorder()

	handler.GetAllTokens(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestTokenHandler_EmptyList(t *testing.T) {
	handler, _ := setupTokenHandlerTest()

	req := httptest.NewRequest(http.MethodGet, "/tokens", nil)
	rec := httptest.NewRecorder()

	handler.GetAllTokens(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TokenListResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Pagination.Total != 0 {
		t.Errorf("expected total 0, got %d", response.Pagination.Total)
	}
	if len(response.Data) != 0 {
		t.Errorf("expected 0 tokens in data, got %d", len(response.Data))
	}
}
