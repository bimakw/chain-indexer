package ethereum

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestTransferEventSignature(t *testing.T) {
	// The keccak256 hash of "Transfer(address,address,uint256)"
	expected := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	if TransferEventSignature != expected {
		t.Errorf("TransferEventSignature mismatch: expected %s, got %s", expected.Hex(), TransferEventSignature.Hex())
	}
}

func TestParseTransferEvent_Success(t *testing.T) {
	// Create a valid Transfer event log
	fromAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	toAddr := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	tokenAddr := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7") // USDT
	txHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")

	// Value: 1000000 (1 USDT with 6 decimals)
	value := big.NewInt(1000000)
	valueBytes := common.LeftPadBytes(value.Bytes(), 32)

	log := types.Log{
		Address: tokenAddr,
		Topics: []common.Hash{
			TransferEventSignature,
			common.BytesToHash(fromAddr.Bytes()),
			common.BytesToHash(toAddr.Bytes()),
		},
		Data:        valueBytes,
		BlockNumber: 12345678,
		TxHash:      txHash,
		Index:       5,
	}

	blockTimestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	transfer, err := ParseTransferEvent(log, blockTimestamp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields
	if transfer.TxHash != txHash.Hex() {
		t.Errorf("TxHash mismatch: expected %s, got %s", txHash.Hex(), transfer.TxHash)
	}
	if transfer.LogIndex != 5 {
		t.Errorf("LogIndex mismatch: expected 5, got %d", transfer.LogIndex)
	}
	if transfer.BlockNumber != 12345678 {
		t.Errorf("BlockNumber mismatch: expected 12345678, got %d", transfer.BlockNumber)
	}
	if !transfer.BlockTimestamp.Equal(blockTimestamp) {
		t.Errorf("BlockTimestamp mismatch: expected %v, got %v", blockTimestamp, transfer.BlockTimestamp)
	}
	if transfer.TokenAddress != "0xdac17f958d2ee523a2206206994597c13d831ec7" {
		t.Errorf("TokenAddress mismatch: expected lowercase, got %s", transfer.TokenAddress)
	}
	if transfer.FromAddress != "0x1234567890123456789012345678901234567890" {
		t.Errorf("FromAddress mismatch: got %s", transfer.FromAddress)
	}
	if transfer.ToAddress != "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd" {
		t.Errorf("ToAddress mismatch: got %s", transfer.ToAddress)
	}
	if transfer.Value.Cmp(value) != 0 {
		t.Errorf("Value mismatch: expected %s, got %s", value.String(), transfer.Value.String())
	}
	if transfer.ValueString != "1000000" {
		t.Errorf("ValueString mismatch: expected 1000000, got %s", transfer.ValueString)
	}
}

func TestParseTransferEvent_LargeValue(t *testing.T) {
	// Test with a large value (e.g., 1 billion tokens with 18 decimals)
	largeValue := new(big.Int)
	largeValue.SetString("1000000000000000000000000000", 10) // 1 billion * 10^18

	valueBytes := common.LeftPadBytes(largeValue.Bytes(), 32)

	log := types.Log{
		Address: common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		Topics: []common.Hash{
			TransferEventSignature,
			common.BytesToHash(common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()),
			common.BytesToHash(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()),
		},
		Data:        valueBytes,
		BlockNumber: 1,
		TxHash:      common.HexToHash("0x0"),
		Index:       0,
	}

	transfer, err := ParseTransferEvent(log, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transfer.Value.Cmp(largeValue) != 0 {
		t.Errorf("Large value mismatch: expected %s, got %s", largeValue.String(), transfer.Value.String())
	}
}

func TestParseTransferEvent_ZeroValue(t *testing.T) {
	valueBytes := make([]byte, 32)

	log := types.Log{
		Address: common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		Topics: []common.Hash{
			TransferEventSignature,
			common.BytesToHash(common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()),
			common.BytesToHash(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()),
		},
		Data:        valueBytes,
		BlockNumber: 1,
		TxHash:      common.HexToHash("0x0"),
		Index:       0,
	}

	transfer, err := ParseTransferEvent(log, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transfer.Value.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Zero value mismatch: expected 0, got %s", transfer.Value.String())
	}
	if transfer.ValueString != "0" {
		t.Errorf("ValueString mismatch: expected '0', got %s", transfer.ValueString)
	}
}

func TestParseTransferEvent_InvalidTopicsCount(t *testing.T) {
	tests := []struct {
		name       string
		topicsLen  int
		errContain string
	}{
		{"no topics", 0, "invalid number of topics"},
		{"one topic", 1, "invalid number of topics"},
		{"two topics", 2, "invalid number of topics"},
		{"four topics", 4, "invalid number of topics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topics := make([]common.Hash, tt.topicsLen)
			if tt.topicsLen > 0 {
				topics[0] = TransferEventSignature
			}

			log := types.Log{
				Address:     common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
				Topics:      topics,
				Data:        make([]byte, 32),
				BlockNumber: 1,
			}

			_, err := ParseTransferEvent(log, time.Now())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != "invalid number of topics: expected 3, got "+string(rune('0'+tt.topicsLen)) {
				// Just check it's an error about topics
				if !contains(err.Error(), "topics") {
					t.Errorf("error should mention topics: %v", err)
				}
			}
		})
	}
}

func TestParseTransferEvent_WrongEventSignature(t *testing.T) {
	// Use Approval event signature instead
	approvalSig := common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")

	log := types.Log{
		Address: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
		Topics: []common.Hash{
			approvalSig,
			common.BytesToHash(common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()),
			common.BytesToHash(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()),
		},
		Data:        make([]byte, 32),
		BlockNumber: 1,
	}

	_, err := ParseTransferEvent(log, time.Now())
	if err == nil {
		t.Fatal("expected error for wrong event signature")
	}
	if !contains(err.Error(), "not a Transfer event") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseTransferEvent_InvalidDataLength(t *testing.T) {
	tests := []struct {
		name    string
		dataLen int
	}{
		{"empty data", 0},
		{"short data", 16},
		{"long data", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := types.Log{
				Address: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
				Topics: []common.Hash{
					TransferEventSignature,
					common.BytesToHash(common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()),
					common.BytesToHash(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()),
				},
				Data:        make([]byte, tt.dataLen),
				BlockNumber: 1,
			}

			_, err := ParseTransferEvent(log, time.Now())
			if err == nil {
				t.Fatal("expected error for invalid data length")
			}
			if !contains(err.Error(), "invalid data length") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseTransferEvent_AddressNormalization(t *testing.T) {
	// Use mixed case addresses
	fromAddr := common.HexToAddress("0xABCDEF1234567890ABCDEF1234567890ABCDEF12")
	toAddr := common.HexToAddress("0x123456ABCDEF123456ABCDEF123456ABCDEF1234")
	tokenAddr := common.HexToAddress("0xDAC17F958D2EE523A2206206994597C13D831EC7")

	log := types.Log{
		Address: tokenAddr,
		Topics: []common.Hash{
			TransferEventSignature,
			common.BytesToHash(fromAddr.Bytes()),
			common.BytesToHash(toAddr.Bytes()),
		},
		Data:        make([]byte, 32),
		BlockNumber: 1,
		TxHash:      common.HexToHash("0x0"),
		Index:       0,
	}

	transfer, err := ParseTransferEvent(log, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All addresses should be lowercase
	if transfer.TokenAddress != "0xdac17f958d2ee523a2206206994597c13d831ec7" {
		t.Errorf("TokenAddress should be lowercase: %s", transfer.TokenAddress)
	}
	if transfer.FromAddress != "0xabcdef1234567890abcdef1234567890abcdef12" {
		t.Errorf("FromAddress should be lowercase: %s", transfer.FromAddress)
	}
	if transfer.ToAddress != "0x123456abcdef123456abcdef123456abcdef1234" {
		t.Errorf("ToAddress should be lowercase: %s", transfer.ToAddress)
	}
}

func TestParseTransferLogs_Success(t *testing.T) {
	blockTimestamps := map[uint64]time.Time{
		100: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		101: time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		102: time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC),
	}

	logs := []types.Log{
		createValidTransferLog(100, 0),
		createValidTransferLog(101, 1),
		createValidTransferLog(102, 2),
	}

	transfers, failed := ParseTransferLogs(logs, blockTimestamps)

	if len(transfers) != 3 {
		t.Errorf("expected 3 transfers, got %d", len(transfers))
	}
	if len(failed) != 0 {
		t.Errorf("expected 0 failed, got %d", len(failed))
	}

	// Verify block numbers
	if transfers[0].BlockNumber != 100 {
		t.Errorf("first transfer block mismatch")
	}
	if transfers[1].BlockNumber != 101 {
		t.Errorf("second transfer block mismatch")
	}
	if transfers[2].BlockNumber != 102 {
		t.Errorf("third transfer block mismatch")
	}
}

func TestParseTransferLogs_MissingTimestamp(t *testing.T) {
	blockTimestamps := map[uint64]time.Time{
		100: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		// 101 is missing
		102: time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC),
	}

	logs := []types.Log{
		createValidTransferLog(100, 0),
		createValidTransferLog(101, 1), // This should fail
		createValidTransferLog(102, 2),
	}

	transfers, failed := ParseTransferLogs(logs, blockTimestamps)

	if len(transfers) != 2 {
		t.Errorf("expected 2 transfers, got %d", len(transfers))
	}
	if len(failed) != 1 {
		t.Errorf("expected 1 failed, got %d", len(failed))
	}
	if failed[0] != 1 {
		t.Errorf("expected failed index 1, got %d", failed[0])
	}
}

func TestParseTransferLogs_InvalidLog(t *testing.T) {
	blockTimestamps := map[uint64]time.Time{
		100: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		101: time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
	}

	validLog := createValidTransferLog(100, 0)
	invalidLog := types.Log{
		Address:     common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
		Topics:      []common.Hash{TransferEventSignature}, // Invalid: only 1 topic
		Data:        make([]byte, 32),
		BlockNumber: 101,
		Index:       1,
	}

	logs := []types.Log{validLog, invalidLog}

	transfers, failed := ParseTransferLogs(logs, blockTimestamps)

	if len(transfers) != 1 {
		t.Errorf("expected 1 transfer, got %d", len(transfers))
	}
	if len(failed) != 1 {
		t.Errorf("expected 1 failed, got %d", len(failed))
	}
	if failed[0] != 1 {
		t.Errorf("expected failed index 1, got %d", failed[0])
	}
}

func TestParseTransferLogs_Empty(t *testing.T) {
	blockTimestamps := map[uint64]time.Time{}
	logs := []types.Log{}

	transfers, failed := ParseTransferLogs(logs, blockTimestamps)

	if len(transfers) != 0 {
		t.Errorf("expected 0 transfers, got %d", len(transfers))
	}
	if len(failed) != 0 {
		t.Errorf("expected 0 failed, got %d", len(failed))
	}
}

func TestIsTransferEvent_Valid(t *testing.T) {
	log := createValidTransferLog(100, 0)
	if !IsTransferEvent(log) {
		t.Error("expected IsTransferEvent to return true for valid Transfer log")
	}
}

func TestIsTransferEvent_WrongSignature(t *testing.T) {
	log := types.Log{
		Topics: []common.Hash{
			common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"), // Approval
			common.BytesToHash(common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()),
			common.BytesToHash(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()),
		},
	}
	if IsTransferEvent(log) {
		t.Error("expected IsTransferEvent to return false for non-Transfer log")
	}
}

func TestIsTransferEvent_WrongTopicCount(t *testing.T) {
	tests := []struct {
		name     string
		topicLen int
		expected bool
	}{
		{"zero topics", 0, false},
		{"one topic", 1, false},
		{"two topics", 2, false},
		{"four topics", 4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topics := make([]common.Hash, tt.topicLen)
			if tt.topicLen > 0 {
				topics[0] = TransferEventSignature
			}

			log := types.Log{Topics: topics}
			if IsTransferEvent(log) != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, !tt.expected)
			}
		})
	}
}

// Helper functions

func createValidTransferLog(blockNumber uint64, index uint) types.Log {
	value := big.NewInt(1000000)
	valueBytes := common.LeftPadBytes(value.Bytes(), 32)

	return types.Log{
		Address: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
		Topics: []common.Hash{
			TransferEventSignature,
			common.BytesToHash(common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()),
			common.BytesToHash(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()),
		},
		Data:        valueBytes,
		BlockNumber: blockNumber,
		TxHash:      common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
		Index:       index,
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
