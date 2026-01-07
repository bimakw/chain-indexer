package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/bimakw/chain-indexer/internal/application/services"
	"github.com/bimakw/chain-indexer/internal/config"
	"github.com/bimakw/chain-indexer/internal/infrastructure/cache"
	"github.com/bimakw/chain-indexer/internal/infrastructure/database"
	"github.com/bimakw/chain-indexer/internal/presentation/handlers"
	"github.com/bimakw/chain-indexer/internal/presentation/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := setupLogger(cfg.Log.Level)
	defer logger.Sync()

	logger.Info("Starting chain-indexer API",
		zap.Int("port", cfg.API.Port),
	)

	// Connect to database
	db, err := database.NewPostgresDB(cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Connect to Redis cache (optional)
	var redisCache *cache.RedisCache
	redisCache, err = cache.NewRedisCache(cfg.Redis, cfg.API.CacheTTL, logger)
	if err != nil {
		logger.Warn("Failed to connect to Redis, running without cache", zap.Error(err))
		redisCache = nil
	} else {
		defer redisCache.Close()
	}

	// Create repositories
	tokenRepo := database.NewTokenRepo(db.DB())
	transferRepo := database.NewTransferRepo(db.DB())

	// Create services
	transferService := services.NewTransferService(transferRepo, tokenRepo, redisCache, logger)
	tokenService := services.NewTokenService(tokenRepo, redisCache, logger)
	statsService := services.NewStatsService(transferRepo, tokenRepo, redisCache, logger)
	holdersService := services.NewHoldersService(transferRepo, tokenRepo, redisCache, logger)

	// Create handlers
	transferHandler := handlers.NewTransferHandler(transferService, logger)
	tokenHandler := handlers.NewTokenHandler(tokenService, logger)
	statsHandler := handlers.NewStatsHandler(statsService, logger)
	holdersHandler := handlers.NewHoldersHandler(holdersService, logger)

	var cacheChecker handlers.HealthChecker
	if redisCache != nil {
		cacheChecker = redisCache
	}
	healthHandler := handlers.NewHealthHandler(db, cacheChecker)

	// Setup router
	r := chi.NewRouter()

	// Middleware stack
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Metrics())
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RateLimiter(cfg.API.RateLimitRPS))

	// Health endpoints (no rate limiting)
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)
	r.Get("/live", healthHandler.Live)
	r.Handle("/metrics", promhttp.Handler())

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		transferHandler.RegisterRoutes(r)
		tokenHandler.RegisterRoutes(r)
		r.Get("/tokens/{address}/stats", statsHandler.GetTokenStats)
		r.Get("/tokens/{address}/holder-count", statsHandler.GetHolderCount)
		r.Get("/tokens/{address}/holders", holdersHandler.GetTopHolders)
		r.Get("/tokens/{address}/holders/{holder_address}", holdersHandler.GetHolderBalance)
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.API.ReadTimeout,
		WriteTimeout: cfg.API.WriteTimeout,
	}

	// Run server in goroutine
	go func() {
		logger.Info("API server starting", zap.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Received shutdown signal, shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.API.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("Server stopped")
}

func setupLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, _ := config.Build()
	return logger
}
