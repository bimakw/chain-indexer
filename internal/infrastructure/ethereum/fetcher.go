package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/bimakw/chain-indexer/internal/config"
	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

// Fetcher handles fetching and parsing blockchain data
type Fetcher struct {
	client  *Client
	config  config.IndexerConfig
	logger  *zap.Logger
}

// NewFetcher creates a new blockchain data fetcher
func NewFetcher(client *Client, cfg config.IndexerConfig, logger *zap.Logger) *Fetcher {
	return &Fetcher{
		client:  client,
		config:  cfg,
		logger:  logger,
	}
}

// FetchResult contains the result of fetching transfers
type FetchResult struct {
	Transfers      []entities.Transfer
	FromBlock      int64
	ToBlock        int64
	FailedLogCount int
}

// FetchTransfers fetches Transfer events for a range of blocks
func (f *Fetcher) FetchTransfers(ctx context.Context, tokenAddresses []string, fromBlock, toBlock int64) (*FetchResult, error) {
	// Convert addresses to common.Address
	addresses := make([]common.Address, len(tokenAddresses))
	for i, addr := range tokenAddresses {
		addresses[i] = common.HexToAddress(addr)
	}

	// Build and execute filter query
	query := f.client.BuildFilterQuery(
		big.NewInt(fromBlock),
		big.NewInt(toBlock),
		addresses,
	)

	f.logger.Debug("Fetching logs",
		zap.Int64("from_block", fromBlock),
		zap.Int64("to_block", toBlock),
		zap.Int("token_count", len(tokenAddresses)),
	)

	logs, err := f.client.GetLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}

	if len(logs) == 0 {
		return &FetchResult{
			Transfers: []entities.Transfer{},
			FromBlock: fromBlock,
			ToBlock:   toBlock,
		}, nil
	}

	// Collect unique block numbers and fetch timestamps concurrently
	blockNumbers := make(map[uint64]struct{})
	for _, log := range logs {
		blockNumbers[log.BlockNumber] = struct{}{}
	}

	blockTimestamps, err := f.fetchBlockTimestamps(ctx, blockNumbers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block timestamps: %w", err)
	}

	// Parse logs into transfers
	transfers, failedIndices := ParseTransferLogs(logs, blockTimestamps)

	if len(failedIndices) > 0 {
		f.logger.Warn("Failed to parse some logs",
			zap.Int("failed_count", len(failedIndices)),
			zap.Int("total_logs", len(logs)),
		)
	}

	f.logger.Info("Fetched transfers",
		zap.Int64("from_block", fromBlock),
		zap.Int64("to_block", toBlock),
		zap.Int("transfer_count", len(transfers)),
	)

	return &FetchResult{
		Transfers:      transfers,
		FromBlock:      fromBlock,
		ToBlock:        toBlock,
		FailedLogCount: len(failedIndices),
	}, nil
}

// fetchBlockTimestamps fetches timestamps for multiple blocks concurrently
func (f *Fetcher) fetchBlockTimestamps(ctx context.Context, blockNumbers map[uint64]struct{}) (map[uint64]time.Time, error) {
	timestamps := make(map[uint64]time.Time)
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(f.config.WorkerCount)

	for blockNum := range blockNumbers {
		blockNum := blockNum // capture
		g.Go(func() error {
			timestamp, err := f.client.GetBlockTimestamp(ctx, blockNum)
			if err != nil {
				return fmt.Errorf("failed to get timestamp for block %d: %w", blockNum, err)
			}

			mu.Lock()
			timestamps[blockNum] = timestamp
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return timestamps, nil
}

// GetSafeBlockNumber returns the latest block number minus confirmations
func (f *Fetcher) GetSafeBlockNumber(ctx context.Context) (int64, error) {
	latestBlock, err := f.client.GetLatestBlockNumber(ctx)
	if err != nil {
		return 0, err
	}

	safeBlock := int64(latestBlock) - int64(f.config.BlockConfirmations)
	if safeBlock < 0 {
		safeBlock = 0
	}

	return safeBlock, nil
}

// BlockRange represents a range of blocks to fetch
type BlockRange struct {
	From int64
	To   int64
}

// SplitBlockRange splits a range into batches
func SplitBlockRange(fromBlock, toBlock int64, batchSize int) []BlockRange {
	if fromBlock > toBlock {
		return nil
	}

	var ranges []BlockRange
	for current := fromBlock; current <= toBlock; current += int64(batchSize) {
		end := current + int64(batchSize) - 1
		if end > toBlock {
			end = toBlock
		}
		ranges = append(ranges, BlockRange{From: current, To: end})
	}

	return ranges
}
