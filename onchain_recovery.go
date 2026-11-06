package blockradar

import (
	"context"
	"net/http"
)

// AssetRecoveryService calls the v1 endpoints in this file.
type AssetRecoveryService struct{ c *Client }

// RecoveryRequest moves stuck assets (native or token) from SenderAddress to RecipientAddress.
type RecoveryRequest struct {
	SenderAddress    string `json:"senderAddress"`
	RecipientAddress string `json:"recipientAddress"`
	Amount           string `json:"amount"`
	BlockchainID     string `json:"blockchainId"`
	TokenAddress     string `json:"tokenAddress,omitempty"` // set for token recovery
	RPCURL           string `json:"rpcUrl,omitempty"`       // for unsupported blockchains
	IsL2             bool   `json:"isL2,omitempty"`         // RPCURL points to a layer-2 chain
	CrossNetwork     bool   `json:"crossNetwork,omitempty"` // assets sent to the wrong network environment; not with RPCURL
}

// RecoveryNetworkFee is the estimated fee of a recovery. NativeBalance and Symbol are set for tokens.
type RecoveryNetworkFee struct {
	Balance       Amount `json:"balance"`
	Fee           Amount `json:"fee"`
	NativeBalance Amount `json:"nativeBalance"`
	Symbol        string `json:"symbol"`
}

// Execute starts an asset recovery. The API returns no data, only a message.
func (s *AssetRecoveryService) Execute(ctx context.Context, walletID string, r RecoveryRequest) error {
	return s.c.do(ctx, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/salvage", nil, r, nil)
}

// NetworkFee estimates the fee of an asset recovery.
func (s *AssetRecoveryService) NetworkFee(ctx context.Context, walletID string, r RecoveryRequest) (*RecoveryNetworkFee, error) {
	return call[RecoveryNetworkFee](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/salvage/network-fee", nil, r)
}
