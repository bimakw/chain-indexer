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

func setupTransferHandlerTest() (*TransferHandler, *testutil.MockTransferRepository, *testutil.MockTokenRepository) {
	transferRepo := testutil.NewMockTransferRepository()
	tokenRepo := testutil.NewMockTokenRepository()
	logger := zap.NewNop()

	service := services.NewTransferService(transferRepo, tokenRepo, nil, logger)
	handler := NewTransferHandler(service, logger)

	return handler, transferRepo, tokenRepo
}

func TestNewTransferHandler(t *testing.T) {
	handler, _, _ := setupTransferHandlerTest()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestTransferHandler_GetTransfers_Success(t *testing.T) {
	handler, transferRepo, _ := setupTransferHandlerTest()

	// Add test data
	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1)),
		testutil.CreateTestTransfer(testutil.WithID(2)),
	)

	req := httptest.NewRequest(http.MethodGet, "/transfers", nil)
	rec := httptest.NewRecorder()

	handler.GetTransfers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TransferResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("expected 2 transfers, got %d", response.Total)
	}
}

func TestTransferHandler_GetTransfers_WithQueryParams(t *testing.T) {
	handler, transferRepo, _ := setupTransferHandlerTest()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithTokenAddress(testutil.USDTAddress)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithTokenAddress(testutil.USDCAddress)),
	)

	// Test with token filter
	req := httptest.NewRequest(http.MethodGet, "/transfers?token="+testutil.USDTAddress, nil)
	rec := httptest.NewRecorder()

	handler.GetTransfers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TransferResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Total != 1 {
		t.Errorf("expected 1 transfer, got %d", response.Total)
	}
}

func TestTransferHandler_GetTransfers_Pagination(t *testing.T) {
	handler, transferRepo, _ := setupTransferHandlerTest()

	transfers := testutil.CreateMultipleTransfers(10)
	transferRepo.AddTransfers(transfers...)

	req := httptest.NewRequest(http.MethodGet, "/transfers?limit=5&offset=2", nil)
	rec := httptest.NewRecorder()

	handler.GetTransfers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TransferResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Limit != 5 {
		t.Errorf("expected limit 5, got %d", response.Limit)
	}
	if response.Offset != 2 {
		t.Errorf("expected offset 2, got %d", response.Offset)
	}
}

func TestTransferHandler_GetTransfers_InvalidLimit(t *testing.T) {
	handler, _, _ := setupTransferHandlerTest()

	// Test with limit > 1000 (should use default)
	req := httptest.NewRequest(http.MethodGet, "/transfers?limit=5000", nil)
	rec := httptest.NewRecorder()

	handler.GetTransfers(rec, req)

	var response services.TransferResponse
	json.NewDecoder(rec.Body).Decode(&response)

	// Should use default limit (100)
	if response.Limit != 100 {
		t.Errorf("expected default limit 100, got %d", response.Limit)
	}
}

func TestTransferHandler_GetTransfers_BlockRange(t *testing.T) {
	handler, transferRepo, _ := setupTransferHandlerTest()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithBlockNumber(100)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithBlockNumber(200)),
		testutil.CreateTestTransfer(testutil.WithID(3), testutil.WithBlockNumber(300)),
	)

	req := httptest.NewRequest(http.MethodGet, "/transfers?from_block=150&to_block=250", nil)
	rec := httptest.NewRecorder()

	handler.GetTransfers(rec, req)

	var response services.TransferResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Total != 1 {
		t.Errorf("expected 1 transfer, got %d", response.Total)
	}
}

func TestTransferHandler_GetTransfers_AddressFilters(t *testing.T) {
	handler, transferRepo, _ := setupTransferHandlerTest()

	// Set up distinct from/to addresses for each transfer
	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithFromAddress(testutil.AliceAddress), testutil.WithToAddress(testutil.CharlieAddr)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithFromAddress(testutil.CharlieAddr), testutil.WithToAddress(testutil.AliceAddress)),
		testutil.CreateTestTransfer(testutil.WithID(3), testutil.WithFromAddress(testutil.BobAddress), testutil.WithToAddress(testutil.CharlieAddr)),
	)

	// Test 'from' filter - Alice sends 1 transfer
	req := httptest.NewRequest(http.MethodGet, "/transfers?from="+testutil.AliceAddress, nil)
	rec := httptest.NewRecorder()
	handler.GetTransfers(rec, req)

	var response services.TransferResponse
	json.NewDecoder(rec.Body).Decode(&response)
	if response.Total != 1 {
		t.Errorf("from filter: expected 1, got %d", response.Total)
	}

	// Test 'to' filter - Alice receives 1 transfer
	req = httptest.NewRequest(http.MethodGet, "/transfers?to="+testutil.AliceAddress, nil)
	rec = httptest.NewRecorder()
	handler.GetTransfers(rec, req)

	json.NewDecoder(rec.Body).Decode(&response)
	if response.Total != 1 {
		t.Errorf("to filter: expected 1, got %d", response.Total)
	}

	// Test 'address' filter (matches from OR to) - Alice involved in 2 transfers
	req = httptest.NewRequest(http.MethodGet, "/transfers?address="+testutil.AliceAddress, nil)
	rec = httptest.NewRecorder()
	handler.GetTransfers(rec, req)

	json.NewDecoder(rec.Body).Decode(&response)
	if response.Total != 2 {
		t.Errorf("address filter: expected 2, got %d", response.Total)
	}
}

func TestTransferHandler_GetTransfers_ServiceError(t *testing.T) {
	handler, transferRepo, _ := setupTransferHandlerTest()

	// Simulate service error
	transferRepo.GetByFilterFunc = func(ctx context.Context, filter entities.TransferFilter) ([]entities.Transfer, error) {
		return nil, errors.New("database error")
	}

	req := httptest.NewRequest(http.MethodGet, "/transfers", nil)
	rec := httptest.NewRecorder()

	handler.GetTransfers(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Failed to get transfers" {
		t.Errorf("unexpected error message: %s", response["error"])
	}
}

func TestTransferHandler_GetTransfersByAddress_Success(t *testing.T) {
	handler, transferRepo, _ := setupTransferHandlerTest()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithFromAddress(testutil.AliceAddress)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithToAddress(testutil.AliceAddress)),
	)

	r := chi.NewRouter()
	r.Get("/transfers/address/{address}", handler.GetTransfersByAddress)

	req := httptest.NewRequest(http.MethodGet, "/transfers/address/"+testutil.AliceAddress, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TransferResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Total != 2 {
		t.Errorf("expected 2 transfers, got %d", response.Total)
	}
}

func TestTransferHandler_GetTransfersByAddress_InvalidAddress(t *testing.T) {
	handler, _, _ := setupTransferHandlerTest()

	r := chi.NewRouter()
	r.Get("/transfers/address/{address}", handler.GetTransfersByAddress)

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
			req := httptest.NewRequest(http.MethodGet, "/transfers/address/"+tt.address, nil)
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

func TestTransferHandler_GetTransfersByAddress_Pagination(t *testing.T) {
	handler, transferRepo, _ := setupTransferHandlerTest()

	transfers := testutil.CreateMultipleTransfers(10, testutil.WithFromAddress(testutil.AliceAddress))
	transferRepo.AddTransfers(transfers...)

	r := chi.NewRouter()
	r.Get("/transfers/address/{address}", handler.GetTransfersByAddress)

	req := httptest.NewRequest(http.MethodGet, "/transfers/address/"+testutil.AliceAddress+"?limit=3&offset=2", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	var response services.TransferResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Limit != 3 {
		t.Errorf("expected limit 3, got %d", response.Limit)
	}
	if response.Offset != 2 {
		t.Errorf("expected offset 2, got %d", response.Offset)
	}
}

func TestTransferHandler_GetTransfersByToken_Success(t *testing.T) {
	handler, transferRepo, _ := setupTransferHandlerTest()

	transferRepo.AddTransfers(
		testutil.CreateTestTransfer(testutil.WithID(1), testutil.WithTokenAddress(testutil.USDTAddress)),
		testutil.CreateTestTransfer(testutil.WithID(2), testutil.WithTokenAddress(testutil.USDTAddress)),
		testutil.CreateTestTransfer(testutil.WithID(3), testutil.WithTokenAddress(testutil.USDCAddress)),
	)

	r := chi.NewRouter()
	r.Get("/tokens/{tokenAddress}/transfers", handler.GetTransfersByToken)

	req := httptest.NewRequest(http.MethodGet, "/tokens/"+testutil.USDTAddress+"/transfers", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response services.TransferResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Total != 2 {
		t.Errorf("expected 2 transfers, got %d", response.Total)
	}
}

func TestTransferHandler_GetTransfersByToken_InvalidAddress(t *testing.T) {
	handler, _, _ := setupTransferHandlerTest()

	r := chi.NewRouter()
	r.Get("/tokens/{tokenAddress}/transfers", handler.GetTransfersByToken)

	req := httptest.NewRequest(http.MethodGet, "/tokens/invalid/transfers", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var response map[string]string
	json.NewDecoder(rec.Body).Decode(&response)
	if response["error"] != "Invalid token address format" {
		t.Errorf("unexpected error: %s", response["error"])
	}
}

func TestTransferHandler_RegisterRoutes(t *testing.T) {
	handler, _, _ := setupTransferHandlerTest()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// Verify routes are registered by making requests
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/transfers"},
		{"GET", "/transfers/address/" + testutil.AliceAddress},
		{"GET", "/tokens/" + testutil.USDTAddress + "/transfers"},
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

func TestIsValidAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		valid   bool
	}{
		{"valid address", "0x1111111111111111111111111111111111111111", true},
		{"valid USDT", "0xdAC17F958D2ee523a2206206994597C13D831ec7", true},
		{"too short", "0x1234", false},
		{"too long", "0x11111111111111111111111111111111111111111", false},
		{"no prefix", "1111111111111111111111111111111111111111", false},
		{"wrong prefix", "1x1111111111111111111111111111111111111111", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAddress(tt.address)
			if result != tt.valid {
				t.Errorf("isValidAddress(%s) = %v, want %v", tt.address, result, tt.valid)
			}
		})
	}
}

func TestTransferHandler_ResponseContentType(t *testing.T) {
	handler, _, _ := setupTransferHandlerTest()

	req := httptest.NewRequest(http.MethodGet, "/transfers", nil)
	rec := httptest.NewRecorder()

	handler.GetTransfers(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}
