package blockradar

import (
	"context"
	"net/http"
)

// AssetsService calls the v1 endpoints in this file.
type AssetsService struct{ c *Client }

// AssetUpdateRequest enables or disables an asset on a wallet.
type AssetUpdateRequest struct {
	IsActive bool `json:"isActive"`
}

func (s *AssetsService) path(walletID string) string {
	return "/v1/wallets/" + seg(walletID) + "/assets"
}

// List returns the assets enabled on a wallet.
func (s *AssetsService) List(ctx context.Context, walletID string) ([]WalletAsset, error) {
	return getSlice[WalletAsset](ctx, s.c, s.path(walletID), nil)
}

// Add enables an asset on a wallet. assetID is the Asset ID (see Misc.Assets).
func (s *AssetsService) Add(ctx context.Context, walletID, assetID string) (*WalletAsset, error) {
	body := struct {
		AssetID string `json:"assetId"`
	}{assetID}
	return call[WalletAsset](ctx, s.c, http.MethodPost, s.path(walletID), nil, body)
}

// Update changes a wallet asset's status. walletAssetID is WalletAsset.ID.
func (s *AssetsService) Update(ctx context.Context, walletID, walletAssetID string, r AssetUpdateRequest) (*WalletAsset, error) {
	return call[WalletAsset](ctx, s.c, http.MethodPatch, s.path(walletID)+"/"+seg(walletAssetID), nil, r)
}

// Remove removes an asset from a wallet. walletAssetID is WalletAsset.ID.
func (s *AssetsService) Remove(ctx context.Context, walletID, walletAssetID string) (*WalletAsset, error) {
	return call[WalletAsset](ctx, s.c, http.MethodDelete, s.path(walletID)+"/"+seg(walletAssetID), nil, nil)
}
