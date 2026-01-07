package testutil

import (
	"context"
	"errors"
	"sync"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
	"github.com/bimakw/chain-indexer/internal/domain/repositories"
)

// MockTransferRepository is a mock implementation of TransferRepository
type MockTransferRepository struct {
	mu        sync.RWMutex
	transfers []entities.Transfer

	// Function hooks for custom behavior
	GetByFilterFunc      func(ctx context.Context, filter entities.TransferFilter) ([]entities.Transfer, error)
	GetCountFunc         func(ctx context.Context, filter entities.TransferFilter) (int64, error)
	BatchInsertFunc      func(ctx context.Context, transfers []entities.Transfer) error
	GetLatestBlockFunc   func(ctx context.Context, tokenAddress string) (int64, error)
	GetTokenStatsFunc    func(ctx context.Context, tokenAddress string) (*repositories.TokenStatsResult, error)
	GetTopHoldersFunc    func(ctx context.Context, tokenAddress string, limit int) ([]repositories.HolderBalance, error)
	GetHolderBalanceFunc func(ctx context.Context, tokenAddress, holderAddress string) (*repositories.HolderBalance, error)

	// Call tracking
	Calls []MockCall
}

type MockCall struct {
	Method string
	Args   []interface{}
}

func NewMockTransferRepository() *MockTransferRepository {
	return &MockTransferRepository{
		transfers: make([]entities.Transfer, 0),
		Calls:     make([]MockCall, 0),
	}
}

func (m *MockTransferRepository) GetByFilter(ctx context.Context, filter entities.TransferFilter) ([]entities.Transfer, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "GetByFilter", Args: []interface{}{filter}})
	m.mu.Unlock()

	if m.GetByFilterFunc != nil {
		return m.GetByFilterFunc(ctx, filter)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simple filtering implementation
	result := make([]entities.Transfer, 0)
	for _, t := range m.transfers {
		if filter.TokenAddress != nil && t.TokenAddress != *filter.TokenAddress {
			continue
		}
		if filter.FromAddress != nil && t.FromAddress != *filter.FromAddress {
			continue
		}
		if filter.ToAddress != nil && t.ToAddress != *filter.ToAddress {
			continue
		}
		if filter.Address != nil && t.FromAddress != *filter.Address && t.ToAddress != *filter.Address {
			continue
		}
		if filter.FromBlock != nil && t.BlockNumber < *filter.FromBlock {
			continue
		}
		if filter.ToBlock != nil && t.BlockNumber > *filter.ToBlock {
			continue
		}
		result = append(result, t)
	}

	// Apply pagination
	start := filter.Offset
	if start > len(result) {
		return []entities.Transfer{}, nil
	}
	end := start + filter.Limit
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

func (m *MockTransferRepository) GetCount(ctx context.Context, filter entities.TransferFilter) (int64, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "GetCount", Args: []interface{}{filter}})
	m.mu.Unlock()

	if m.GetCountFunc != nil {
		return m.GetCountFunc(ctx, filter)
	}

	transfers, err := m.GetByFilter(ctx, entities.TransferFilter{
		TokenAddress: filter.TokenAddress,
		FromAddress:  filter.FromAddress,
		ToAddress:    filter.ToAddress,
		Address:      filter.Address,
		FromBlock:    filter.FromBlock,
		ToBlock:      filter.ToBlock,
		FromTime:     filter.FromTime,
		ToTime:       filter.ToTime,
		Limit:        1000000,
		Offset:       0,
	})
	if err != nil {
		return 0, err
	}
	return int64(len(transfers)), nil
}

func (m *MockTransferRepository) BatchInsert(ctx context.Context, transfers []entities.Transfer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, MockCall{Method: "BatchInsert", Args: []interface{}{transfers}})

	if m.BatchInsertFunc != nil {
		return m.BatchInsertFunc(ctx, transfers)
	}

	m.transfers = append(m.transfers, transfers...)
	return nil
}

func (m *MockTransferRepository) GetLatestBlock(ctx context.Context, tokenAddress string) (int64, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "GetLatestBlock", Args: []interface{}{tokenAddress}})
	m.mu.Unlock()

	if m.GetLatestBlockFunc != nil {
		return m.GetLatestBlockFunc(ctx, tokenAddress)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var latest int64 = 0
	for _, t := range m.transfers {
		if t.TokenAddress == tokenAddress && t.BlockNumber > latest {
			latest = t.BlockNumber
		}
	}
	return latest, nil
}

func (m *MockTransferRepository) GetTokenStats(ctx context.Context, tokenAddress string) (*repositories.TokenStatsResult, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "GetTokenStats", Args: []interface{}{tokenAddress}})
	m.mu.Unlock()

	if m.GetTokenStatsFunc != nil {
		return m.GetTokenStatsFunc(ctx, tokenAddress)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Default mock implementation - count transfers for the token
	var count int64
	uniqueFrom := make(map[string]bool)
	uniqueTo := make(map[string]bool)
	for _, t := range m.transfers {
		if t.TokenAddress == tokenAddress {
			count++
			uniqueFrom[t.FromAddress] = true
			uniqueTo[t.ToAddress] = true
		}
	}

	return &repositories.TokenStatsResult{
		TotalTransfers:  count,
		UniqueFromAddrs: int64(len(uniqueFrom)),
		UniqueToAddrs:   int64(len(uniqueTo)),
		TotalVolume:     "0",
		Transfers24h:    0,
		Volume24h:       "0",
		Transfers7d:     0,
		Volume7d:        "0",
	}, nil
}

func (m *MockTransferRepository) GetTopHolders(ctx context.Context, tokenAddress string, limit int) ([]repositories.HolderBalance, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "GetTopHolders", Args: []interface{}{tokenAddress, limit}})
	m.mu.Unlock()

	if m.GetTopHoldersFunc != nil {
		return m.GetTopHoldersFunc(ctx, tokenAddress, limit)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate balances from transfers
	balances := make(map[string]int64)
	for _, t := range m.transfers {
		if t.TokenAddress == tokenAddress {
			balances[t.ToAddress]++
			balances[t.FromAddress]--
		}
	}

	// Build result
	var result []repositories.HolderBalance
	rank := 1
	for addr, bal := range balances {
		if bal > 0 {
			result = append(result, repositories.HolderBalance{
				Address: addr,
				Balance: "1000000000000000000", // Mock balance
				Rank:    rank,
			})
			rank++
			if rank > limit {
				break
			}
		}
	}

	return result, nil
}

func (m *MockTransferRepository) GetHolderBalance(ctx context.Context, tokenAddress, holderAddress string) (*repositories.HolderBalance, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "GetHolderBalance", Args: []interface{}{tokenAddress, holderAddress}})
	m.mu.Unlock()

	if m.GetHolderBalanceFunc != nil {
		return m.GetHolderBalanceFunc(ctx, tokenAddress, holderAddress)
	}

	return &repositories.HolderBalance{
		Address: holderAddress,
		Balance: "1000000000000000000",
		Rank:    1,
	}, nil
}

// AddTransfers adds transfers to the mock store
func (m *MockTransferRepository) AddTransfers(transfers ...entities.Transfer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transfers = append(m.transfers, transfers...)
}

// Reset clears all stored data and calls
func (m *MockTransferRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transfers = make([]entities.Transfer, 0)
	m.Calls = make([]MockCall, 0)
}

// MockTokenRepository is a mock implementation of TokenRepository
type MockTokenRepository struct {
	mu     sync.RWMutex
	tokens map[string]*entities.Token

	// Function hooks
	GetByAddressFunc    func(ctx context.Context, address string) (*entities.Token, error)
	GetAllFunc          func(ctx context.Context) ([]entities.Token, error)
	GetAllPaginatedFunc func(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]*entities.Token, int64, error)
	CountFunc           func(ctx context.Context) (int64, error)
	UpsertFunc          func(ctx context.Context, token *entities.Token) error
	UpdateStatsFunc     func(ctx context.Context, address string, transferCount int64, lastBlock int64) error

	Calls []MockCall
}

func NewMockTokenRepository() *MockTokenRepository {
	return &MockTokenRepository{
		tokens: make(map[string]*entities.Token),
		Calls:  make([]MockCall, 0),
	}
}

func (m *MockTokenRepository) GetByAddress(ctx context.Context, address string) (*entities.Token, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "GetByAddress", Args: []interface{}{address}})
	m.mu.Unlock()

	if m.GetByAddressFunc != nil {
		return m.GetByAddressFunc(ctx, address)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if token, ok := m.tokens[address]; ok {
		return token, nil
	}
	return nil, nil
}

func (m *MockTokenRepository) GetAll(ctx context.Context) ([]entities.Token, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "GetAll", Args: nil})
	m.mu.Unlock()

	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]entities.Token, 0, len(m.tokens))
	for _, token := range m.tokens {
		result = append(result, *token)
	}
	return result, nil
}

func (m *MockTokenRepository) GetAllPaginated(ctx context.Context, limit, offset int, sortBy, sortOrder string) ([]*entities.Token, int64, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "GetAllPaginated", Args: []interface{}{limit, offset, sortBy, sortOrder}})
	m.mu.Unlock()

	if m.GetAllPaginatedFunc != nil {
		return m.GetAllPaginatedFunc(ctx, limit, offset, sortBy, sortOrder)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*entities.Token, 0, len(m.tokens))
	for _, token := range m.tokens {
		result = append(result, token)
	}

	total := int64(len(result))

	// Apply pagination
	start := offset
	if start > len(result) {
		return []*entities.Token{}, total, nil
	}
	end := start + limit
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], total, nil
}

func (m *MockTokenRepository) Count(ctx context.Context) (int64, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "Count", Args: nil})
	m.mu.Unlock()

	if m.CountFunc != nil {
		return m.CountFunc(ctx)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return int64(len(m.tokens)), nil
}

func (m *MockTokenRepository) Upsert(ctx context.Context, token *entities.Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, MockCall{Method: "Upsert", Args: []interface{}{token}})

	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, token)
	}

	m.tokens[token.Address] = token
	return nil
}

func (m *MockTokenRepository) UpdateStats(ctx context.Context, address string, transferCount int64, lastBlock int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, MockCall{Method: "UpdateStats", Args: []interface{}{address, transferCount, lastBlock}})

	if m.UpdateStatsFunc != nil {
		return m.UpdateStatsFunc(ctx, address, transferCount, lastBlock)
	}

	if token, ok := m.tokens[address]; ok {
		token.TotalIndexedTransfers += transferCount
		if token.LastSeenBlock == nil || lastBlock > *token.LastSeenBlock {
			token.LastSeenBlock = &lastBlock
		}
	}
	return nil
}

// AddToken adds a token to the mock store
func (m *MockTokenRepository) AddToken(token *entities.Token) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token.Address] = token
}

// Reset clears all stored data and calls
func (m *MockTokenRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens = make(map[string]*entities.Token)
	m.Calls = make([]MockCall, 0)
}

// MockIndexerStateRepository is a mock implementation of IndexerStateRepository
type MockIndexerStateRepository struct {
	mu     sync.RWMutex
	states map[string]*entities.IndexerState

	// Function hooks
	GetFunc             func(ctx context.Context, tokenAddress string) (*entities.IndexerState, error)
	UpsertFunc          func(ctx context.Context, state *entities.IndexerState) error
	UpdateLastBlockFunc func(ctx context.Context, tokenAddress string, blockNumber int64) error
	SetBackfillingFunc  func(ctx context.Context, tokenAddress string, isBackfilling bool, fromBlock, toBlock *int64) error

	Calls []MockCall
}

func NewMockIndexerStateRepository() *MockIndexerStateRepository {
	return &MockIndexerStateRepository{
		states: make(map[string]*entities.IndexerState),
		Calls:  make([]MockCall, 0),
	}
}

func (m *MockIndexerStateRepository) Get(ctx context.Context, tokenAddress string) (*entities.IndexerState, error) {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "Get", Args: []interface{}{tokenAddress}})
	m.mu.Unlock()

	if m.GetFunc != nil {
		return m.GetFunc(ctx, tokenAddress)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if state, ok := m.states[tokenAddress]; ok {
		return state, nil
	}
	return nil, nil
}

func (m *MockIndexerStateRepository) Upsert(ctx context.Context, state *entities.IndexerState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, MockCall{Method: "Upsert", Args: []interface{}{state}})

	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, state)
	}

	m.states[state.TokenAddress] = state
	return nil
}

func (m *MockIndexerStateRepository) UpdateLastBlock(ctx context.Context, tokenAddress string, blockNumber int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, MockCall{Method: "UpdateLastBlock", Args: []interface{}{tokenAddress, blockNumber}})

	if m.UpdateLastBlockFunc != nil {
		return m.UpdateLastBlockFunc(ctx, tokenAddress, blockNumber)
	}

	if state, ok := m.states[tokenAddress]; ok {
		state.LastIndexedBlock = blockNumber
	}
	return nil
}

func (m *MockIndexerStateRepository) SetBackfilling(ctx context.Context, tokenAddress string, isBackfilling bool, fromBlock, toBlock *int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Calls = append(m.Calls, MockCall{Method: "SetBackfilling", Args: []interface{}{tokenAddress, isBackfilling, fromBlock, toBlock}})

	if m.SetBackfillingFunc != nil {
		return m.SetBackfillingFunc(ctx, tokenAddress, isBackfilling, fromBlock, toBlock)
	}

	if state, ok := m.states[tokenAddress]; ok {
		state.IsBackfilling = isBackfilling
		state.BackfillFromBlock = fromBlock
		state.BackfillToBlock = toBlock
	}
	return nil
}

// AddState adds a state to the mock store
func (m *MockIndexerStateRepository) AddState(state *entities.IndexerState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[state.TokenAddress] = state
}

// Reset clears all stored data and calls
func (m *MockIndexerStateRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states = make(map[string]*entities.IndexerState)
	m.Calls = make([]MockCall, 0)
}

// MockHealthChecker is a mock implementation of HealthChecker
type MockHealthChecker struct {
	mu sync.RWMutex

	Healthy bool
	Error   error
	Calls   []MockCall
}

func NewMockHealthChecker(healthy bool) *MockHealthChecker {
	var err error
	if !healthy {
		err = errors.New("health check failed")
	}
	return &MockHealthChecker{
		Healthy: healthy,
		Error:   err,
		Calls:   make([]MockCall, 0),
	}
}

func (m *MockHealthChecker) HealthCheck(ctx context.Context) error {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockCall{Method: "HealthCheck", Args: nil})
	m.mu.Unlock()

	return m.Error
}

func (m *MockHealthChecker) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Healthy = healthy
	if healthy {
		m.Error = nil
	} else {
		m.Error = errors.New("health check failed")
	}
}
