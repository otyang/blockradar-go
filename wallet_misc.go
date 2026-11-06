package blockradar

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// MiscService calls the v1 endpoints in this file.
type MiscService struct{ c *Client }

// MiscAssetsParams filters the supported-assets list. All fields are optional.
type MiscAssetsParams struct {
	Network      string // mainnet or testnet
	Symbol       string // e.g. USDC
	BlockchainID string
}

func (p MiscAssetsParams) values() url.Values {
	q := url.Values{}
	setStr(q, "network", p.Network)
	setStr(q, "symbol", p.Symbol)
	setStr(q, "blockchainId", p.BlockchainID)
	return q
}

// MiscRates maps asset symbol -> currency -> rate, e.g. rates["USDC"]["USD"].
type MiscRates map[string]map[string]Amount

// Assets lists the assets Blockradar supports.
func (s *MiscService) Assets(ctx context.Context, p MiscAssetsParams) ([]Asset, error) {
	return getSlice[Asset](ctx, s.c, "/v1/assets", p.values())
}

// Rates returns current rates of assets (symbols, e.g. "USDT") in currency (e.g. "USD").
func (s *MiscService) Rates(ctx context.Context, currency string, assets ...string) (MiscRates, error) {
	q := url.Values{}
	setStr(q, "currency", currency)
	setStr(q, "assets", strings.Join(assets, ","))
	out, err := call[MiscRates](ctx, s.c, http.MethodGet, "/v1/assets/rates", q, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// Blockchains lists the blockchains Blockradar supports.
func (s *MiscService) Blockchains(ctx context.Context) ([]Blockchain, error) {
	return getSlice[Blockchain](ctx, s.c, "/v1/blockchains", nil)
}
