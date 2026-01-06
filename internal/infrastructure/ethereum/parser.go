package ethereum

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/bimakw/chain-indexer/internal/domain/entities"
)

// TransferEventSignature is the keccak256 hash of Transfer(address,address,uint256)
var TransferEventSignature = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

// ParseTransferEvent parses a raw log into a Transfer entity
func ParseTransferEvent(log types.Log, blockTimestamp time.Time) (*entities.Transfer, error) {
	// Validate log has correct topic structure
	if len(log.Topics) != 3 {
		return nil, fmt.Errorf("invalid number of topics: expected 3, got %d", len(log.Topics))
	}

	// Verify this is a Transfer event
	if log.Topics[0] != TransferEventSignature {
		return nil, fmt.Errorf("not a Transfer event")
	}

	// Parse addresses from topics (indexed parameters)
	// Topics[1] = from address (padded to 32 bytes)
	// Topics[2] = to address (padded to 32 bytes)
	fromAddress := common.BytesToAddress(log.Topics[1].Bytes())
	toAddress := common.BytesToAddress(log.Topics[2].Bytes())

	// Parse value from data (non-indexed parameter)
	if len(log.Data) != 32 {
		return nil, fmt.Errorf("invalid data length: expected 32, got %d", len(log.Data))
	}
	value := new(big.Int).SetBytes(log.Data)

	return &entities.Transfer{
		TxHash:         log.TxHash.Hex(),
		LogIndex:       int(log.Index),
		BlockNumber:    int64(log.BlockNumber),
		BlockTimestamp: blockTimestamp,
		TokenAddress:   strings.ToLower(log.Address.Hex()),
		FromAddress:    strings.ToLower(fromAddress.Hex()),
		ToAddress:      strings.ToLower(toAddress.Hex()),
		Value:          value,
		ValueString:    value.String(),
	}, nil
}

// ParseTransferLogs parses multiple logs into Transfer entities
// Returns parsed transfers and a list of failed log indices
func ParseTransferLogs(logs []types.Log, blockTimestamps map[uint64]time.Time) ([]entities.Transfer, []int) {
	transfers := make([]entities.Transfer, 0, len(logs))
	failedIndices := make([]int, 0)

	for i, log := range logs {
		timestamp, ok := blockTimestamps[log.BlockNumber]
		if !ok {
			failedIndices = append(failedIndices, i)
			continue
		}

		transfer, err := ParseTransferEvent(log, timestamp)
		if err != nil {
			failedIndices = append(failedIndices, i)
			continue
		}

		transfers = append(transfers, *transfer)
	}

	return transfers, failedIndices
}

// IsTransferEvent checks if a log is a Transfer event
func IsTransferEvent(log types.Log) bool {
	return len(log.Topics) == 3 && log.Topics[0] == TransferEventSignature
}
