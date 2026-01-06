/*
 * Copyright (c) 2024 Bima Kharisma Wicaksana
 * GitHub: https://github.com/bimakw
 *
 * Licensed under MIT License with Attribution Requirement.
 * See LICENSE file for details.
 */

package ethereum

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

// TokenMetadata holds ERC-20 token metadata
type TokenMetadata struct {
	Name     string
	Symbol   string
	Decimals uint8
}

// MetadataFetcher fetches ERC-20 token metadata via eth_call
type MetadataFetcher struct {
	client *Client
	logger *zap.Logger
}

// NewMetadataFetcher creates a new metadata fetcher
func NewMetadataFetcher(client *Client, logger *zap.Logger) *MetadataFetcher {
	return &MetadataFetcher{
		client: client,
		logger: logger,
	}
}

// ERC-20 function selectors (first 4 bytes of keccak256 hash)
var (
	// name() -> 0x06fdde03
	nameSig = common.FromHex("0x06fdde03")
	// symbol() -> 0x95d89b41
	symbolSig = common.FromHex("0x95d89b41")
	// decimals() -> 0x313ce567
	decimalsSig = common.FromHex("0x313ce567")
)

// FetchMetadata fetches token metadata for a given contract address
func (f *MetadataFetcher) FetchMetadata(ctx context.Context, tokenAddress string) (*TokenMetadata, error) {
	addr := common.HexToAddress(tokenAddress)

	name, err := f.fetchName(ctx, addr)
	if err != nil {
		f.logger.Warn("Failed to fetch token name, using fallback",
			zap.String("token", tokenAddress),
			zap.Error(err),
		)
		name = "Unknown"
	}

	symbol, err := f.fetchSymbol(ctx, addr)
	if err != nil {
		f.logger.Warn("Failed to fetch token symbol, using fallback",
			zap.String("token", tokenAddress),
			zap.Error(err),
		)
		symbol = "UNK"
	}

	decimals, err := f.fetchDecimals(ctx, addr)
	if err != nil {
		f.logger.Warn("Failed to fetch token decimals, using fallback",
			zap.String("token", tokenAddress),
			zap.Error(err),
		)
		decimals = 18
	}

	return &TokenMetadata{
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}, nil
}

// fetchName fetches token name via eth_call
func (f *MetadataFetcher) fetchName(ctx context.Context, addr common.Address) (string, error) {
	result, err := f.client.CallContract(ctx, addr, nameSig)
	if err != nil {
		return "", err
	}
	return decodeStringOrBytes32(result)
}

// fetchSymbol fetches token symbol via eth_call
func (f *MetadataFetcher) fetchSymbol(ctx context.Context, addr common.Address) (string, error) {
	result, err := f.client.CallContract(ctx, addr, symbolSig)
	if err != nil {
		return "", err
	}
	return decodeStringOrBytes32(result)
}

// fetchDecimals fetches token decimals via eth_call
func (f *MetadataFetcher) fetchDecimals(ctx context.Context, addr common.Address) (uint8, error) {
	result, err := f.client.CallContract(ctx, addr, decimalsSig)
	if err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, fmt.Errorf("empty result for decimals")
	}

	// Decimals returns uint8, but padded to 32 bytes
	if len(result) < 32 {
		return 0, fmt.Errorf("invalid decimals response length: %d", len(result))
	}

	// Take the last byte for uint8
	return result[31], nil
}

// decodeStringOrBytes32 decodes a response that could be either:
// 1. ABI-encoded string: offset (32 bytes) + length (32 bytes) + data (padded to 32 bytes)
// 2. bytes32: raw 32 bytes (e.g., MKR token)
func decodeStringOrBytes32(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty data")
	}

	// If data is less than 32 bytes, invalid
	if len(data) < 32 {
		return "", fmt.Errorf("data too short: %d bytes", len(data))
	}

	// Try to decode as ABI-encoded string first
	// Check if first 32 bytes could be an offset (typically 0x20 = 32)
	if len(data) >= 64 {
		offset := new(big.Int).SetBytes(data[:32])
		if offset.Uint64() == 32 {
			// This looks like an ABI-encoded string
			length := new(big.Int).SetBytes(data[32:64])
			strLen := int(length.Uint64())

			// Handle empty string (length = 0)
			if strLen == 0 {
				return "", nil
			}

			if len(data) >= 64+strLen {
				strData := data[64 : 64+strLen]
				return strings.TrimRight(string(strData), "\x00"), nil
			}
		}
	}

	// Fallback: treat as bytes32
	// Remove trailing null bytes
	result := bytes.TrimRight(data[:32], "\x00")

	// Check if result is printable ASCII
	if isPrintableASCII(result) {
		return string(result), nil
	}

	// Return hex representation if not printable
	return "0x" + hex.EncodeToString(data[:32]), nil
}

// isPrintableASCII checks if all bytes are printable ASCII characters
func isPrintableASCII(data []byte) bool {
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return len(data) > 0
}

// FetchMetadataBatch fetches metadata for multiple tokens
func (f *MetadataFetcher) FetchMetadataBatch(ctx context.Context, tokenAddresses []string) (map[string]*TokenMetadata, error) {
	results := make(map[string]*TokenMetadata)

	for _, addr := range tokenAddresses {
		normalizedAddr := strings.ToLower(addr)
		metadata, err := f.FetchMetadata(ctx, normalizedAddr)
		if err != nil {
			f.logger.Warn("Failed to fetch metadata for token",
				zap.String("token", addr),
				zap.Error(err),
			)
			// Use fallback values
			metadata = &TokenMetadata{
				Name:     "Unknown",
				Symbol:   "UNK",
				Decimals: 18,
			}
		}
		results[normalizedAddr] = metadata
	}

	return results, nil
}
