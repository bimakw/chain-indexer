package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/config"
)

// Client wraps the Ethereum client with retry logic and utilities
type Client struct {
	client  *ethclient.Client
	config  config.EthereumConfig
	logger  *zap.Logger
	chainID *big.Int
}

// NewClient creates a new Ethereum client
func NewClient(cfg config.EthereumConfig, logger *zap.Logger) (*Client, error) {
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
	defer cancel()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	if chainID.Int64() != cfg.ChainID {
		return nil, fmt.Errorf("chain ID mismatch: expected %d, got %d", cfg.ChainID, chainID.Int64())
	}

	logger.Info("Connected to Ethereum node",
		zap.String("rpc_url", cfg.RPCURL),
		zap.Int64("chain_id", chainID.Int64()),
	)

	return &Client{
		client:  client,
		config:  cfg,
		logger:  logger,
		chainID: chainID,
	}, nil
}

// Close closes the Ethereum client connection
func (c *Client) Close() {
	c.client.Close()
}

// GetLatestBlockNumber returns the latest block number
func (c *Client) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	var blockNumber uint64
	var err error

	for i := 0; i <= c.config.MaxRetries; i++ {
		blockNumber, err = c.client.BlockNumber(ctx)
		if err == nil {
			return blockNumber, nil
		}

		c.logger.Warn("Failed to get latest block number, retrying",
			zap.Int("attempt", i+1),
			zap.Error(err),
		)

		if i < c.config.MaxRetries {
			time.Sleep(c.config.RetryDelay)
		}
	}

	return 0, fmt.Errorf("failed to get latest block number after %d retries: %w", c.config.MaxRetries, err)
}

// GetBlockByNumber returns a block by its number
func (c *Client) GetBlockByNumber(ctx context.Context, blockNumber *big.Int) (*types.Block, error) {
	var block *types.Block
	var err error

	for i := 0; i <= c.config.MaxRetries; i++ {
		block, err = c.client.BlockByNumber(ctx, blockNumber)
		if err == nil {
			return block, nil
		}

		c.logger.Warn("Failed to get block, retrying",
			zap.String("block_number", blockNumber.String()),
			zap.Int("attempt", i+1),
			zap.Error(err),
		)

		if i < c.config.MaxRetries {
			time.Sleep(c.config.RetryDelay)
		}
	}

	return nil, fmt.Errorf("failed to get block %s after %d retries: %w", blockNumber.String(), c.config.MaxRetries, err)
}

// GetLogs retrieves logs matching the filter query
func (c *Client) GetLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	var logs []types.Log
	var err error

	for i := 0; i <= c.config.MaxRetries; i++ {
		logs, err = c.client.FilterLogs(ctx, query)
		if err == nil {
			return logs, nil
		}

		c.logger.Warn("Failed to get logs, retrying",
			zap.Int("attempt", i+1),
			zap.Error(err),
		)

		if i < c.config.MaxRetries {
			time.Sleep(c.config.RetryDelay)
		}
	}

	return nil, fmt.Errorf("failed to get logs after %d retries: %w", c.config.MaxRetries, err)
}

// GetBlockTimestamp returns the timestamp of a block
func (c *Client) GetBlockTimestamp(ctx context.Context, blockNumber uint64) (time.Time, error) {
	block, err := c.GetBlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(block.Time()), 0), nil
}

// BuildFilterQuery builds a filter query for ERC-20 Transfer events
func (c *Client) BuildFilterQuery(fromBlock, toBlock *big.Int, addresses []common.Address) ethereum.FilterQuery {
	// ERC-20 Transfer event signature: Transfer(address,address,uint256)
	// keccak256("Transfer(address,address,uint256)") = 0xddf252ad...
	transferEventSig := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	return ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: addresses,
		Topics: [][]common.Hash{
			{transferEventSig},
		},
	}
}

// ChainID returns the chain ID
func (c *Client) ChainID() *big.Int {
	return c.chainID
}

// EthClient returns the underlying ethclient for advanced operations
func (c *Client) EthClient() *ethclient.Client {
	return c.client
}
