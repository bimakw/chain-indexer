package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/application/services"
)

// TokenHandler handles HTTP requests for tokens
type TokenHandler struct {
	service *services.TokenService
	logger  *zap.Logger
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(service *services.TokenService, logger *zap.Logger) *TokenHandler {
	return &TokenHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers the token routes
func (h *TokenHandler) RegisterRoutes(r chi.Router) {
	r.Get("/tokens", h.GetAllTokens)
	r.Get("/tokens/{address}", h.GetByAddress)
}

// GetAllTokens handles GET /api/v1/tokens
func (h *TokenHandler) GetAllTokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters with defaults
	limit := 100
	offset := 0
	sortBy := "total_indexed_transfers"
	sortOrder := "desc"

	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
		}
	}
	if v := r.URL.Query().Get("sort_by"); v != "" {
		sortBy = v
	}
	if v := r.URL.Query().Get("sort_order"); v != "" {
		v = strings.ToLower(v)
		if v == "asc" || v == "desc" {
			sortOrder = v
		}
	}

	response, err := h.service.GetAllTokens(ctx, limit, offset, sortBy, sortOrder)
	if err != nil {
		h.logger.Error("Failed to get tokens", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get tokens")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetByAddress handles GET /api/v1/tokens/{address}
func (h *TokenHandler) GetByAddress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	address := chi.URLParam(r, "address")

	if !isValidAddress(address) {
		h.respondError(w, http.StatusBadRequest, "Invalid address format")
		return
	}

	address = strings.ToLower(address)

	response, err := h.service.GetByAddress(ctx, address)
	if err != nil {
		h.logger.Error("Failed to get token", zap.Error(err), zap.String("address", address))
		h.respondError(w, http.StatusInternalServerError, "Failed to get token")
		return
	}

	if response == nil {
		h.respondError(w, http.StatusNotFound, "token not found")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *TokenHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *TokenHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
