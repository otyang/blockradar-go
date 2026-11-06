package blockradar

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// Amounts are decimal strings (e.g. "100.50") to keep full precision.
// Nullable string fields decode to "" when the API returns null.

// JSONSchema is a JSON Schema (draft-07) document describing a dynamic form.
// Kept raw so you can pass it straight to a validator or a frontend.
type JSONSchema = json.RawMessage

// AmountSide says which side of a conversion Amount refers to.
type AmountSide string

const (
	AmountSideSource AmountSide = "source" // Amount is the stablecoin budget (default)
	AmountSideTarget AmountSide = "target" // Amount is the fiat payout
)

// Page is a paginated list response.
type Page[T any] struct {
	Data  []T             `json:"data"`
	Meta  PaginationMeta  `json:"meta"`
	Links PaginationLinks `json:"links"`
}

type PaginationMeta struct {
	Total           int  `json:"total"`
	Page            int  `json:"page"`
	Limit           int  `json:"limit"`
	PageCount       int  `json:"pageCount"`
	HasPreviousPage bool `json:"hasPreviousPage"`
	HasNextPage     bool `json:"hasNextPage"`
}

type PaginationLinks struct {
	First    string `json:"first"`
	Previous string `json:"previous"`
	Next     string `json:"next"`
	Last     string `json:"last"`
}

// FiatTransaction is a fiat withdrawal or virtual-account deposit.
type FiatTransaction struct {
	ID                          string    `json:"id"`
	Reference                   string    `json:"reference"`
	SenderAddress               string    `json:"senderAddress"`
	RecipientAddress            string    `json:"recipientAddress"`
	TokenAddress                string    `json:"tokenAddress"`
	Amount                      string    `json:"amount"`
	AmountPaid                  string    `json:"amountPaid"`
	AmountUSD                   string    `json:"amountUSD"`
	RateUSD                     string    `json:"rateUSD"`
	Fee                         string    `json:"fee"`
	FeeUSD                      string    `json:"feeUSD"`
	Currency                    string    `json:"currency"`
	ToCurrency                  string    `json:"toCurrency"`
	Hash                        string    `json:"hash"`
	Confirmed                   bool      `json:"confirmed"`
	Status                      string    `json:"status"`
	ProcessingStatus            string    `json:"processingStatus"`
	ProcessingProviderReference string    `json:"processingProviderReference"`
	Type                        string    `json:"type"`
	Note                        string    `json:"note"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

// ---- v1 shared models (used across many endpoints) ----

// Amount is a decimal value. The API sends amounts as strings ("7.0") in most
// places and as JSON numbers in a few; Amount accepts both and keeps the text.
type Amount string

func (a *Amount) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*a = ""
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*a = Amount(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("blockradar: amount %s: %w", b, err)
	}
	*a = Amount(n.String())
	return nil
}

func (a Amount) String() string { return string(a) }

// List is a v1 paginated list: {"data": [...], "meta": {...}}.
type List[T any] struct {
	Data []T      `json:"data"`
	Meta ListMeta `json:"meta"`
}

type ListMeta struct {
	CurrentPage  int `json:"currentPage"`
	ItemCount    int `json:"itemCount"`
	ItemsPerPage int `json:"itemsPerPage"`
	TotalItems   int `json:"totalItems"`
	TotalPages   int `json:"totalPages"`
}

// PageParams paginates v1 list endpoints that accept page/limit.
type PageParams struct {
	Page  int
	Limit int
}

type Blockchain struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	Symbol          string    `json:"symbol"`
	DerivationPath  string    `json:"derivationPath"`
	TokenStandard   string    `json:"tokenStandard"`
	IsActive        bool      `json:"isActive"`
	IsEvmCompatible bool      `json:"isEvmCompatible"`
	IsL2            bool      `json:"isL2"`
	LogoURL         string    `json:"logoUrl"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Asset is a token on a blockchain (USDC on Base, USDT on Tron, ...).
type Asset struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Symbol     string      `json:"symbol"`
	Decimals   int         `json:"decimals"`
	Address    string      `json:"address"` // token contract address
	Standard   string      `json:"standard"`
	Network    string      `json:"network"` // "testnet" or "mainnet"
	IsActive   bool        `json:"isActive"`
	IsNative   bool        `json:"isNative"`
	Currency   string      `json:"currency"`
	LogoURL    string      `json:"logoUrl"`
	Blockchain *Blockchain `json:"blockchain,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

// WalletAsset is an asset enabled on a wallet. Its ID is the "assetId" that
// balance, withdraw and swap endpoints expect.
type WalletAsset struct {
	ID        string    `json:"id"`
	IsActive  bool      `json:"isActive"`
	Asset     Asset     `json:"asset"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Balance of one asset in a wallet or address.
type Balance struct {
	Asset            WalletAsset `json:"asset"`
	Balance          Amount      `json:"balance"`
	ConvertedBalance Amount      `json:"convertedBalance"`
}

// AMLResult is an anti-money-laundering screening result.
type AMLResult struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Provider string `json:"provider"`
}

type WalletAnalytics struct {
	CurrentBalance Amount `json:"currentBalance"`
	NumberOfAssets int    `json:"numberOfAssets"`
	UnsweptBalance Amount `json:"unsweptBalance"`
}

// Wallet is a master wallet.
type Wallet struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	Address        string           `json:"address"`
	DerivationPath string           `json:"derivationPath"`
	Network        string           `json:"network"`
	Status         string           `json:"status"`
	IsActive       bool             `json:"isActive"`
	Blockchain     *Blockchain      `json:"blockchain,omitempty"`
	Assets         []WalletAsset    `json:"assets,omitempty"`
	Analytics      *WalletAnalytics `json:"analytics,omitempty"`
	Configurations map[string]any   `json:"configurations,omitempty"`
	PrivateKey     string           `json:"privateKey,omitempty"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
}

type AddressConfigurations struct {
	AML                   *AMLResult `json:"aml,omitempty"`
	DisableAutoSweep      bool       `json:"disableAutoSweep"`
	EnableGaslessWithdraw bool       `json:"enableGaslessWithdraw"`
	ShowPrivateKey        bool       `json:"showPrivateKey"`
}

// Address is a child address generated under a master wallet.
type Address struct {
	ID             string                 `json:"id"`
	Address        string                 `json:"address"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"` // INTERNAL or EXTERNAL (whitelisted)
	Network        string                 `json:"network"`
	DerivationPath string                 `json:"derivationPath"`
	IsActive       bool                   `json:"isActive"`
	Metadata       map[string]any         `json:"metadata"`
	Configurations *AddressConfigurations `json:"configurations,omitempty"`
	Blockchain     *Blockchain            `json:"blockchain,omitempty"`
	PrivateKey     string                 `json:"privateKey,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

// Transaction is an on-chain deposit, withdrawal, sweep, swap, etc.
type Transaction struct {
	ID                         string         `json:"id"`
	Reference                  string         `json:"reference"`
	Hash                       string         `json:"hash"`
	Type                       string         `json:"type"`   // DEPOSIT, WITHDRAW, SALVAGE, ...
	Status                     string         `json:"status"` // PENDING, SUCCESS, FAILED, ...
	Network                    string         `json:"network"`
	Amount                     Amount         `json:"amount"`
	AmountPaid                 Amount         `json:"amountPaid"`
	Fee                        Amount         `json:"fee"`
	FeeUSD                     Amount         `json:"feeUSD"`
	FeeHash                    string         `json:"feeHash"`
	AmountUSD                  Amount         `json:"amountUSD"`
	Rate                       Amount         `json:"rate"`
	RateUSD                    Amount         `json:"rateUSD"`
	ToAmount                   Amount         `json:"toAmount"`
	ToCurrency                 string         `json:"toCurrency"`
	TokenAddress               string         `json:"tokenAddress"`
	ExplorerURL                string         `json:"explorerUrl"`
	CreatedChannel             string         `json:"createdChannel"`
	IsAutoSweep                bool           `json:"isAutoSweep"`
	Currency                   string         `json:"currency"`
	SenderAddress              string         `json:"senderAddress"`
	RecipientAddress           string         `json:"recipientAddress"`
	Confirmed                  bool           `json:"confirmed"`
	Confirmations              int            `json:"confirmations"`
	BlockHash                  string         `json:"blockHash"`
	BlockNumber                int64          `json:"blockNumber"`
	GasFee                     Amount         `json:"gasFee"`
	GasPrice                   Amount         `json:"gasPrice"`
	GasUsed                    Amount         `json:"gasUsed"`
	ChainID                    json.Number    `json:"chainId"`
	Note                       string         `json:"note"`
	Reason                     string         `json:"reason"`
	Metadata                   map[string]any `json:"metadata"`
	AMLScreening               *AMLResult     `json:"amlScreening,omitempty"`
	Asset                      *Asset         `json:"asset,omitempty"`
	Blockchain                 *Blockchain    `json:"blockchain,omitempty"`
	Wallet                     *Wallet        `json:"wallet,omitempty"`
	Address                    *Address       `json:"address,omitempty"`
	Beneficiary                map[string]any `json:"beneficiary,omitempty"`
	AssetSwept                 bool           `json:"assetSwept"`
	AssetSweptAmount           Amount         `json:"assetSweptAmount"`
	AssetSweptAt               *time.Time     `json:"assetSweptAt,omitempty"`
	AssetSweptGasFee           Amount         `json:"assetSweptGasFee"`
	AssetSweptHash             string         `json:"assetSweptHash"`
	AssetSweptSenderAddress    string         `json:"assetSweptSenderAddress"`
	AssetSweptRecipientAddress string         `json:"assetSweptRecipientAddress"`
	CreatedAt                  time.Time      `json:"createdAt"`
	UpdatedAt                  time.Time      `json:"updatedAt"`
}

func (p PageParams) values() url.Values {
	q := url.Values{}
	setInt(q, "page", p.Page)
	setInt(q, "limit", p.Limit)
	return q
}
