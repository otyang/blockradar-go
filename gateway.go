package blockradar

import (
	"context"
	"net/http"
)

// GatewayService covers /v1/gateway and /v1/addresses/{id}/gateway: a unified
// USDC balance you deposit into from one chain and withdraw (mint) on another.
type GatewayService struct{ c *Client }

// ---- types ----

// GatewayBalance is the unified Gateway USDC balance.
type GatewayBalance struct {
	Asset            Asset  `json:"asset"`
	Balance          Amount `json:"balance"`
	ConvertedBalance Amount `json:"convertedBalance"`
}

// GatewayWallet is a master wallet that supports Gateway, with its balance.
type GatewayWallet struct {
	Wallet
	Balance          Amount `json:"balance"`
	ConvertedBalance Amount `json:"convertedBalance"`
}

// GatewayNetworkFee is a fee estimate for a Gateway deposit or withdrawal.
type GatewayNetworkFee struct {
	NetworkFee           Amount `json:"networkFee"`
	NetworkFeeInUSD      Amount `json:"networkFeeInUSD"`
	NativeBalance        Amount `json:"nativeBalance"`
	NativeBalanceInUSD   Amount `json:"nativeBalanceInUSD"`
	TransactionFee       Amount `json:"transactionFee"`       // string or number on the wire
	EstimatedArrivalTime int    `json:"estimatedArrivalTime"` // seconds
}

// ---- requests ----

// GatewayDepositRequest moves USDC into the Gateway balance. Amount is required.
type GatewayDepositRequest struct {
	Amount    string         `json:"amount"`
	Reference string         `json:"reference,omitempty"`
	Note      string         `json:"note,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Code      string         `json:"code,omitempty"` // 2FA code, when required
}

// GatewayWithdrawRequest mints USDC from the Gateway balance to Address on Blockchain.
// Amount, Blockchain (slug, e.g. "base") and Address are required.
type GatewayWithdrawRequest struct {
	Amount     string         `json:"amount"`
	Blockchain string         `json:"blockchain"`
	Address    string         `json:"address"`
	Reference  string         `json:"reference,omitempty"`
	Note       string         `json:"note,omitempty"` // ignored: Blockradar sets the note
	Metadata   map[string]any `json:"metadata,omitempty"`
}

func addressGateway(addressID string) string {
	return "/v1/addresses/" + seg(addressID) + "/gateway"
}

// ---- master wallet endpoints ----

// Wallets lists the master wallets that support Gateway.
func (s *GatewayService) Wallets(ctx context.Context, p PageParams) ([]GatewayWallet, error) {
	return getSlice[GatewayWallet](ctx, s.c, "/v1/gateway/wallets", p.values())
}

// Balance returns the business's unified Gateway balance.
func (s *GatewayService) Balance(ctx context.Context) (*GatewayBalance, error) {
	return call[GatewayBalance](ctx, s.c, http.MethodGet, "/v1/gateway/balance", nil, nil)
}

// Transactions lists Gateway deposits and withdrawals across master wallets.
func (s *GatewayService) Transactions(ctx context.Context, p PageParams) (*List[Transaction], error) {
	var out List[Transaction]
	// Capital T is how the path is documented.
	if err := s.c.do(ctx, http.MethodGet, "/v1/gateway/Transactions", p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Deposit moves USDC from a master wallet into the Gateway balance.
func (s *GatewayService) Deposit(ctx context.Context, walletID string, r GatewayDepositRequest) (*Transaction, error) {
	return call[Transaction](ctx, s.c, http.MethodPost, "/v1/gateway/deposit/"+seg(walletID), nil, r)
}

// DepositNetworkFee estimates the fee for Deposit.
func (s *GatewayService) DepositNetworkFee(ctx context.Context, walletID string, r GatewayDepositRequest) (*GatewayNetworkFee, error) {
	return call[GatewayNetworkFee](ctx, s.c, http.MethodPost, "/v1/gateway/deposit/"+seg(walletID)+"/network-fee", nil, r)
}

// Withdraw mints USDC from the master wallets' Gateway balance on the destination chain.
func (s *GatewayService) Withdraw(ctx context.Context, r GatewayWithdrawRequest) (*Transaction, error) {
	return call[Transaction](ctx, s.c, http.MethodPost, "/v1/gateway/withdraw", nil, r)
}

// WithdrawNetworkFee estimates the fee for Withdraw.
func (s *GatewayService) WithdrawNetworkFee(ctx context.Context, r GatewayWithdrawRequest) (*GatewayNetworkFee, error) {
	return call[GatewayNetworkFee](ctx, s.c, http.MethodPost, "/v1/gateway/withdraw/network-fee", nil, r)
}

// ---- child address endpoints ----

// AddressBalance returns a child address's Gateway balance.
func (s *GatewayService) AddressBalance(ctx context.Context, addressID string) (*GatewayBalance, error) {
	return call[GatewayBalance](ctx, s.c, http.MethodGet, addressGateway(addressID)+"/balance", nil, nil)
}

// AddressTransactions lists a child address's Gateway transactions.
func (s *GatewayService) AddressTransactions(ctx context.Context, addressID string, p PageParams) (*List[Transaction], error) {
	var out List[Transaction]
	if err := s.c.do(ctx, http.MethodGet, addressGateway(addressID)+"/transactions", p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddressDeposit moves USDC from a child address of walletID into its Gateway balance.
func (s *GatewayService) AddressDeposit(ctx context.Context, addressID, walletID string, r GatewayDepositRequest) (*Transaction, error) {
	return call[Transaction](ctx, s.c, http.MethodPost, addressGateway(addressID)+"/deposit/"+seg(walletID), nil, r)
}

// AddressDepositNetworkFee estimates the fee for AddressDeposit.
func (s *GatewayService) AddressDepositNetworkFee(ctx context.Context, addressID, walletID string, r GatewayDepositRequest) (*GatewayNetworkFee, error) {
	return call[GatewayNetworkFee](ctx, s.c, http.MethodPost, addressGateway(addressID)+"/deposit/"+seg(walletID)+"/network-fee", nil, r)
}

// AddressWithdraw mints USDC from a child address's Gateway balance on the destination chain.
func (s *GatewayService) AddressWithdraw(ctx context.Context, addressID string, r GatewayWithdrawRequest) (*Transaction, error) {
	return call[Transaction](ctx, s.c, http.MethodPost, addressGateway(addressID)+"/withdraw", nil, r)
}

// AddressWithdrawNetworkFee estimates the fee for AddressWithdraw.
func (s *GatewayService) AddressWithdrawNetworkFee(ctx context.Context, addressID string, r GatewayWithdrawRequest) (*GatewayNetworkFee, error) {
	return call[GatewayNetworkFee](ctx, s.c, http.MethodPost, addressGateway(addressID)+"/withdraw/network-fee", nil, r)
}
