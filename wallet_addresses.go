package blockradar

import (
	"context"
	"net/http"
	"net/url"
)

// AddressesService calls the v1 endpoints in this file.
type AddressesService struct{ c *Client }

// AddressGenerateRequest creates a child address. All fields are optional.
type AddressGenerateRequest struct {
	Name                  string         `json:"name,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
	DisableAutoSweep      bool           `json:"disableAutoSweep,omitempty"`
	EnableGaslessWithdraw bool           `json:"enableGaslessWithdraw,omitempty"`
}

// AddressUpdateRequest changes a child address. Nil fields are left unchanged.
type AddressUpdateRequest struct {
	Name                  string         `json:"name,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
	IsActive              *bool          `json:"isActive,omitempty"`
	DisableAutoSweep      *bool          `json:"disableAutoSweep,omitempty"`
	EnableGaslessWithdraw *bool          `json:"enableGaslessWithdraw,omitempty"`
	ShowPrivateKey        *bool          `json:"showPrivateKey,omitempty"`
	PrivateKey            string         `json:"privateKey,omitempty"`
}

// AddressWhitelistRequest whitelists an external address. Address is required.
type AddressWhitelistRequest struct {
	Address               string         `json:"address"`
	Name                  string         `json:"name,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
	DisableAutoSweep      bool           `json:"disableAutoSweep,omitempty"`
	EnableGaslessWithdraw bool           `json:"enableGaslessWithdraw,omitempty"`
	ShowPrivateKey        bool           `json:"showPrivateKey,omitempty"`
	PrivateKey            string         `json:"privateKey,omitempty"`
}

// AddressAnalytics counts a wallet's addresses.
type AddressAnalytics struct {
	ActiveAddressesCount   int `json:"activeAddressesCount"`
	AddressLimit           int `json:"addressLimit"` // -1 means unlimited
	AddressesCount         int `json:"addressesCount"`
	ExternalAddressesCount int `json:"externalAddressesCount"`
	InactiveAddressesCount int `json:"inactiveAddressesCount"`
	InternalAddressesCount int `json:"internalAddressesCount"`
}

// AddressList is a page of child addresses with analytics.
type AddressList struct {
	List[Address]
	Analytics AddressAnalytics `json:"analytics"`
}

// AddressValidation is the result of Validate.
type AddressValidation struct {
	IsValid    bool   `json:"isValid"`
	Address    string `json:"address"` // EIP-55 checksummed on EVM chains
	Blockchain string `json:"blockchain"`
}

func (s *AddressesService) base(walletID string) string {
	return "/v1/wallets/" + seg(walletID) + "/addresses"
}

func (s *AddressesService) path(walletID, addressID string) string {
	return s.base(walletID) + "/" + seg(addressID)
}

// Generate creates a new child address under the wallet.
func (s *AddressesService) Generate(ctx context.Context, walletID string, r AddressGenerateRequest) (*Address, error) {
	return call[Address](ctx, s.c, http.MethodPost, s.base(walletID), nil, r)
}

// List returns the wallet's child addresses. Paging is not in the docs but the response is paginated.
func (s *AddressesService) List(ctx context.Context, walletID string, p PageParams) (*AddressList, error) {
	var out AddressList
	if err := s.c.do(ctx, http.MethodGet, s.base(walletID), p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns one child address. showPrivateKey adds the private key to the response.
func (s *AddressesService) Get(ctx context.Context, walletID, addressID string, showPrivateKey bool) (*Address, error) {
	q := url.Values{}
	if showPrivateKey {
		q.Set("showPrivateKey", "true")
	}
	return call[Address](ctx, s.c, http.MethodGet, s.path(walletID, addressID), q, nil)
}

// Update changes a child address (name, metadata, status, sweep and gasless settings).
func (s *AddressesService) Update(ctx context.Context, walletID, addressID string, r AddressUpdateRequest) (*Address, error) {
	return call[Address](ctx, s.c, http.MethodPatch, s.path(walletID, addressID), nil, r)
}

// Balance returns the address balance of one asset (assetId is the wallet asset ID).
func (s *AddressesService) Balance(ctx context.Context, walletID, addressID, assetID string) (*Balance, error) {
	q := url.Values{}
	setStr(q, "assetId", assetID)
	return call[Balance](ctx, s.c, http.MethodGet, s.path(walletID, addressID)+"/balance", q, nil)
}

// Balances returns the address balance of every enabled asset.
func (s *AddressesService) Balances(ctx context.Context, walletID, addressID string) ([]Balance, error) {
	return getSlice[Balance](ctx, s.c, s.path(walletID, addressID)+"/balances", nil)
}

// Transactions returns the transactions of a child address.
func (s *AddressesService) Transactions(ctx context.Context, walletID, addressID string, p TransactionListParams) (*List[TransactionDetail], error) {
	var out List[TransactionDetail]
	if err := s.c.do(ctx, http.MethodGet, s.path(walletID, addressID)+"/transactions", p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Transaction returns one transaction of a child address.
func (s *AddressesService) Transaction(ctx context.Context, walletID, addressID, transactionID string) (*TransactionDetail, error) {
	return call[TransactionDetail](ctx, s.c, http.MethodGet, s.path(walletID, addressID)+"/transactions/"+seg(transactionID), nil, nil)
}

// Whitelist adds an external address to the wallet so it is monitored.
func (s *AddressesService) Whitelist(ctx context.Context, walletID string, r AddressWhitelistRequest) (*Address, error) {
	return call[Address](ctx, s.c, http.MethodPost, s.base(walletID)+"/whitelist", nil, r)
}

// Unwhitelist removes a whitelisted external address.
func (s *AddressesService) Unwhitelist(ctx context.Context, walletID, addressID string) (*Address, error) {
	return call[Address](ctx, s.c, http.MethodDelete, s.base(walletID)+"/whitelist/"+seg(addressID), nil, nil)
}

// Validate checks offline whether address is well-formed for blockchain (a slug such as "ethereum").
// An invalid address is not an error: check IsValid.
func (s *AddressesService) Validate(ctx context.Context, address, blockchain string) (*AddressValidation, error) {
	q := url.Values{}
	setStr(q, "address", address)
	setStr(q, "blockchain", blockchain)
	return call[AddressValidation](ctx, s.c, http.MethodGet, "/v1/addresses/validate", q, nil)
}
