package entities

import (
	"math/big"
	"time"
)

type TokenHolding struct {
	TokenAddress string   `json:"token_address"`
	TokenName    string   `json:"token_name"`
	TokenSymbol  string   `json:"token_symbol"`
	Decimals     int      `json:"decimals"`
	Balance      *big.Int `json:"-"`
	BalanceStr   string   `json:"balance"`           // Raw balance (wei)
	BalanceHuman string   `json:"balance_formatted"` // Human readable (with decimals)
}

type WalletPortfolio struct {
	WalletAddress string         `json:"wallet_address"`
	Holdings      []TokenHolding `json:"holdings"`
	TotalTokens   int            `json:"total_tokens"`
	LastUpdated   time.Time      `json:"last_updated"`
}

type PortfolioFilter struct {
	WalletAddress string
	TokenAddress  *string  // Optional: filter specific token
	MinBalance    *big.Int // Optional: minimum balance filter
	IncludeZero   bool     // Include zero balance tokens
}
