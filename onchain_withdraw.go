package blockradar

import (
	"context"
	"net/http"
)

// WithdrawService calls the v1 endpoints in this file.
type WithdrawService struct{ c *Client }

// ---- requests ----

// WithdrawRecipientRouting sets the Stellar memo for a G... recipient.
// Type is "memo_id", "memo_text" or "none"; Value is omitted for "none".
type WithdrawRecipientRouting struct {
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
}

// WithdrawAsset is one item of a batch withdrawal (max 20).
type WithdrawAsset struct {
	ID               string                    `json:"id"` // wallet asset ID
	Amount           string                    `json:"amount"`
	Address          string                    `json:"address"`
	Reference        string                    `json:"reference,omitempty"`
	Note             string                    `json:"note,omitempty"`
	RecipientRouting *WithdrawRecipientRouting `json:"recipientRouting,omitempty"`
	Metadata         map[string]any            `json:"metadata,omitempty"`
}

// WithdrawRequest is a single withdrawal (AssetID, Address, Amount) or a batch (Assets).
type WithdrawRequest struct {
	AssetID          string                    `json:"assetId,omitempty"`
	Address          string                    `json:"address,omitempty"`
	Amount           string                    `json:"amount,omitempty"`
	Reference        string                    `json:"reference,omitempty"`
	Note             string                    `json:"note,omitempty"`
	RecipientRouting *WithdrawRecipientRouting `json:"recipientRouting,omitempty"` // Stellar, single withdrawals only
	Metadata         map[string]any            `json:"metadata,omitempty"`
	Assets           []WithdrawAsset           `json:"assets,omitempty"`
}

// ---- responses ----

// WithdrawResult is the withdrawal transaction (single) or Success/Errors (batch).
type WithdrawResult struct {
	Transaction
	SignedTransaction string              `json:"signedTransaction"` // sign-only
	Success           []WithdrawBatchItem `json:"success"`
	Errors            []any               `json:"errors"`
}

// WithdrawBatchItem is one queued (or signed) withdrawal of a batch.
type WithdrawBatchItem struct {
	ID                string `json:"id"`
	Reference         string `json:"reference"`
	Amount            Amount `json:"amount"`
	SenderAddress     string `json:"senderAddress"`
	RecipientAddress  string `json:"recipientAddress"`
	Status            string `json:"status"`
	Hash              string `json:"hash"`              // sign-only
	SignedTransaction string `json:"signedTransaction"` // sign-only
}

// WithdrawNetworkFee is a withdrawal fee estimate; Fees and the Total* fields are set for batches.
type WithdrawNetworkFee struct {
	NetworkFee           Amount        `json:"networkFee"`
	NetworkFeeInUSD      Amount        `json:"networkFeeInUSD"` // returned by the live API, not in the docs
	TransactionFee       Amount        `json:"transactionFee"`  // returned by the live API, not in the docs
	NativeBalance        Amount        `json:"nativeBalance"`
	NativeBalanceInUSD   Amount        `json:"nativeBalanceInUSD"`
	EstimatedArrivalTime int           `json:"estimatedArrivalTime"` // seconds
	Fees                 []WithdrawFee `json:"fees"`
	TotalNetworkFee      Amount        `json:"totalNetworkFee"`
	TotalNetworkFeeInUSD Amount        `json:"totalNetworkFeeInUSD"`
	TotalTransactionFee  Amount        `json:"totalTransactionFee"`
	Errors               []any         `json:"errors"`
}

// WithdrawFee is the fee estimate for one item of a batch.
type WithdrawFee struct {
	Index                int    `json:"index"`
	AssetID              string `json:"assetId"`
	Address              string `json:"address"`
	Amount               Amount `json:"amount"`
	NetworkFee           Amount `json:"networkFee"`
	TransactionFee       Amount `json:"transactionFee"`
	EstimatedArrivalTime int    `json:"estimatedArrivalTime"`
}

// ---- master wallet ----

// Execute withdraws crypto from the master wallet.
func (s *WithdrawService) Execute(ctx context.Context, walletID string, r WithdrawRequest) (*WithdrawResult, error) {
	return call[WithdrawResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/withdraw", nil, r)
}

// Sign signs a master wallet withdrawal without broadcasting it (not supported on Stellar).
func (s *WithdrawService) Sign(ctx context.Context, walletID string, r WithdrawRequest) (*WithdrawResult, error) {
	return call[WithdrawResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/withdraw/sign", nil, r)
}

// NetworkFee estimates the fee of a master wallet withdrawal.
func (s *WithdrawService) NetworkFee(ctx context.Context, walletID string, r WithdrawRequest) (*WithdrawNetworkFee, error) {
	return call[WithdrawNetworkFee](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/withdraw/network-fee", nil, r)
}

// ---- child address ----

// AddressExecute withdraws crypto from a child address.
func (s *WithdrawService) AddressExecute(ctx context.Context, walletID, addressID string, r WithdrawRequest) (*WithdrawResult, error) {
	return call[WithdrawResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/withdraw", nil, r)
}

// AddressSign signs a child address withdrawal without broadcasting it (not supported on Stellar).
func (s *WithdrawService) AddressSign(ctx context.Context, walletID, addressID string, r WithdrawRequest) (*WithdrawResult, error) {
	return call[WithdrawResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/withdraw/sign", nil, r)
}

// AddressNetworkFee estimates the fee of a child address withdrawal.
func (s *WithdrawService) AddressNetworkFee(ctx context.Context, walletID, addressID string, r WithdrawRequest) (*WithdrawNetworkFee, error) {
	return call[WithdrawNetworkFee](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/withdraw/network-fee", nil, r)
}
