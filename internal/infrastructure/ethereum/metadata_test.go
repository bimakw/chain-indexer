/*
 * Copyright (c) 2024 Bima Kharisma Wicaksana
 * GitHub: https://github.com/bimakw
 *
 * Licensed under MIT License with Attribution Requirement.
 * See LICENSE file for details.
 */

package ethereum

import (
	"encoding/hex"
	"testing"
)

func TestDecodeStringOrBytes32(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
		wantErr  bool
	}{
		{
			name: "ABI-encoded string - USDT",
			input: func() []byte {
				// offset = 32 (0x20)
				// length = 10 ("TetherUSD" is 9 chars, but let's use "Tether USD" = 10)
				// data = "Tether USD"
				data, _ := hex.DecodeString(
					"0000000000000000000000000000000000000000000000000000000000000020" + // offset = 32
						"000000000000000000000000000000000000000000000000000000000000000a" + // length = 10
						"5465746865722055534400000000000000000000000000000000000000000000", // "Tether USD" padded
				)
				return data
			}(),
			expected: "Tether USD",
			wantErr:  false,
		},
		{
			name: "ABI-encoded string - USDC",
			input: func() []byte {
				// "USD Coin" = 8 chars
				data, _ := hex.DecodeString(
					"0000000000000000000000000000000000000000000000000000000000000020" + // offset = 32
						"0000000000000000000000000000000000000000000000000000000000000008" + // length = 8
						"55534420436f696e000000000000000000000000000000000000000000000000", // "USD Coin" padded
				)
				return data
			}(),
			expected: "USD Coin",
			wantErr:  false,
		},
		{
			name: "bytes32 - MKR style",
			input: func() []byte {
				// MKR returns "Maker" as bytes32 (not ABI-encoded string)
				data, _ := hex.DecodeString(
					"4d616b6572000000000000000000000000000000000000000000000000000000", // "Maker" as bytes32
				)
				return data
			}(),
			expected: "Maker",
			wantErr:  false,
		},
		{
			name: "bytes32 - DAI style",
			input: func() []byte {
				// "Dai" as bytes32
				data, _ := hex.DecodeString(
					"4461690000000000000000000000000000000000000000000000000000000000", // "Dai" as bytes32
				)
				return data
			}(),
			expected: "Dai",
			wantErr:  false,
		},
		{
			name: "short symbol - ETH",
			input: func() []byte {
				// "ETH" as bytes32
				data, _ := hex.DecodeString(
					"4554480000000000000000000000000000000000000000000000000000000000", // "ETH"
				)
				return data
			}(),
			expected: "ETH",
			wantErr:  false,
		},
		{
			name: "ABI-encoded empty string",
			input: func() []byte {
				data, _ := hex.DecodeString(
					"0000000000000000000000000000000000000000000000000000000000000020" + // offset = 32
						"0000000000000000000000000000000000000000000000000000000000000000", // length = 0
				)
				return data
			}(),
			expected: "",
			wantErr:  false,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "short input (less than 32 bytes)",
			input:    []byte{0x01, 0x02, 0x03},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeStringOrBytes32(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsPrintableASCII(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "printable ASCII - letters",
			input:    []byte("Hello"),
			expected: true,
		},
		{
			name:     "printable ASCII - with numbers",
			input:    []byte("Token123"),
			expected: true,
		},
		{
			name:     "printable ASCII - with symbols",
			input:    []byte("USD-T_v2"),
			expected: true,
		},
		{
			name:     "contains null byte",
			input:    []byte("Test\x00Name"),
			expected: false,
		},
		{
			name:     "contains control character",
			input:    []byte("Test\x1FName"),
			expected: false,
		},
		{
			name:     "contains DEL character",
			input:    []byte("Test\x7FName"),
			expected: false,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: false,
		},
		{
			name:     "high ASCII (non-printable)",
			input:    []byte{0x80, 0x81},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPrintableASCII(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFunctionSelectors(t *testing.T) {
	// Verify that our function selectors are correct
	// These are the first 4 bytes of keccak256 hash of function signatures

	tests := []struct {
		name     string
		selector []byte
		expected string
	}{
		{
			name:     "name()",
			selector: nameSig,
			expected: "06fdde03",
		},
		{
			name:     "symbol()",
			selector: symbolSig,
			expected: "95d89b41",
		},
		{
			name:     "decimals()",
			selector: decimalsSig,
			expected: "313ce567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hex.EncodeToString(tt.selector)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
