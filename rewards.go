package blockradar

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// RewardsService covers /v1/rewards and /v1/wallets/{id}/rewards: yield (Earn)
// on stablecoins through DeFi products such as Aave and Compound.
//
// Child addresses use the same endpoints with AddressID set.
type RewardsService struct{ c *Client }

// ---- types ----

// RewardPosition is an earn position aggregated per product. When a single-asset
// query spans several products, ProductID/ProductKey/Provider are "" and
// Products holds the breakdown.
type RewardPosition struct {
	Type                  string           `json:"type"` // "defi"
	ProductID             string           `json:"productId"`
	ProductKey            string           `json:"productKey"` // e.g. "aave-base-usdc"
	Provider              string           `json:"provider"`   // e.g. "aave"
	APY                   Amount           `json:"apy"`
	Balance               Amount           `json:"balance"`
	BalanceInUSD          Amount           `json:"balanceInUSD"`
	TotalDeposited        Amount           `json:"totalDeposited"`
	TotalDepositedInUSD   Amount           `json:"totalDepositedInUSD"`
	Earnings              Amount           `json:"earnings"`
	EarningsInUSD         Amount           `json:"earningsInUSD"`
	TodayEarnings         Amount           `json:"todayEarnings"`
	WeeklyEarnings        Amount           `json:"weeklyEarnings"`
	LifetimeEarnings      Amount           `json:"lifetimeEarnings"`
	LifetimeEarningsInUSD Amount           `json:"lifetimeEarningsInUSD"`
	Asset                 WalletAsset      `json:"asset"`
	Products              []RewardPosition `json:"products,omitempty"`
}

// RewardProviderPaused reports whether the provider has paused deposits or withdrawals.
type RewardProviderPaused struct {
	Deposits    bool `json:"deposits"`
	Withdrawals bool `json:"withdrawals"`
}

// RewardProduct is an earn product (provider + blockchain + asset) from the catalogue.
type RewardProduct struct {
	ProductID       string               `json:"productId"`
	ProductKey      string               `json:"productKey"`
	StrategyID      string               `json:"strategyId"`
	Provider        string               `json:"provider"`
	Type            string               `json:"type"`
	Name            string               `json:"name"`
	Blockchain      string               `json:"blockchain"` // slug
	Network         string               `json:"network"`
	APY             Amount               `json:"apy"`
	APYUpdatedAt    *time.Time           `json:"apyUpdatedAt"`
	MinDeposit      Amount               `json:"minDeposit"`
	DepositsEnabled bool                 `json:"depositsEnabled"`
	ProviderPaused  RewardProviderPaused `json:"providerPaused"`
	IsActive        bool                 `json:"isActive"`
	CreatedAt       time.Time            `json:"createdAt"`
	Asset           Asset                `json:"asset"`
}

// RewardAsset is a wallet asset eligible for one earn product. ID is the wallet asset ID.
type RewardAsset struct {
	ID              string               `json:"id"`
	ProductID       string               `json:"productId"`
	ProductKey      string               `json:"productKey"`
	StrategyID      string               `json:"strategyId"`
	Provider        string               `json:"provider"`
	Type            string               `json:"type"`
	APY             Amount               `json:"apy"`
	DepositsEnabled bool                 `json:"depositsEnabled"`
	ProviderPaused  RewardProviderPaused `json:"providerPaused"`
	IsActive        bool                 `json:"isActive"`
	Asset           Asset                `json:"asset"`
}

// RewardSupportedChains lists blockchain slugs that support earning.
type RewardSupportedChains struct {
	All        []string            `json:"all"`
	ByType     map[string][]string `json:"byType"`     // e.g. "defi"
	ByProvider map[string][]string `json:"byProvider"` // e.g. "aave", "compound"
}

// RewardNetworkFee is a fee preview for a rewards deposit or withdrawal.
type RewardNetworkFee = GatewayNetworkFee

// ---- requests / params ----

// RewardNetworkFeeRequest previews a deposit or withdrawal. AssetID and Amount are required.
type RewardNetworkFeeRequest struct {
	AssetID   string `json:"assetId"`
	Amount    string `json:"amount"` // "max" for a full withdrawal
	ProductID string `json:"productId,omitempty"`
	Type      string `json:"type,omitempty"` // legacy; prefer ProductID
	AddressID string `json:"addressId,omitempty"`
}

// RewardDepositRequest deposits into an earn product. AssetID and Amount are required.
type RewardDepositRequest struct {
	AssetID   string         `json:"assetId"`
	Amount    string         `json:"amount"`
	ProductID string         `json:"productId,omitempty"` // required if the asset has several products
	Type      string         `json:"type,omitempty"`      // legacy; prefer ProductID
	Reference string         `json:"reference,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	AddressID string         `json:"addressId,omitempty"` // child address; omit for master wallet
}

// RewardWithdrawRequest withdraws from a position. AssetID and Amount (net, or "max") are required.
type RewardWithdrawRequest struct {
	AssetID   string         `json:"assetId"`
	Amount    string         `json:"amount"`
	ProductID string         `json:"productId,omitempty"` // required if several positions exist for the asset
	Type      string         `json:"type,omitempty"`      // legacy; ignored when ProductID is set
	Address   string         `json:"address,omitempty"`   // external destination; omit to credit the signer
	Reference string         `json:"reference,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	AddressID string         `json:"addressId,omitempty"` // child address; omit for master wallet
}

// RewardProductsParams filters the product catalogue.
type RewardProductsParams struct {
	Blockchain string // slug, e.g. "base"
	Asset      string // symbol, e.g. "USDC"
}

func (p RewardProductsParams) values() url.Values {
	q := url.Values{}
	setStr(q, "blockchain", p.Blockchain)
	setStr(q, "asset", p.Asset)
	return q
}

// RewardWalletParams scopes Wallet to a child address and/or one asset.
type RewardWalletParams struct {
	AddressID string
	AssetID   string
}

func (p RewardWalletParams) values() url.Values {
	q := url.Values{}
	setStr(q, "addressId", p.AddressID)
	setStr(q, "assetId", p.AssetID)
	return q
}

// RewardAssetsParams filters WalletAssets.
type RewardAssetsParams struct {
	Type      string // e.g. "defi"
	AddressID string
}

func (p RewardAssetsParams) values() url.Values {
	q := url.Values{}
	setStr(q, "type", p.Type)
	setStr(q, "addressId", p.AddressID)
	return q
}

// RewardTransactionsParams: Limit defaults to 10, max 50.
type RewardTransactionsParams struct {
	Limit int
}

func walletRewards(walletID string) string {
	return "/v1/wallets/" + seg(walletID) + "/rewards"
}

// oneOrMany decodes either a single JSON object or an array into a slice.
type oneOrMany[T any] []T

func (o *oneOrMany[T]) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*o = nil
		return nil
	}
	if b[0] == '[' {
		return json.Unmarshal(b, (*[]T)(o))
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*o = []T{v}
	return nil
}

// ---- business endpoints ----

// Business returns earn positions across all wallets, per product.
func (s *RewardsService) Business(ctx context.Context) ([]RewardPosition, error) {
	return getSlice[RewardPosition](ctx, s.c, "/v1/rewards", nil)
}

// Products lists the earn product catalogue.
func (s *RewardsService) Products(ctx context.Context, p RewardProductsParams) ([]RewardProduct, error) {
	return getSlice[RewardProduct](ctx, s.c, "/v1/rewards/products", p.values())
}

// SupportedChains returns the blockchains that support earning.
func (s *RewardsService) SupportedChains(ctx context.Context) (*RewardSupportedChains, error) {
	return call[RewardSupportedChains](ctx, s.c, http.MethodGet, "/v1/rewards/supported-chains", nil, nil)
}

// Transactions returns the most recent reward deposits and withdrawals, newest first.
func (s *RewardsService) Transactions(ctx context.Context, p RewardTransactionsParams) ([]Transaction, error) {
	q := url.Values{}
	setInt(q, "limit", p.Limit)
	return getSlice[Transaction](ctx, s.c, "/v1/rewards/transactions", q)
}

// ---- wallet endpoints ----

// Wallet returns a wallet's earn positions. With AssetID set the API returns a
// single aggregated position, which comes back as a one-element slice.
func (s *RewardsService) Wallet(ctx context.Context, walletID string, p RewardWalletParams) ([]RewardPosition, error) {
	out, err := call[oneOrMany[RewardPosition]](ctx, s.c, http.MethodGet, walletRewards(walletID), p.values(), nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// WalletAssets lists a wallet's reward-eligible assets, one entry per product.
func (s *RewardsService) WalletAssets(ctx context.Context, walletID string, p RewardAssetsParams) ([]RewardAsset, error) {
	return getSlice[RewardAsset](ctx, s.c, walletRewards(walletID)+"/assets", p.values())
}

// Deposit deposits into an earn product. Returns a PENDING transaction; completion arrives by webhook.
func (s *RewardsService) Deposit(ctx context.Context, walletID string, r RewardDepositRequest) (*Transaction, error) {
	return call[Transaction](ctx, s.c, http.MethodPost, walletRewards(walletID)+"/deposit", nil, r)
}

// DepositNetworkFee previews the fee for Deposit.
func (s *RewardsService) DepositNetworkFee(ctx context.Context, walletID string, r RewardNetworkFeeRequest) (*RewardNetworkFee, error) {
	return call[RewardNetworkFee](ctx, s.c, http.MethodPost, walletRewards(walletID)+"/deposit/network-fee", nil, r)
}

// Withdraw withdraws from an earn position. Returns a PENDING transaction; completion arrives by webhook.
func (s *RewardsService) Withdraw(ctx context.Context, walletID string, r RewardWithdrawRequest) (*Transaction, error) {
	return call[Transaction](ctx, s.c, http.MethodPost, walletRewards(walletID)+"/withdraw", nil, r)
}

// WithdrawNetworkFee previews the fee for Withdraw.
func (s *RewardsService) WithdrawNetworkFee(ctx context.Context, walletID string, r RewardNetworkFeeRequest) (*RewardNetworkFee, error) {
	return call[RewardNetworkFee](ctx, s.c, http.MethodPost, walletRewards(walletID)+"/withdraw/network-fee", nil, r)
}
