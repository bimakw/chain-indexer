package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/bimakw/chain-indexer/internal/config"
	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
	"github.com/bimakw/chain-indexer/internal/infrastructure/ethereum"
)

// IndexerService orchestrates the indexing process
type IndexerService struct {
	fetcher       *ethereum.Fetcher
	ethClient     *ethereum.Client
	tokenRepo     repositories.TokenRepository
	transferRepo  repositories.TransferRepository
	stateRepo     repositories.IndexerStateRepository
	config        config.IndexerConfig
	logger        *zap.Logger
	metrics       *IndexerMetrics
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// IndexerMetrics tracks indexer performance
type IndexerMetrics struct {
	mu                  sync.RWMutex
	BlocksIndexed       int64
	TransfersIndexed    int64
	LastIndexedBlock    int64
	LastIndexedTime     time.Time
	IndexingLatencyMs   int64
	ErrorCount          int64
}

// NewIndexerService creates a new indexer service
func NewIndexerService(
	fetcher *ethereum.Fetcher,
	ethClient *ethereum.Client,
	tokenRepo repositories.TokenRepository,
	transferRepo repositories.TransferRepository,
	stateRepo repositories.IndexerStateRepository,
	cfg config.IndexerConfig,
	logger *zap.Logger,
) *IndexerService {
	return &IndexerService{
		fetcher:      fetcher,
		ethClient:    ethClient,
		tokenRepo:    tokenRepo,
		transferRepo: transferRepo,
		stateRepo:    stateRepo,
		config:       cfg,
		logger:       logger,
		metrics:      &IndexerMetrics{},
		stopCh:       make(chan struct{}),
	}
}

// Start begins the indexing process
func (s *IndexerService) Start(ctx context.Context) error {
	s.logger.Info("Starting indexer service",
		zap.Strings("tokens", s.config.TokenAddresses),
	)

	// Initialize tokens in database
	if err := s.initializeTokens(ctx); err != nil {
		return fmt.Errorf("failed to initialize tokens: %w", err)
	}

	// Start the main indexing loop
	s.wg.Add(1)
	go s.runIndexingLoop(ctx)

	return nil
}

// Stop gracefully stops the indexer
func (s *IndexerService) Stop() {
	s.logger.Info("Stopping indexer service")
	close(s.stopCh)
	s.wg.Wait()
}

// GetMetrics returns current indexer metrics
func (s *IndexerService) GetMetrics() IndexerMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()
	return *s.metrics
}

// initializeTokens ensures all configured tokens exist in the database
func (s *IndexerService) initializeTokens(ctx context.Context) error {
	for _, addr := range s.config.TokenAddresses {
		addr = strings.ToLower(addr)

		existing, err := s.tokenRepo.GetByAddress(ctx, addr)
		if err != nil {
			return fmt.Errorf("failed to check token %s: %w", addr, err)
		}

		if existing == nil {
			// Create token entry (metadata can be fetched later)
			token := &entities.Token{
				Address:  addr,
				Name:     "Unknown",
				Symbol:   "UNK",
				Decimals: 18,
			}

			if err := s.tokenRepo.Upsert(ctx, token); err != nil {
				return fmt.Errorf("failed to create token %s: %w", addr, err)
			}

			// Initialize indexer state
			state := &entities.IndexerState{
				TokenAddress:     addr,
				LastIndexedBlock: 0,
			}
			if err := s.stateRepo.Upsert(ctx, state); err != nil {
				return fmt.Errorf("failed to create indexer state for %s: %w", addr, err)
			}

			s.logger.Info("Initialized token", zap.String("address", addr))
		}
	}

	return nil
}

// runIndexingLoop continuously indexes new blocks
func (s *IndexerService) runIndexingLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.indexNewBlocks(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.indexNewBlocks(ctx)
		}
	}
}

// indexNewBlocks indexes any new blocks since last checkpoint
func (s *IndexerService) indexNewBlocks(ctx context.Context) {
	startTime := time.Now()

	// Get safe block number (latest - confirmations)
	safeBlock, err := s.fetcher.GetSafeBlockNumber(ctx)
	if err != nil {
		s.logger.Error("Failed to get safe block number", zap.Error(err))
		s.incrementErrorCount()
		return
	}

	// Process each token
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(s.config.WorkerCount)

	for _, addr := range s.config.TokenAddresses {
		addr := strings.ToLower(addr)
		g.Go(func() error {
			return s.indexTokenTransfers(gCtx, addr, safeBlock)
		})
	}

	if err := g.Wait(); err != nil {
		s.logger.Error("Error indexing transfers", zap.Error(err))
		s.incrementErrorCount()
	}

	s.metrics.mu.Lock()
	s.metrics.IndexingLatencyMs = time.Since(startTime).Milliseconds()
	s.metrics.LastIndexedTime = time.Now()
	s.metrics.mu.Unlock()
}

// indexTokenTransfers indexes transfers for a single token
func (s *IndexerService) indexTokenTransfers(ctx context.Context, tokenAddress string, toBlock int64) error {
	// Get current state
	state, err := s.stateRepo.Get(ctx, tokenAddress)
	if err != nil {
		return fmt.Errorf("failed to get indexer state: %w", err)
	}

	if state == nil {
		return fmt.Errorf("indexer state not found for %s", tokenAddress)
	}

	fromBlock := state.LastIndexedBlock + 1
	if fromBlock > toBlock {
		// Already up to date
		return nil
	}

	// Split into batches
	ranges := ethereum.SplitBlockRange(fromBlock, toBlock, s.config.BatchSize)

	for _, r := range ranges {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := s.fetcher.FetchTransfers(ctx, []string{tokenAddress}, r.From, r.To)
		if err != nil {
			return fmt.Errorf("failed to fetch transfers for blocks %d-%d: %w", r.From, r.To, err)
		}

		if len(result.Transfers) > 0 {
			if err := s.transferRepo.BatchInsert(ctx, result.Transfers); err != nil {
				return fmt.Errorf("failed to insert transfers: %w", err)
			}

			// Update token stats
			if err := s.tokenRepo.UpdateStats(ctx, tokenAddress, int64(len(result.Transfers)), r.To); err != nil {
				s.logger.Warn("Failed to update token stats", zap.Error(err))
			}
		}

		// Update checkpoint
		if err := s.stateRepo.UpdateLastBlock(ctx, tokenAddress, r.To); err != nil {
			return fmt.Errorf("failed to update checkpoint: %w", err)
		}

		s.updateMetrics(r.To-r.From+1, int64(len(result.Transfers)), r.To)

		s.logger.Debug("Indexed block range",
			zap.String("token", tokenAddress),
			zap.Int64("from", r.From),
			zap.Int64("to", r.To),
			zap.Int("transfers", len(result.Transfers)),
		)
	}

	return nil
}

// Backfill indexes historical blocks for a token
func (s *IndexerService) Backfill(ctx context.Context, tokenAddress string, fromBlock, toBlock int64) error {
	tokenAddress = strings.ToLower(tokenAddress)

	s.logger.Info("Starting backfill",
		zap.String("token", tokenAddress),
		zap.Int64("from_block", fromBlock),
		zap.Int64("to_block", toBlock),
	)

	// Mark as backfilling
	if err := s.stateRepo.SetBackfilling(ctx, tokenAddress, true, &fromBlock, &toBlock); err != nil {
		return fmt.Errorf("failed to set backfilling state: %w", err)
	}

	defer func() {
		s.stateRepo.SetBackfilling(ctx, tokenAddress, false, nil, nil)
	}()

	ranges := ethereum.SplitBlockRange(fromBlock, toBlock, s.config.BackfillBatchSize)

	for i, r := range ranges {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := s.fetcher.FetchTransfers(ctx, []string{tokenAddress}, r.From, r.To)
		if err != nil {
			return fmt.Errorf("backfill failed at blocks %d-%d: %w", r.From, r.To, err)
		}

		if len(result.Transfers) > 0 {
			if err := s.transferRepo.BatchInsert(ctx, result.Transfers); err != nil {
				return fmt.Errorf("failed to insert backfill transfers: %w", err)
			}
		}

		s.logger.Info("Backfill progress",
			zap.String("token", tokenAddress),
			zap.Int("batch", i+1),
			zap.Int("total_batches", len(ranges)),
			zap.Int64("from", r.From),
			zap.Int64("to", r.To),
			zap.Int("transfers", len(result.Transfers)),
		)
	}

	s.logger.Info("Backfill completed",
		zap.String("token", tokenAddress),
		zap.Int64("from_block", fromBlock),
		zap.Int64("to_block", toBlock),
	)

	return nil
}

func (s *IndexerService) updateMetrics(blocks, transfers, lastBlock int64) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.BlocksIndexed += blocks
	s.metrics.TransfersIndexed += transfers
	s.metrics.LastIndexedBlock = lastBlock
}

func (s *IndexerService) incrementErrorCount() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.ErrorCount++
}
