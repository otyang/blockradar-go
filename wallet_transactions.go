package blockradar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TransactionsService calls the v1 endpoints in this file.
type TransactionsService struct{ c *Client }

// TransactionDetail is a Transaction plus the extra fields recent API
// responses carry (converted values, swap targets, ...).
// Fields already on Transaction are not redeclared, so they stay set on the embedded value.
type TransactionDetail struct {
	Transaction
	ConvertedAmount           Amount          `json:"convertedAmount"`
	ConvertedGasFee           Amount          `json:"convertedGasFee"`
	ConvertedAssetSweptGasFee Amount          `json:"convertedAssetSweptGasFee"`
	SignedTransaction         json.RawMessage `json:"signedTransaction,omitempty"`
	Customer                  map[string]any  `json:"customer,omitempty"`
	PaymentLink               map[string]any  `json:"paymentLink,omitempty"`
	ToExplorerURL             string          `json:"toExplorerUrl"`
	ToAsset                   *Asset          `json:"toAsset,omitempty"`
	ToBlockchain              *Blockchain     `json:"toBlockchain,omitempty"`
	ToWallet                  *Wallet         `json:"toWallet,omitempty"`
}

// TransactionAnalytics summarises a wallet's transactions.
//
// The live API returns the Total* fields; the docs show the shorter names
// (depositsCount, ...). Both are kept so either shape decodes.
type TransactionAnalytics struct {
	TotalTransactionsCount         int    `json:"totalTransactionsCount"`
	TotalTransactionsSum           Amount `json:"totalTransactionsSum"`
	TotalDepositsCount             int    `json:"totalDepositsCount"`
	TotalDepositsSum               Amount `json:"totalDepositsSum"`
	TotalWithdrawsCount            int    `json:"totalWithdrawsCount"`
	TotalWithdrawsSum              Amount `json:"totalWithdrawsSum"`
	TotalNetworkFee                Amount `json:"totalNetworkFee"`
	TotalSwapsCount                int    `json:"totalSwapsCount"`
	TotalSalvagesCount             int    `json:"totalSalvagesCount"`
	TotalSignedCount               int    `json:"totalSignedCount"`
	TotalCustomSmartContractsCount int    `json:"totalCustomSmartContractsCount"`
	TotalGatewayDepositsCount      int    `json:"totalGatewayDepositsCount"`
	TotalGatewayWithdrawsCount     int    `json:"totalGatewayWithdrawsCount"`
	TotalAutoFundingsCount         int    `json:"totalAutoFundingsCount"`
	TotalRewardDepositsCount       int    `json:"totalRewardDepositsCount"`
	TotalRewardWithdrawsCount      int    `json:"totalRewardWithdrawsCount"`
	TotalStakingsCount             int    `json:"totalStakingsCount"`
	TotalUnstakingsCount           int    `json:"totalUnstakingsCount"`
	TotalRestakingsCount           int    `json:"totalRestakingsCount"`
	TotalUnstakingWithdrawsCount   int    `json:"totalUnstakingWithdrawsCount"`

	// Documented names (not seen on the live API).
	DepositsCount            int    `json:"depositsCount"`
	DepositsSumIn24Hours     Amount `json:"depositsSumIn24Hours"`
	TransactionsCount        int    `json:"transactionsCount"`
	TransactionsSumIn24Hours Amount `json:"transactionsSumIn24Hours"`
	WithdrawsCount           int    `json:"withdrawsCount"`
	WithdrawsSumIn24Hours    Amount `json:"withdrawsSumIn24Hours"`
}

// TransactionList is a page of wallet transactions with analytics.
type TransactionList struct {
	List[TransactionDetail]
	Analytics TransactionAnalytics `json:"analytics"`
}

// TransactionWebhookLog is one webhook delivery attempt record.
type TransactionWebhookLog struct {
	ID        string          `json:"id"`
	Event     string          `json:"event"` // e.g. deposit.success
	Attempts  int             `json:"attempts"`
	Status    string          `json:"status"` // success, failed
	URL       string          `json:"url"`
	Reference string          `json:"reference"`
	Payload   json.RawMessage `json:"payload"`  // event data; usually a Transaction
	Response  json.RawMessage `json:"response"` // string ("Ok") or object ({"status":404,"text":"Not Found"})
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// TransactionListParams filters transaction lists.
type TransactionListParams struct {
	ID     string
	Hash   string
	Status string   // PENDING, SUCCESS, FAILED
	Type   string   // DEPOSIT, WITHDRAW, SALVAGE, ...
	Order  string   // DESC or ASC
	Assets []string // asset symbols, e.g. USDT, USDC
	Page   int      // not in the docs, but responses are paginated
	Limit  int
}

func (p TransactionListParams) values() url.Values {
	q := url.Values{}
	setStr(q, "id", p.ID)
	setStr(q, "hash", p.Hash)
	setStr(q, "status", p.Status)
	setStr(q, "type", p.Type)
	setStr(q, "order", p.Order)
	setStr(q, "assets", strings.Join(p.Assets, ","))
	setInt(q, "page", p.Page)
	setInt(q, "limit", p.Limit)
	return q
}

// TransactionSweepRequest sweeps the asset of one transaction to the master wallet.
type TransactionSweepRequest struct {
	TransactionID string `json:"transactionId"`
	ForceSweep    *bool  `json:"forceSweep,omitempty"`
}

func (s *TransactionsService) path(walletID string) string {
	return "/v1/wallets/" + seg(walletID) + "/transactions"
}

// List returns a wallet's transactions.
func (s *TransactionsService) List(ctx context.Context, walletID string, p TransactionListParams) (*TransactionList, error) {
	var out TransactionList
	if err := s.c.do(ctx, http.MethodGet, s.path(walletID), p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns one transaction of a wallet.
func (s *TransactionsService) Get(ctx context.Context, walletID, transactionID string) (*TransactionDetail, error) {
	return call[TransactionDetail](ctx, s.c, http.MethodGet, s.path(walletID)+"/"+seg(transactionID), nil, nil)
}

// WebhookLogs lists the webhook deliveries for a transaction.
func (s *TransactionsService) WebhookLogs(ctx context.Context, walletID, transactionID string) ([]TransactionWebhookLog, error) {
	return getSlice[TransactionWebhookLog](ctx, s.c, s.path(walletID)+"/"+seg(transactionID)+"/webhooks", nil)
}

// Cancel cancels a pending transaction.
func (s *TransactionsService) Cancel(ctx context.Context, walletID, transactionID string) (*TransactionDetail, error) {
	return call[TransactionDetail](ctx, s.c, http.MethodPost, s.path(walletID)+"/cancel", nil, map[string]string{"id": transactionID})
}

// Retry retries a failed transaction.
func (s *TransactionsService) Retry(ctx context.Context, walletID, transactionID string) (*TransactionDetail, error) {
	return call[TransactionDetail](ctx, s.c, http.MethodPost, s.path(walletID)+"/retry", nil, map[string]string{"id": transactionID})
}

// SweepAssets triggers the sweep of a transaction's asset into the master wallet.
func (s *TransactionsService) SweepAssets(ctx context.Context, walletID string, r TransactionSweepRequest) error {
	return s.c.do(ctx, http.MethodPost, s.path(walletID)+"/sweep/assets", nil, r, nil)
}

// ResendWebhook re-sends the webhook for a transaction.
func (s *TransactionsService) ResendWebhook(ctx context.Context, walletID, transactionID string) (*TransactionDetail, error) {
	return call[TransactionDetail](ctx, s.c, http.MethodPost, s.path(walletID)+"/webhooks/resend", nil, map[string]string{"id": transactionID})
}
