package blockradar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DepositFiatService covers /v1/.../deposit/fiat: collecting fiat from a payer
// and settling it as stablecoin into a wallet or child address (the on-ramp
// mirror of WithdrawFiat).
//
// Typical flow:
//
//	SupportedAssets -> Currencies -> Rails -> Quote -> (ResolvePaymentOrderRequirements) -> CreateOrder
//
// Quote returns a SessionID. Pass it to later calls: the session fixes the
// asset, currency, amount, rail, provider and customer chosen at the start.
// Then show the order's PaymentInstructions to the payer and track it with
// GetOrder / RefreshOrder (or payment-order webhooks).
type DepositFiatService struct{ c *Client }

// ---- types ----

// PaymentOrderRail is a fiat collection method.
type PaymentOrderRail string

const (
	PaymentOrderRailOneTimeVirtualAccount PaymentOrderRail = "one_time_virtual_account"
	PaymentOrderRailPaymentLink           PaymentOrderRail = "payment_link"
	PaymentOrderRailQRCode                PaymentOrderRail = "qr_code"
)

// DepositFiatProvider is the provider that collects the fiat. ID is empty on rails.
type DepositFiatProvider struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// DepositFiatRail is a supported collection rail with its fiat amount limits.
type DepositFiatRail struct {
	Rail               PaymentOrderRail    `json:"rail"`
	SourceCurrencyCode string              `json:"sourceCurrencyCode"` // fiat paid, e.g. "NGN"
	TargetCurrencyCode string              `json:"targetCurrencyCode"` // stablecoin received, e.g. "USDC"
	MinAmount          string              `json:"minAmount"`
	MaxAmount          string              `json:"maxAmount"`
	Provider           DepositFiatProvider `json:"provider"`
}

// DepositFiatQuote prices a fiat deposit and opens a session.
type DepositFiatQuote struct {
	Amount                     string                  `json:"amount"`            // stablecoin to settle
	AmountFiat                 string                  `json:"amountFiat"`        // fiat before fees
	FeeFiat                    string                  `json:"feeFiat"`           // fiat fee
	AmountFiatPayable          string                  `json:"amountFiatPayable"` // total fiat the payer pays
	Rate                       string                  `json:"rate"`
	Unlocked                   bool                    `json:"unlocked"`
	AdditionalDataRequirements DepositFiatRequirements `json:"additionalDataRequirements"`
	SessionID                  string                  `json:"sessionId"`
	SessionExpiresAt           time.Time               `json:"sessionExpiresAt"`
}

// DepositFiatRequirements describes extra data to send as PaymentOrderCreateRequest.AdditionalData.
type DepositFiatRequirements struct {
	Schema JSONSchema     `json:"schema"` // may be JSON null
	Data   map[string]any `json:"data"`   // previously saved values, usable as form defaults
}

// PaymentOrderResolution is the result of ResolvePaymentOrderRequirements.
type PaymentOrderResolution struct {
	Data             map[string]any `json:"data"`     // merge into the create-order AdditionalData
	Metadata         map[string]any `json:"metadata"` // opaque provider metadata
	SessionID        string         `json:"sessionId"`
	SessionExpiresAt time.Time      `json:"sessionExpiresAt"`
}

// PaymentOrder is a fiat deposit order with the instructions the payer follows.
type PaymentOrder struct {
	ID                  string                    `json:"id"`
	Type                string                    `json:"type"` // "deposit"
	Rail                PaymentOrderRail          `json:"rail"`
	Reference           string                    `json:"reference"`
	ProviderReference   string                    `json:"providerReference"`
	TransactionID       string                    `json:"transactionId"`
	Status              string                    `json:"status"` // pending, paid, processing, expired, failed, cancelled
	Currency            string                    `json:"currency"`
	Amount              string                    `json:"amount"`
	AmountFiat          string                    `json:"amountFiat"`
	FeeFiat             string                    `json:"feeFiat"`
	AmountFiatPayable   string                    `json:"amountFiatPayable"`
	AmountFiatReceived  string                    `json:"amountFiatReceived"`
	AmountSettled       string                    `json:"amountSettled"`
	ExpiresAt           *time.Time                `json:"expiresAt"`
	PaymentInstructions *PaymentOrderInstructions `json:"paymentInstructions"`
	Metadata            map[string]any            `json:"metadata"`
	Provider            DepositFiatProvider       `json:"provider"`
	Wallet              PaymentOrderAddress       `json:"wallet"`
	Address             *PaymentOrderAddress      `json:"address"` // set for child-address orders
	Asset               PaymentOrderAsset         `json:"asset"`
	Customer            *PaymentOrderCustomer     `json:"customer"`
	CreatedAt           time.Time                 `json:"createdAt"`
	UpdatedAt           time.Time                 `json:"updatedAt"`
}

// PaymentOrderAddress is the wallet or child address an order settles into.
type PaymentOrderAddress struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// PaymentOrderAsset is the stablecoin an order settles in.
type PaymentOrderAsset struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// PaymentOrderCustomer: responses carry only ID; on create, Name/Email/Phone are optional payer contact details.
type PaymentOrderCustomer struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"` // international format, e.g. "+2348012345678"
}

// PaymentOrderInstructions tells the payer how to pay. Which of Bank, PaymentLink
// or QRCode is set depends on Type.
type PaymentOrderInstructions struct {
	Type        PaymentOrderRail    `json:"type"`
	Bank        *PaymentOrderBank   `json:"bank"`
	PaymentLink *PaymentOrderLink   `json:"paymentLink"`
	QRCode      *PaymentOrderQRCode `json:"qrCode"`
	ExpiresAt   *time.Time          `json:"expiresAt"`
	Metadata    map[string]any      `json:"metadata"`
}

// PaymentOrderBank is a one-time bank account to pay into.
type PaymentOrderBank struct {
	Name          string `json:"name"`
	Code          string `json:"code"`
	AccountName   string `json:"accountName"`
	AccountNumber string `json:"accountNumber"`
}

// PaymentOrderLink is a hosted payment page.
type PaymentOrderLink struct {
	URL string `json:"url"`
}

// PaymentOrderQRCode is a QR code to scan and pay.
type PaymentOrderQRCode struct {
	Payload  string `json:"payload"`
	ImageURL string `json:"imageUrl"`
}

// PaymentOrderPage is a page of payment orders plus status counts.
// The live API returns {"data": {"items", "meta", "analytics"}}; the docs show a flat array.
type PaymentOrderPage struct {
	Items     []PaymentOrder        `json:"items"`
	Meta      ListMeta              `json:"meta"`
	Analytics PaymentOrderAnalytics `json:"analytics"`
}

// PaymentOrderAnalytics counts orders by status.
type PaymentOrderAnalytics struct {
	TotalOrdersCount           int `json:"totalOrdersCount"`
	TotalPendingOrdersCount    int `json:"totalPendingOrdersCount"`
	TotalPaidOrdersCount       int `json:"totalPaidOrdersCount"`
	TotalProcessingOrdersCount int `json:"totalProcessingOrdersCount"`
	TotalExpiredOrdersCount    int `json:"totalExpiredOrdersCount"`
	TotalFailedOrdersCount     int `json:"totalFailedOrdersCount"`
	TotalCancelledOrdersCount  int `json:"totalCancelledOrdersCount"`
}

// ---- params / requests ----

// DepositFiatRailsParams selects a route. AssetID, Currency and Amount are required.
type DepositFiatRailsParams struct {
	AssetID  string
	Currency string // ISO 4217, e.g. "NGN"
	Amount   string
	Provider string
}

func (p DepositFiatRailsParams) values() url.Values {
	q := url.Values{}
	setStr(q, "assetId", p.AssetID)
	setStr(q, "currency", p.Currency)
	setStr(q, "amount", p.Amount)
	setStr(q, "provider", p.Provider)
	return q
}

// DepositFiatQuoteRequest asks for a deposit quote. Set SessionID, or AssetID+Currency+Amount+Rail.
type DepositFiatQuoteRequest struct {
	SessionID  string           `json:"sessionId,omitempty"`
	CustomerID string           `json:"customerId,omitempty"`
	AssetID    string           `json:"assetId,omitempty"`
	Currency   string           `json:"currency,omitempty"`
	Amount     string           `json:"amount,omitempty"` // stablecoin amount to receive
	Rail       PaymentOrderRail `json:"rail,omitempty"`
	Provider   string           `json:"provider,omitempty"`
}

// PaymentOrderResolveRequest resolves a dynamic requirements field (x-resolution).
// Set SessionID (recommended), or AssetID+Currency+Amount+Rail.
type PaymentOrderResolveRequest struct {
	SessionID      string           `json:"sessionId,omitempty"`
	CustomerID     string           `json:"customerId,omitempty"`
	AssetID        string           `json:"assetId,omitempty"`
	Currency       string           `json:"currency,omitempty"`
	Amount         string           `json:"amount,omitempty"`
	Rail           PaymentOrderRail `json:"rail,omitempty"`
	AdditionalData map[string]any   `json:"additionalData,omitempty"` // current form values
	Provider       string           `json:"provider,omitempty"`
}

// PaymentOrderCreateRequest creates a payment order. Set SessionID (recommended),
// or the deprecated routing fields AssetID+Currency+Amount+Rail.
type PaymentOrderCreateRequest struct {
	SessionID           string                `json:"sessionId,omitempty"`
	CustomerID          string                `json:"customerId,omitempty"`
	AssetID             string                `json:"assetId,omitempty"`
	Currency            string                `json:"currency,omitempty"`
	Amount              string                `json:"amount,omitempty"`
	Rail                PaymentOrderRail      `json:"rail,omitempty"`
	Provider            string                `json:"provider,omitempty"`
	Reference           string                `json:"reference,omitempty"`           // your idempotency/reconciliation reference
	ValidityTimeMinutes json.Number           `json:"validityTimeMinutes,omitempty"` // 30-1440
	Customer            *PaymentOrderCustomer `json:"customer,omitempty"`            // payer contact details
	AdditionalData      map[string]any        `json:"additionalData,omitempty"`      // values for the quote's requirements schema
	Metadata            map[string]any        `json:"metadata,omitempty"`
	Code                string                `json:"code,omitempty"` // 2FA code, when required
	PasskeyResponse     map[string]any        `json:"passkeyResponse,omitempty"`
}

// PaymentOrderListParams filters ListOrders / AddressListOrders. StartDate and EndDate go together.
type PaymentOrderListParams struct {
	Status    string // pending, paid, processing, expired, failed, cancelled
	Rail      PaymentOrderRail
	Currency  string
	AssetID   string
	Search    string // reference, label, account name or number
	StartDate time.Time
	EndDate   time.Time
	Page      int
	Limit     int // max 100
}

func (p PaymentOrderListParams) values() url.Values {
	q := url.Values{}
	setStr(q, "status", p.Status)
	setStr(q, "rail", string(p.Rail))
	setStr(q, "currency", p.Currency)
	setStr(q, "assetId", p.AssetID)
	setStr(q, "search", p.Search)
	setTime(q, "startDate", p.StartDate)
	setTime(q, "endDate", p.EndDate)
	setInt(q, "page", p.Page)
	setInt(q, "limit", p.Limit)
	return q
}

// errMissingDepositFlow matches ErrMissingFlow; deposit routing also needs Rail.
var errMissingDepositFlow = fmt.Errorf("%w (deposit fiat also needs Rail)", ErrMissingFlow)

func checkDepositFlow(sessionID, assetID, currency, amount string, rail PaymentOrderRail) error {
	if sessionID != "" || (assetID != "" && currency != "" && amount != "" && rail != "") {
		return nil
	}
	return errMissingDepositFlow
}

func depositFiatPath(walletID string) string {
	return "/v1/wallets/" + seg(walletID) + "/deposit/fiat"
}

func addressDepositFiatPath(walletID, addressID string) string {
	return "/v1/wallets/" + seg(walletID) + "/addresses/" + seg(addressID) + "/deposit/fiat"
}

// ---- discovery ----

// SupportedAssets lists stablecoins that fiat deposits can settle into.
// provider is optional ("" lets Blockradar choose).
func (s *DepositFiatService) SupportedAssets(ctx context.Context, walletID, provider string) ([]FiatAsset, error) {
	q := url.Values{}
	setStr(q, "provider", provider)
	return getSlice[FiatAsset](ctx, s.c, depositFiatPath(walletID)+"/assets", q)
}

// Currencies lists fiat currencies a payer can pay in for assetID. provider is optional.
func (s *DepositFiatService) Currencies(ctx context.Context, walletID, assetID, provider string) ([]FiatCurrency, error) {
	if assetID == "" {
		return nil, errors.New("blockradar: Currencies needs assetID")
	}
	q := url.Values{}
	setStr(q, "assetId", assetID)
	setStr(q, "provider", provider)
	return getSlice[FiatCurrency](ctx, s.c, depositFiatPath(walletID)+"/currencies", q)
}

// Rails lists collection rails (bank transfer, payment link, QR) for a route.
func (s *DepositFiatService) Rails(ctx context.Context, walletID string, p DepositFiatRailsParams) ([]DepositFiatRail, error) {
	if p.AssetID == "" || p.Currency == "" || p.Amount == "" {
		return nil, errors.New("blockradar: Rails needs AssetID, Currency and Amount")
	}
	out, err := call[struct {
		Rails []DepositFiatRail `json:"rails"`
	}](ctx, s.c, http.MethodGet, depositFiatPath(walletID)+"/rails", p.values(), nil)
	if err != nil {
		return nil, err
	}
	return out.Rails, nil
}

// ResolvePaymentOrderRequirements resolves a dynamic requirements field (e.g. an account name).
func (s *DepositFiatService) ResolvePaymentOrderRequirements(ctx context.Context, walletID string, r PaymentOrderResolveRequest) (*PaymentOrderResolution, error) {
	if err := checkDepositFlow(r.SessionID, r.AssetID, r.Currency, r.Amount, r.Rail); err != nil {
		return nil, err
	}
	return call[PaymentOrderResolution](ctx, s.c, http.MethodPost, depositFiatPath(walletID)+"/payment-order-requirements/resolve", nil, r)
}

// ---- master wallet endpoints ----

// Quote prices a fiat deposit into the master wallet and opens a session.
func (s *DepositFiatService) Quote(ctx context.Context, walletID string, r DepositFiatQuoteRequest) (*DepositFiatQuote, error) {
	if err := checkDepositFlow(r.SessionID, r.AssetID, r.Currency, r.Amount, r.Rail); err != nil {
		return nil, err
	}
	return call[DepositFiatQuote](ctx, s.c, http.MethodPost, depositFiatPath(walletID)+"/quote", nil, r)
}

// CreateOrder creates a payment order settling into the master wallet.
func (s *DepositFiatService) CreateOrder(ctx context.Context, walletID string, r PaymentOrderCreateRequest) (*PaymentOrder, error) {
	if err := checkDepositFlow(r.SessionID, r.AssetID, r.Currency, r.Amount, r.Rail); err != nil {
		return nil, err
	}
	return call[PaymentOrder](ctx, s.c, http.MethodPost, depositFiatPath(walletID)+"/orders", nil, r)
}

// ListOrders lists the wallet's payment orders.
func (s *DepositFiatService) ListOrders(ctx context.Context, walletID string, p PaymentOrderListParams) (*PaymentOrderPage, error) {
	return call[PaymentOrderPage](ctx, s.c, http.MethodGet, depositFiatPath(walletID)+"/orders", p.values(), nil)
}

// GetOrder fetches a payment order.
func (s *DepositFiatService) GetOrder(ctx context.Context, walletID, orderID string) (*PaymentOrder, error) {
	return call[PaymentOrder](ctx, s.c, http.MethodGet, depositFiatPath(walletID)+"/orders/"+seg(orderID), nil, nil)
}

// CancelOrder cancels a pending payment order.
func (s *DepositFiatService) CancelOrder(ctx context.Context, walletID, orderID string) (*PaymentOrder, error) {
	return call[PaymentOrder](ctx, s.c, http.MethodPost, depositFiatPath(walletID)+"/orders/"+seg(orderID)+"/cancel", nil, nil)
}

// RefreshOrder re-syncs a payment order's status with the provider.
func (s *DepositFiatService) RefreshOrder(ctx context.Context, walletID, orderID string) (*PaymentOrder, error) {
	return call[PaymentOrder](ctx, s.c, http.MethodPost, depositFiatPath(walletID)+"/orders/"+seg(orderID)+"/refresh", nil, nil)
}

// ---- child address endpoints ----

// AddressQuote prices a fiat deposit into a child address and opens a session.
func (s *DepositFiatService) AddressQuote(ctx context.Context, walletID, addressID string, r DepositFiatQuoteRequest) (*DepositFiatQuote, error) {
	if err := checkDepositFlow(r.SessionID, r.AssetID, r.Currency, r.Amount, r.Rail); err != nil {
		return nil, err
	}
	return call[DepositFiatQuote](ctx, s.c, http.MethodPost, addressDepositFiatPath(walletID, addressID)+"/quote", nil, r)
}

// AddressCreateOrder creates a payment order settling into a child address.
func (s *DepositFiatService) AddressCreateOrder(ctx context.Context, walletID, addressID string, r PaymentOrderCreateRequest) (*PaymentOrder, error) {
	if err := checkDepositFlow(r.SessionID, r.AssetID, r.Currency, r.Amount, r.Rail); err != nil {
		return nil, err
	}
	return call[PaymentOrder](ctx, s.c, http.MethodPost, addressDepositFiatPath(walletID, addressID)+"/orders", nil, r)
}

// AddressListOrders lists a child address's payment orders.
func (s *DepositFiatService) AddressListOrders(ctx context.Context, walletID, addressID string, p PaymentOrderListParams) (*PaymentOrderPage, error) {
	return call[PaymentOrderPage](ctx, s.c, http.MethodGet, addressDepositFiatPath(walletID, addressID)+"/orders", p.values(), nil)
}
