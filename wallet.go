package blockradar

import (
	"context"
	"net/http"
	"net/url"
)

// WalletsService calls the v1 endpoints in this file.
type WalletsService struct{ c *Client }

// WalletUpdateRequest changes a wallet's name or description.
type WalletUpdateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// WalletRescanRequest asks the deposit finder to rescan blocks.
type WalletRescanRequest struct {
	TransactionHash string `json:"transactionHash,omitempty"`
}

// WalletSweepRequest triggers a sweep of unswept assets into the master wallet.
type WalletSweepRequest struct {
	TransactionID string `json:"transactionId,omitempty"`
	ForceSweep    *bool  `json:"forceSweep,omitempty"`
}

func (s *WalletsService) path(walletID string) string { return "/v1/wallets/" + seg(walletID) }

// List returns every master wallet the API key can access.
// GET /v1/wallets is not in the API reference but is served by the API.
func (s *WalletsService) List(ctx context.Context) ([]Wallet, error) {
	return getSlice[Wallet](ctx, s.c, "/v1/wallets", nil)
}

// Get returns a master wallet. showPrivateKey adds the private key to the response.
func (s *WalletsService) Get(ctx context.Context, walletID string, showPrivateKey bool) (*Wallet, error) {
	q := url.Values{}
	if showPrivateKey {
		q.Set("showPrivateKey", "true")
	}
	return call[Wallet](ctx, s.c, http.MethodGet, s.path(walletID), q, nil)
}

// Update changes a wallet's name or description.
func (s *WalletsService) Update(ctx context.Context, walletID string, r WalletUpdateRequest) (*Wallet, error) {
	return call[Wallet](ctx, s.c, http.MethodPatch, s.path(walletID), nil, r)
}

// Balance returns the wallet balance of one asset (assetId is the wallet asset ID).
func (s *WalletsService) Balance(ctx context.Context, walletID, assetID string) (*Balance, error) {
	q := url.Values{}
	setStr(q, "assetId", assetID)
	return call[Balance](ctx, s.c, http.MethodGet, s.path(walletID)+"/balance", q, nil)
}

// Balances returns the wallet balance of every enabled asset.
// The docs also show a JSON body on this GET; it is not sent.
func (s *WalletsService) Balances(ctx context.Context, walletID string) ([]Balance, error) {
	return getSlice[Balance](ctx, s.c, s.path(walletID)+"/balances", nil)
}

// WebhookLogs lists webhook deliveries for the wallet.
func (s *WalletsService) WebhookLogs(ctx context.Context, walletID string, p PageParams) (*List[TransactionWebhookLog], error) {
	var out List[TransactionWebhookLog]
	if err := s.c.do(ctx, http.MethodGet, s.path(walletID)+"/webhooks", p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RescanBlocks runs the deposit finder to pick up missed deposits.
func (s *WalletsService) RescanBlocks(ctx context.Context, walletID string, r WalletRescanRequest) error {
	return s.c.do(ctx, http.MethodPost, s.path(walletID)+"/rescan/blocks", nil, r, nil)
}

// SweepAssets triggers a sweep of all unswept assets into the master wallet.
func (s *WalletsService) SweepAssets(ctx context.Context, walletID string, r WalletSweepRequest) error {
	return s.c.do(ctx, http.MethodPost, s.path(walletID)+"/sweep/assets", nil, r, nil)
}
