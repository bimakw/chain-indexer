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

// HoldersHandler handles HTTP requests for token holders
type HoldersHandler struct {
	service *services.HoldersService
	logger  *zap.Logger
}

// NewHoldersHandler creates a new holders handler
func NewHoldersHandler(service *services.HoldersService, logger *zap.Logger) *HoldersHandler {
	return &HoldersHandler{
		service: service,
		logger:  logger,
	}
}

// GetTopHolders handles GET /api/v1/tokens/{address}/holders
func (h *HoldersHandler) GetTopHolders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	address := chi.URLParam(r, "address")

	if !isValidAddress(address) {
		h.respondError(w, http.StatusBadRequest, "Invalid address format")
		return
	}

	address = strings.ToLower(address)

	// Parse limit parameter (default 100, max 1000)
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			if l > 1000 {
				l = 1000
			}
			limit = l
		}
	}

	// Parse offset parameter (default 0)
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
		}
	}

	response, err := h.service.GetTopHolders(ctx, address, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get top holders", zap.Error(err), zap.String("address", address))
		h.respondError(w, http.StatusInternalServerError, "Failed to get top holders")
		return
	}

	if response == nil {
		h.respondError(w, http.StatusNotFound, "token not found")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetHolderBalance handles GET /api/v1/tokens/{address}/holders/{holder_address}
func (h *HoldersHandler) GetHolderBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tokenAddress := chi.URLParam(r, "address")
	holderAddress := chi.URLParam(r, "holder_address")

	if !isValidAddress(tokenAddress) {
		h.respondError(w, http.StatusBadRequest, "Invalid token address format")
		return
	}

	if !isValidAddress(holderAddress) {
		h.respondError(w, http.StatusBadRequest, "Invalid holder address format")
		return
	}

	tokenAddress = strings.ToLower(tokenAddress)
	holderAddress = strings.ToLower(holderAddress)

	response, err := h.service.GetHolderBalance(ctx, tokenAddress, holderAddress)
	if err != nil {
		h.logger.Error("Failed to get holder balance",
			zap.Error(err),
			zap.String("token", tokenAddress),
			zap.String("holder", holderAddress),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to get holder balance")
		return
	}

	if response == nil {
		h.respondError(w, http.StatusNotFound, "token not found")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *HoldersHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *HoldersHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
