package blockradar

import (
	"context"
	"net/http"
)

// SwapService calls the v1 endpoints in this file.
type SwapService struct{ c *Client }

// SwapRequest quotes or executes a swap between two wallet assets (same or cross chain).
type SwapRequest struct {
	FromAssetID      string                    `json:"fromAssetId"`
	ToAssetID        string                    `json:"toAssetId"`
	Amount           string                    `json:"amount"`
	AmountSide       AmountSide                `json:"amountSide,omitempty"`       // AmountSideSource (default) or AmountSideTarget
	RecipientAddress string                    `json:"recipientAddress,omitempty"` // external destination
	RecipientRouting *WithdrawRecipientRouting `json:"recipientRouting,omitempty"` // Stellar G... recipients: "none" or "memo_id"
	Order            string                    `json:"order,omitempty"`            // FASTEST, CHEAPEST, RECOMMENDED (default), NO_SLIPPAGE
	Reference        string                    `json:"reference,omitempty"`        // Execute only
	Metadata         map[string]any            `json:"metadata,omitempty"`         // Execute only
}

// SwapQuote is the price and fee estimate of a swap.
type SwapQuote struct {
	Amount               Amount                    `json:"amount"`
	FromAmount           Amount                    `json:"fromAmount"`
	ToAmount             Amount                    `json:"toAmount"`
	RequestedAmount      Amount                    `json:"requestedAmount"`
	AmountSide           string                    `json:"amountSide"`
	RecipientRouting     *SwapRecipientRoutingInfo `json:"recipientRouting,omitempty"`
	EstimatedArrivalTime int                       `json:"estimatedArrivalTime"` // seconds
	Impact               Amount                    `json:"impact"`
	ImpactInUSD          Amount                    `json:"impactInUSD"`
	MinAmount            Amount                    `json:"minAmount"`
	NativeBalance        Amount                    `json:"nativeBalance"`
	NativeBalanceInUSD   Amount                    `json:"nativeBalanceInUSD"`
	NetworkFee           Amount                    `json:"networkFee"`
	NetworkFeeInUSD      Amount                    `json:"networkFeeInUSD"`
	Rate                 Amount                    `json:"rate"`
	Slippage             Amount                    `json:"slippage"`
	TransactionFee       Amount                    `json:"transactionFee"`
}

// SwapRecipientRoutingInfo says how a Stellar routing ID will be used ("none" or "embedded").
type SwapRecipientRoutingInfo struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// SwapTransaction is the SWAP transaction created by Execute.
type SwapTransaction struct {
	Transaction
	AddressID    string      `json:"addressId"`
	ToAsset      *Asset      `json:"toAsset,omitempty"`
	ToBlockchain *Blockchain `json:"toBlockchain,omitempty"`
	ToWallet     *Wallet     `json:"toWallet,omitempty"`
}

// Quote prices a swap from the master wallet.
func (s *SwapService) Quote(ctx context.Context, walletID string, r SwapRequest) (*SwapQuote, error) {
	return call[SwapQuote](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/swaps/quote", nil, r)
}

// Execute runs a swap from the master wallet.
func (s *SwapService) Execute(ctx context.Context, walletID string, r SwapRequest) (*SwapTransaction, error) {
	return call[SwapTransaction](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/swaps/execute", nil, r)
}

// AddressQuote prices a swap from a child address.
func (s *SwapService) AddressQuote(ctx context.Context, walletID, addressID string, r SwapRequest) (*SwapQuote, error) {
	return call[SwapQuote](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/swaps/quote", nil, r)
}

// AddressExecute runs a swap from a child address.
func (s *SwapService) AddressExecute(ctx context.Context, walletID, addressID string, r SwapRequest) (*SwapTransaction, error) {
	return call[SwapTransaction](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/swaps/execute", nil, r)
}
