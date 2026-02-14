package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/application/services"
)

type StatsHandler struct {
	service *services.StatsService
	logger  *zap.Logger
}

func NewStatsHandler(service *services.StatsService, logger *zap.Logger) *StatsHandler {
	return &StatsHandler{
		service: service,
		logger:  logger,
	}
}

func (h *StatsHandler) GetTokenStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	address := chi.URLParam(r, "address")

	if !isValidAddress(address) {
		h.respondError(w, http.StatusBadRequest, "Invalid address format")
		return
	}

	address = strings.ToLower(address)

	response, err := h.service.GetTokenStats(ctx, address)
	if err != nil {
		h.logger.Error("Failed to get token stats", zap.Error(err), zap.String("address", address))
		h.respondError(w, http.StatusInternalServerError, "Failed to get token stats")
		return
	}

	if response == nil {
		h.respondError(w, http.StatusNotFound, "token not found")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *StatsHandler) GetHolderCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	address := chi.URLParam(r, "address")

	if !isValidAddress(address) {
		h.respondError(w, http.StatusBadRequest, "Invalid address format")
		return
	}

	address = strings.ToLower(address)

	response, err := h.service.GetHolderCount(ctx, address)
	if err != nil {
		h.logger.Error("Failed to get holder count", zap.Error(err), zap.String("address", address))
		h.respondError(w, http.StatusInternalServerError, "Failed to get holder count")
		return
	}

	if response == nil {
		h.respondError(w, http.StatusNotFound, "token not found")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *StatsHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *StatsHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
