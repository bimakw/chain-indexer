package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/application/services"
)

// PortfolioHandler handles HTTP requests for wallet portfolio endpoints
type PortfolioHandler struct {
	service *services.PortfolioService
	logger  *zap.Logger
}

// NewPortfolioHandler creates a new portfolio handler
func NewPortfolioHandler(service *services.PortfolioService, logger *zap.Logger) *PortfolioHandler {
	return &PortfolioHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers the portfolio routes on a chi router
func (h *PortfolioHandler) RegisterRoutes(r chi.Router) {
	r.Route("/wallets", func(r chi.Router) {
		r.Get("/{address}/portfolio", h.GetPortfolio)
		r.Get("/{address}/portfolio/tokens/{tokenAddress}", h.GetTokenHolding)
		r.Get("/{address}/summary", h.GetWalletSummary)
	})
}

// GetPortfolio handles GET /api/v1/wallets/{address}/portfolio
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	address := chi.URLParam(r, "address")

	if !isValidAddress(address) {
		h.respondError(w, http.StatusBadRequest, "Invalid wallet address format")
		return
	}

	address = strings.ToLower(address)

	response, err := h.service.GetPortfolio(ctx, address)
	if err != nil {
		h.logger.Error("Failed to get portfolio",
			zap.Error(err),
			zap.String("address", address),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to get portfolio")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetTokenHolding handles GET /api/v1/wallets/{address}/portfolio/tokens/{tokenAddress}
func (h *PortfolioHandler) GetTokenHolding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	walletAddress := chi.URLParam(r, "address")
	tokenAddress := chi.URLParam(r, "tokenAddress")

	if !isValidAddress(walletAddress) {
		h.respondError(w, http.StatusBadRequest, "Invalid wallet address format")
		return
	}

	if !isValidAddress(tokenAddress) {
		h.respondError(w, http.StatusBadRequest, "Invalid token address format")
		return
	}

	walletAddress = strings.ToLower(walletAddress)
	tokenAddress = strings.ToLower(tokenAddress)

	response, err := h.service.GetPortfolioByToken(ctx, walletAddress, tokenAddress)
	if err != nil {
		h.logger.Error("Failed to get token holding",
			zap.Error(err),
			zap.String("wallet", walletAddress),
			zap.String("token", tokenAddress),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to get token holding")
		return
	}

	if response == nil {
		h.respondError(w, http.StatusNotFound, "Token not found")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetWalletSummary handles GET /api/v1/wallets/{address}/summary
func (h *PortfolioHandler) GetWalletSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	address := chi.URLParam(r, "address")

	if !isValidAddress(address) {
		h.respondError(w, http.StatusBadRequest, "Invalid wallet address format")
		return
	}

	address = strings.ToLower(address)

	response, err := h.service.GetWalletSummary(ctx, address)
	if err != nil {
		h.logger.Error("Failed to get wallet summary",
			zap.Error(err),
			zap.String("address", address),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to get wallet summary")
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *PortfolioHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *PortfolioHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
