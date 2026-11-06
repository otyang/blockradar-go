package blockradar

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// VirtualAccountsService covers /v2/.../virtual-accounts: bank accounts that
// receive fiat and settle it as stablecoin into a wallet or child address.
//
// Typical flow:
//
//	Discovery -> Requirements -> Create (master wallet) or AddressCreate (child address)
type VirtualAccountsService struct{ c *Client }

// ---- types ----

type VirtualAccount struct {
	ID                 string    `json:"id"`
	Reference          string    `json:"reference"`
	AccountNumber      string    `json:"accountNumber"`
	AccountName        string    `json:"accountName"`
	BankName           string    `json:"bankName"`
	BankCode           string    `json:"bankCode"`
	Currency           string    `json:"currency"`
	InstitutionAddress string    `json:"institutionAddress"`
	AssetID            string    `json:"assetId"`
	ProviderID         string    `json:"providerId"`
	IsActive           bool      `json:"isActive"`
	Status             string    `json:"status"`
	Type               string    `json:"type"`
	Label              string    `json:"label"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// VirtualAccountCreateResult: Created is false when an existing account was returned.
type VirtualAccountCreateResult struct {
	Created        bool           `json:"created"`
	VirtualAccount VirtualAccount `json:"virtualAccount"`
}

type VirtualAccountDiscovery struct {
	Assets     []VirtualAccountAssetGroup        `json:"assets"`
	Currencies map[string]VirtualAccountCurrency `json:"currencies"`
}

type VirtualAccountAssetGroup struct {
	Asset      VirtualAccountAsset `json:"asset"`
	Currencies []string            `json:"currencies"`
}

type VirtualAccountAsset struct {
	ID         string                    `json:"id"`
	Symbol     string                    `json:"symbol"`
	Name       string                    `json:"name"`
	LogoURL    string                    `json:"logoUrl"`
	Blockchain *VirtualAccountBlockchain `json:"blockchain"`
}

type VirtualAccountBlockchain struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	Slug          string `json:"slug"`
	TokenStandard string `json:"tokenStandard"`
	LogoURL       string `json:"logoUrl"`
}

type VirtualAccountCurrency struct {
	Currency  string `json:"currency"`
	Code      string `json:"code"`
	ShortName string `json:"shortName"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
}

// VirtualAccountRequirements: if AdditionalDataRequired, collect fields
// matching AdditionalDataSchema and send them as AdditionalData on Create.
type VirtualAccountRequirements struct {
	Supported              bool       `json:"supported"`
	Unlocked               bool       `json:"unlocked"`
	AdditionalDataRequired bool       `json:"additionalDataRequired"`
	AdditionalDataSchema   JSONSchema `json:"additionalDataSchema"`
	Provider               string     `json:"provider"`
}

type VirtualAccountDestinationDiscovery struct {
	Supported    bool                        `json:"supported"`
	Provider     string                      `json:"provider"`
	Destinations []VirtualAccountDestination `json:"destinations"`
}

type VirtualAccountDestination struct {
	Wallet VirtualAccountDestinationWallet `json:"wallet"`
	Assets []VirtualAccountAssetGroup      `json:"assets"`
}

type VirtualAccountDestinationWallet struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Network    string                   `json:"network"` // "testnet" or "mainnet"
	Blockchain VirtualAccountBlockchain `json:"blockchain"`
}

// ---- params / requests ----

// VirtualAccountRequirementsParams: Currency is required.
type VirtualAccountRequirementsParams struct {
	Currency   string
	AssetID    string
	CustomerID string
	Provider   string
}

// CreateVirtualAccountRequest: Provider and Currency are required.
type CreateVirtualAccountRequest struct {
	Provider        string         `json:"provider"`
	Currency        string         `json:"currency"`
	AssetID         string         `json:"assetId,omitempty"`
	CustomerID      string         `json:"customerId,omitempty"`
	AdditionalData  map[string]any `json:"additionalData,omitempty"`
	Label           string         `json:"label,omitempty"`
	Code            string         `json:"code,omitempty"` // 2FA code, when required
	PasskeyResponse map[string]any `json:"passkeyResponse,omitempty"`
}

// RegenerateVirtualAccountRequest replaces an account; Reason is required.
type RegenerateVirtualAccountRequest struct {
	CreateVirtualAccountRequest
	Reason string `json:"reason"`
}

type UpdateVirtualAccountRequest struct {
	IsActive        bool           `json:"isActive"`
	Code            string         `json:"code,omitempty"`
	PasskeyResponse map[string]any `json:"passkeyResponse,omitempty"`
}

type ChangeVirtualAccountDestinationRequest struct {
	WalletID  string `json:"walletId"`
	AddressID string `json:"addressId,omitempty"`
	AssetID   string `json:"assetId"`
}

// VirtualAccountListParams filters and paginates. Zero values are omitted.
// StartDate and EndDate must be set together.
type VirtualAccountListParams struct {
	Search    string
	StartDate time.Time
	EndDate   time.Time
	Page      int // 1-based
	Limit     int // max 100
}

func (p VirtualAccountListParams) values() url.Values {
	q := url.Values{}
	setStr(q, "search", p.Search)
	setTime(q, "startDate", p.StartDate)
	setTime(q, "endDate", p.EndDate)
	setInt(q, "page", p.Page)
	setInt(q, "limit", p.Limit)
	return q
}

type ListVirtualAccountsParams struct {
	VirtualAccountListParams
	IsActive *bool // use blockradar.Bool(true)
}

func (p ListVirtualAccountsParams) values() url.Values {
	q := p.VirtualAccountListParams.values()
	setBool(q, "isActive", p.IsActive)
	return q
}

// ---- paths ----

func walletVA(walletID string) string {
	return "/v2/wallets/" + seg(walletID) + "/virtual-accounts"
}

func addressVA(walletID, addressID string) string {
	return "/v2/wallets/" + seg(walletID) + "/addresses/" + seg(addressID) + "/virtual-accounts"
}

// ---- master wallet endpoints ----

// Discovery lists assets and currencies available for virtual accounts.
func (s *VirtualAccountsService) Discovery(ctx context.Context, walletID, provider string) (*VirtualAccountDiscovery, error) {
	q := url.Values{}
	setStr(q, "provider", provider)
	return call[VirtualAccountDiscovery](ctx, s.c, http.MethodGet, walletVA(walletID)+"/discovery", q, nil)
}

// Requirements returns what is needed to create an account for a currency.
func (s *VirtualAccountsService) Requirements(ctx context.Context, walletID string, p VirtualAccountRequirementsParams) (*VirtualAccountRequirements, error) {
	q := url.Values{}
	q.Set("currency", p.Currency)
	setStr(q, "assetId", p.AssetID)
	setStr(q, "customerId", p.CustomerID)
	setStr(q, "provider", p.Provider)
	return call[VirtualAccountRequirements](ctx, s.c, http.MethodGet, walletVA(walletID)+"/requirements", q, nil)
}

// Create creates a virtual account that settles into the master wallet.
func (s *VirtualAccountsService) Create(ctx context.Context, walletID string, r CreateVirtualAccountRequest) (*VirtualAccountCreateResult, error) {
	return call[VirtualAccountCreateResult](ctx, s.c, http.MethodPost, walletVA(walletID), nil, r)
}

// List lists the master wallet's virtual accounts.
func (s *VirtualAccountsService) List(ctx context.Context, walletID string, p ListVirtualAccountsParams) (*Page[VirtualAccount], error) {
	var out Page[VirtualAccount]
	if err := s.c.do(ctx, http.MethodGet, walletVA(walletID), p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *VirtualAccountsService) Get(ctx context.Context, walletID, virtualAccountID string) (*VirtualAccount, error) {
	return call[VirtualAccount](ctx, s.c, http.MethodGet, walletVA(walletID)+"/"+seg(virtualAccountID), nil, nil)
}

// Update activates or deactivates an account.
func (s *VirtualAccountsService) Update(ctx context.Context, walletID, virtualAccountID string, r UpdateVirtualAccountRequest) (*VirtualAccount, error) {
	return call[VirtualAccount](ctx, s.c, http.MethodPatch, walletVA(walletID)+"/"+seg(virtualAccountID), nil, r)
}

// Regenerate replaces an account with a new one.
func (s *VirtualAccountsService) Regenerate(ctx context.Context, walletID, virtualAccountID string, r RegenerateVirtualAccountRequest) (*VirtualAccount, error) {
	return call[VirtualAccount](ctx, s.c, http.MethodPost, walletVA(walletID)+"/"+seg(virtualAccountID)+"/regenerate", nil, r)
}

// VirtualAccountDestinationDiscovery lists wallets and assets the account can settle into.
func (s *VirtualAccountsService) DestinationDiscovery(ctx context.Context, walletID, virtualAccountID string) (*VirtualAccountDestinationDiscovery, error) {
	return call[VirtualAccountDestinationDiscovery](ctx, s.c, http.MethodGet, walletVA(walletID)+"/"+seg(virtualAccountID)+"/destination/discovery", nil, nil)
}

// ChangeDestination changes where incoming funds settle.
func (s *VirtualAccountsService) ChangeDestination(ctx context.Context, walletID, virtualAccountID string, r ChangeVirtualAccountDestinationRequest) (*VirtualAccount, error) {
	return call[VirtualAccount](ctx, s.c, http.MethodPatch, walletVA(walletID)+"/"+seg(virtualAccountID)+"/destination", nil, r)
}

// ---- child address endpoints ----

// AddressCreate creates a virtual account that settles into a child address.
func (s *VirtualAccountsService) AddressCreate(ctx context.Context, walletID, addressID string, r CreateVirtualAccountRequest) (*VirtualAccountCreateResult, error) {
	return call[VirtualAccountCreateResult](ctx, s.c, http.MethodPost, addressVA(walletID, addressID), nil, r)
}

func (s *VirtualAccountsService) AddressList(ctx context.Context, walletID, addressID string, p ListVirtualAccountsParams) (*Page[VirtualAccount], error) {
	var out Page[VirtualAccount]
	if err := s.c.do(ctx, http.MethodGet, addressVA(walletID, addressID), p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *VirtualAccountsService) AddressGet(ctx context.Context, walletID, addressID, virtualAccountID string) (*VirtualAccount, error) {
	return call[VirtualAccount](ctx, s.c, http.MethodGet, addressVA(walletID, addressID)+"/"+seg(virtualAccountID), nil, nil)
}

func (s *VirtualAccountsService) AddressUpdate(ctx context.Context, walletID, addressID, virtualAccountID string, r UpdateVirtualAccountRequest) (*VirtualAccount, error) {
	return call[VirtualAccount](ctx, s.c, http.MethodPatch, addressVA(walletID, addressID)+"/"+seg(virtualAccountID), nil, r)
}

func (s *VirtualAccountsService) AddressRegenerate(ctx context.Context, walletID, addressID, virtualAccountID string, r RegenerateVirtualAccountRequest) (*VirtualAccount, error) {
	return call[VirtualAccount](ctx, s.c, http.MethodPost, addressVA(walletID, addressID)+"/"+seg(virtualAccountID)+"/regenerate", nil, r)
}

// AddressTransactions lists deposits received by a child address's virtual account.
func (s *VirtualAccountsService) AddressTransactions(ctx context.Context, walletID, addressID, virtualAccountID string, p VirtualAccountListParams) (*Page[FiatTransaction], error) {
	var out Page[FiatTransaction]
	if err := s.c.do(ctx, http.MethodGet, addressVA(walletID, addressID)+"/"+seg(virtualAccountID)+"/transactions", p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
