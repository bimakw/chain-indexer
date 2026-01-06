package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/application/services"
	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

// TransferHandler handles HTTP requests for transfers
type TransferHandler struct {
	service *services.TransferService
	logger  *zap.Logger
}

// NewTransferHandler creates a new transfer handler
func NewTransferHandler(service *services.TransferService, logger *zap.Logger) *TransferHandler {
	return &TransferHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers the transfer routes
func (h *TransferHandler) RegisterRoutes(r chi.Router) {
	r.Get("/transfers", h.GetTransfers)
	r.Get("/transfers/address/{address}", h.GetTransfersByAddress)
	r.Get("/tokens/{tokenAddress}/transfers", h.GetTransfersByToken)
}

// GetTransfers handles GET /transfers
func (h *TransferHandler) GetTransfers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := entities.DefaultTransferFilter()

	// Parse query parameters
	if v := r.URL.Query().Get("token"); v != "" {
		addr := strings.ToLower(v)
		filter.TokenAddress = &addr
	}
	if v := r.URL.Query().Get("from"); v != "" {
		addr := strings.ToLower(v)
		filter.FromAddress = &addr
	}
	if v := r.URL.Query().Get("to"); v != "" {
		addr := strings.ToLower(v)
		filter.ToAddress = &addr
	}
	if v := r.URL.Query().Get("address"); v != "" {
		addr := strings.ToLower(v)
		filter.Address = &addr
	}
	if v := r.URL.Query().Get("from_block"); v != "" {
		if block, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.FromBlock = &block
		}
	}
	if v := r.URL.Query().Get("to_block"); v != "" {
		if block, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.ToBlock = &block
		}
	}
	if v := r.URL.Query().Get("from_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.FromTime = &t
		}
	}
	if v := r.URL.Query().Get("to_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.ToTime = &t
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err := strconv.Atoi(v); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	response, err := h.service.GetTransfers(ctx, filter)
	if err != nil {
		h.logger.Error("Failed to get transfers", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get transfers")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetTransfersByAddress handles GET /transfers/address/{address}
func (h *TransferHandler) GetTransfersByAddress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	address := chi.URLParam(r, "address")

	if !isValidAddress(address) {
		h.respondError(w, http.StatusBadRequest, "Invalid address format")
		return
	}

	limit := 100
	offset := 0

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

	response, err := h.service.GetTransfersByAddress(ctx, address, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get transfers by address", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get transfers")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetTransfersByToken handles GET /tokens/{tokenAddress}/transfers
func (h *TransferHandler) GetTransfersByToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tokenAddress := chi.URLParam(r, "tokenAddress")

	if !isValidAddress(tokenAddress) {
		h.respondError(w, http.StatusBadRequest, "Invalid token address format")
		return
	}

	limit := 100
	offset := 0

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

	response, err := h.service.GetTransfersByToken(ctx, tokenAddress, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get transfers by token", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to get transfers")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *TransferHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *TransferHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

func isValidAddress(addr string) bool {
	if len(addr) != 42 {
		return false
	}
	if !strings.HasPrefix(addr, "0x") {
		return false
	}
	return true
}
