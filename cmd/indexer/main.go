package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/bimakw/chain-indexer/internal/application/services"
	"github.com/bimakw/chain-indexer/internal/config"
	"github.com/bimakw/chain-indexer/internal/infrastructure/database"
	"github.com/bimakw/chain-indexer/internal/infrastructure/ethereum"
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

	logger.Info("Starting chain-indexer",
		zap.Strings("tokens", cfg.Indexer.TokenAddresses),
		zap.String("rpc_url", cfg.Ethereum.RPCURL),
	)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	db, err := database.NewPostgresDB(cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Connect to Ethereum node
	ethClient, err := ethereum.NewClient(cfg.Ethereum, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Ethereum node", zap.Error(err))
	}
	defer ethClient.Close()

	// Create repositories
	tokenRepo := database.NewTokenRepo(db.DB())
	transferRepo := database.NewTransferRepo(db.DB())
	stateRepo := database.NewIndexerStateRepo(db.DB())

	// Create fetcher
	fetcher := ethereum.NewFetcher(ethClient, cfg.Indexer, logger)

	// Create indexer service
	indexerService := services.NewIndexerService(
		fetcher,
		ethClient,
		tokenRepo,
		transferRepo,
		stateRepo,
		cfg.Indexer,
		logger,
	)

	// Start indexer
	if err := indexerService.Start(ctx); err != nil {
		logger.Fatal("Failed to start indexer", zap.Error(err))
	}

	// Start metrics server
	go startMetricsServer(cfg.Indexer.MetricsPort, logger)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Received shutdown signal, stopping indexer...")

	// Graceful shutdown
	indexerService.Stop()

	logger.Info("Indexer stopped")
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

func startMetricsServer(port int, logger *zap.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf(":%d", port)
	logger.Info("Starting metrics server", zap.String("addr", addr))

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Metrics server error", zap.Error(err))
	}
}
