package blockradar

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"
)

// WithdrawFiatService covers /v2/.../withdraw/fiat: converting stablecoins
// to fiat and paying out to a bank account or other payment method.
//
// Typical flow:
//
//	Rates -> PaymentMethods -> PaymentMethodRequirements -> (ResolvePaymentMethod) -> Quote -> Execute
//
// Rates returns a SessionID. Pass it to every later call: the session fixes
// the asset, currency, amount, provider and customer chosen at the start.
type WithdrawFiatService struct{ c *Client }

// ---- types ----

// FiatAsset is an asset that can be converted to or from fiat. ID is the
// assetId to pass to Rates/Quote.
//
// The docs describe a flat asset, but the live API returns a wallet asset
// ({id, isActive, asset: {...}}), so the details may be in Asset instead.
// Use Info() to read them either way.
type FiatAsset struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	Address  string `json:"address"`
	Standard string `json:"standard"`
	Currency string `json:"currency"`
	IsActive bool   `json:"isActive"`
	Network  string `json:"network"`
	IsNative bool   `json:"isNative"`
	Asset    *Asset `json:"asset,omitempty"`
}

// Info returns the token details whichever shape the API used.
func (a FiatAsset) Info() Asset {
	if a.Asset != nil {
		return *a.Asset
	}
	return Asset{ID: a.ID, Name: a.Name, Symbol: a.Symbol, Decimals: a.Decimals, Address: a.Address,
		Standard: a.Standard, Currency: a.Currency, IsActive: a.IsActive, Network: a.Network, IsNative: a.IsNative}
}

type FiatCurrency struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	Decimals  int    `json:"decimals"`
	Symbol    string `json:"symbol"`
}

type WithdrawFiatRate struct {
	Rate             string    `json:"rate"`
	Amount           string    `json:"amount"`
	Unlocked         bool      `json:"unlocked"`
	SessionID        string    `json:"sessionId"`
	SessionExpiresAt time.Time `json:"sessionExpiresAt"`
	Fee              string    `json:"fee"`
	FeeCurrency      string    `json:"feeCurrency"`
	DebitAmount      string    `json:"debitAmount"`
	Provider         string    `json:"provider"`
}

type PaymentMethodDiscovery struct {
	PaymentMethods []PaymentMethod `json:"paymentMethods"`
	// Requirements is keyed by PaymentMethod.RequirementsRef.
	Requirements     map[string]SharedPaymentMethodRequirements `json:"requirements"`
	SessionID        string                                     `json:"sessionId"`
	SessionExpiresAt time.Time                                  `json:"sessionExpiresAt"`
}

// RequirementsFor returns the form schema for a payment method, if listed.
func (d *PaymentMethodDiscovery) RequirementsFor(m PaymentMethod) (SharedPaymentMethodRequirements, bool) {
	r, ok := d.Requirements[m.RequirementsRef]
	return r, ok
}

type PaymentMethod struct {
	ID                              string                  `json:"id"`
	SupportsPaymentMethodResolution bool                    `json:"supportsPaymentMethodResolution"`
	Type                            string                  `json:"type"`
	Country                         string                  `json:"country"`
	FiatCurrency                    string                  `json:"fiatCurrency"`
	PaymentMethodCategory           string                  `json:"paymentMethodCategory"`
	PaymentMethodType               string                  `json:"paymentMethodType"`
	ProcessingSeconds               *int                    `json:"processingSeconds"`
	Fee                             string                  `json:"fee"`
	Providers                       []PaymentMethodProvider `json:"providers"`
	RequirementsRef                 string                  `json:"requirementsRef"`
}

type PaymentMethodProvider struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Metadata map[string]any `json:"metadata"`
}

type SharedPaymentMethodRequirements struct {
	Schema     JSONSchema                       `json:"schema"`
	Metadata   map[string]any                   `json:"metadata"`
	Resolution *DynamicFormSubmissionResolution `json:"resolution"`
}

type PaymentMethodRequirements struct {
	PaymentMethodID  string                           `json:"paymentMethodId"`
	Schema           JSONSchema                       `json:"schema"`
	Metadata         map[string]any                   `json:"metadata"`
	Resolution       *DynamicFormSubmissionResolution `json:"resolution"`
	SessionID        string                           `json:"sessionId"`
	SessionExpiresAt time.Time                        `json:"sessionExpiresAt"`
}

// DynamicFormSubmissionResolution tells you which call resolves a form field
// on submit (usually ResolvePaymentMethod).
type DynamicFormSubmissionResolution struct {
	Trigger   string         `json:"trigger"`
	Method    string         `json:"method"`
	URL       string         `json:"url"`
	Body      map[string]any `json:"body"`
	InputPath string         `json:"inputPath"`
}

// PaymentMethodResolution is the result of ResolvePaymentMethod.
// Pass it back as WithdrawFiatQuoteRequest.ResolvedPaymentMethod.
type PaymentMethodResolution struct {
	PaymentMethod         string                 `json:"paymentMethod"`
	Data                  map[string]any         `json:"data"`
	Metadata              map[string]any         `json:"metadata"`
	SessionID             string                 `json:"sessionId"`
	SessionExpiresAt      time.Time              `json:"sessionExpiresAt"`
	Status                string                 `json:"status"` // "resolved" or "requires_action"
	ResolutionID          string                 `json:"resolutionId,omitempty"`
	PaymentMethodNextStep *PaymentMethodNextStep `json:"nextStep,omitempty"`
}

type PaymentMethodNextStep struct {
	ID     string     `json:"id"`
	Type   string     `json:"type"`
	Schema JSONSchema `json:"schema"`
}

type WithdrawFiatQuote struct {
	SessionID            string                  `json:"sessionId"`
	SessionExpiresAt     time.Time               `json:"sessionExpiresAt"`
	DebitAmount          string                  `json:"debitAmount"`
	MinAmount            string                  `json:"minAmount"`
	NetworkFee           string                  `json:"networkFee"`
	NetworkFeeInUSD      string                  `json:"networkFeeInUSD"`
	EstimatedArrivalTime int                     `json:"estimatedArrivalTime"`
	TransactionFee       string                  `json:"transactionFee"`
	NativeBalance        string                  `json:"nativeBalance"`
	NativeBalanceInUSD   string                  `json:"nativeBalanceInUSD"`
	AmountFiat           string                  `json:"amountFiat"`
	Deductions           []WithdrawFiatDeduction `json:"deductions"`
}

type WithdrawFiatDeduction struct {
	Code   string `json:"code"`
	Label  string `json:"label"`
	Amount string `json:"amount"`
}

// ---- params / requests ----

// WithdrawFiatRatesParams starts a withdrawal flow. AssetID, Currency and Amount are required.
type WithdrawFiatRatesParams struct {
	AssetID       string
	Currency      string // ISO 4217, e.g. "NGN"
	Amount        string
	AmountSide    AmountSide
	Provider      string
	CustomerID    string // set for customer-owned withdrawals
	PaymentMethod string
}

func (p WithdrawFiatRatesParams) values() url.Values {
	q := url.Values{}
	setStr(q, "assetId", p.AssetID)
	setStr(q, "currency", p.Currency)
	setStr(q, "amount", p.Amount)
	setStr(q, "amountSide", string(p.AmountSide))
	setStr(q, "provider", p.Provider)
	setStr(q, "customerId", p.CustomerID)
	setStr(q, "paymentMethod", p.PaymentMethod)
	return q
}

// WithdrawFiatFlowParams identifies a withdrawal flow. Set SessionID (recommended), or
// the routing fields (AssetID, Currency, Amount, ...).
type WithdrawFiatFlowParams struct {
	SessionID             string
	AssetID               string
	Currency              string
	Amount                string
	AmountSide            AmountSide
	Country               string // ISO 3166-1 alpha-2
	PaymentMethodCategory string
	Provider              string
	CustomerID            string
}

func (p WithdrawFiatFlowParams) values() url.Values {
	q := url.Values{}
	setStr(q, "sessionId", p.SessionID)
	setStr(q, "assetId", p.AssetID)
	setStr(q, "currency", p.Currency)
	setStr(q, "amount", p.Amount)
	setStr(q, "amountSide", string(p.AmountSide))
	setStr(q, "country", p.Country)
	setStr(q, "paymentMethodCategory", p.PaymentMethodCategory)
	setStr(q, "provider", p.Provider)
	setStr(q, "customerId", p.CustomerID)
	return q
}

// WithdrawFiatQuoteRequest asks for a withdrawal quote. PaymentMethod is required, plus
// either SessionID or AssetID+Currency+Amount.
type WithdrawFiatQuoteRequest struct {
	SessionID             string                   `json:"sessionId,omitempty"`
	CustomerID            string                   `json:"customerId,omitempty"`
	AssetID               string                   `json:"assetId,omitempty"`
	Amount                string                   `json:"amount,omitempty"`
	AmountSide            AmountSide               `json:"amountSide,omitempty"`
	Currency              string                   `json:"currency,omitempty"`
	PaymentMethod         string                   `json:"paymentMethod"`
	PaymentMethodData     map[string]any           `json:"paymentMethodData,omitempty"`
	IsGaslessWithdraw     bool                     `json:"isGaslessWithdraw,omitempty"`
	Provider              string                   `json:"provider,omitempty"`
	ResolvedPaymentMethod *PaymentMethodResolution `json:"resolvedPaymentMethod,omitempty"`
}

// WithdrawFiatExecuteRequest runs the withdrawal. Reuse the WithdrawFiatQuoteRequest you quoted with.
type WithdrawFiatExecuteRequest struct {
	WithdrawFiatQuoteRequest
	Reference       string         `json:"reference,omitempty"` // your idempotency/reconciliation reference
	Metadata        map[string]any `json:"metadata,omitempty"`
	Note            string         `json:"note,omitempty"`
	Code            string         `json:"code,omitempty"` // 2FA code, when required
	PasskeyResponse map[string]any `json:"passkeyResponse,omitempty"`
}

// ResolvePaymentMethodRequest resolves a dynamic payment-method field
// (e.g. looks up an account name from an account number).
type ResolvePaymentMethodRequest struct {
	SessionID         string                            `json:"sessionId,omitempty"`
	CustomerID        string                            `json:"customerId,omitempty"`
	AssetID           string                            `json:"assetId,omitempty"`
	Currency          string                            `json:"currency,omitempty"`
	Amount            string                            `json:"amount,omitempty"`
	AmountSide        AmountSide                        `json:"amountSide,omitempty"`
	PaymentMethod     string                            `json:"paymentMethod"`
	PaymentMethodData map[string]any                    `json:"paymentMethodData"`
	Provider          string                            `json:"provider,omitempty"`
	Continuation      *PaymentMethodResolveContinuation `json:"continuation,omitempty"`
}

// PaymentMethodResolveContinuation answers a "requires_action" resolution step.
type PaymentMethodResolveContinuation struct {
	ResolutionID string         `json:"resolutionId"`
	StepID       string         `json:"stepId"`
	Data         map[string]any `json:"data"`
}

// ErrMissingFlow means a request had neither a SessionID nor AssetID+Currency+Amount.
var ErrMissingFlow = errors.New("blockradar: set SessionID, or AssetID, Currency and Amount")

func checkFlow(sessionID, assetID, currency, amount string) error {
	if sessionID != "" || (assetID != "" && currency != "" && amount != "") {
		return nil
	}
	return ErrMissingFlow
}

// ---- master wallet endpoints ----

// SupportedAssets lists stablecoins that can be withdrawn to fiat.
// provider is optional ("" lets Blockradar choose).
func (s *WithdrawFiatService) SupportedAssets(ctx context.Context, walletID, provider string) ([]FiatAsset, error) {
	q := url.Values{}
	setStr(q, "provider", provider)
	return getSlice[FiatAsset](ctx, s.c, "/v2/wallets/"+seg(walletID)+"/withdraw/fiat/assets", q)
}

// Currencies lists supported payout currencies.
func (s *WithdrawFiatService) Currencies(ctx context.Context, walletID, provider string) ([]FiatCurrency, error) {
	q := url.Values{}
	setStr(q, "provider", provider)
	return getSlice[FiatCurrency](ctx, s.c, "/v2/wallets/"+seg(walletID)+"/withdraw/fiat/currencies", q)
}

// Rates gets the exchange rate and opens a session. Keep rate.SessionID.
func (s *WithdrawFiatService) Rates(ctx context.Context, walletID string, p WithdrawFiatRatesParams) (*WithdrawFiatRate, error) {
	if p.AssetID == "" || p.Currency == "" || p.Amount == "" {
		return nil, errors.New("blockradar: Rates needs AssetID, Currency and Amount")
	}
	return call[WithdrawFiatRate](ctx, s.c, http.MethodGet, "/v2/wallets/"+seg(walletID)+"/withdraw/fiat/rates", p.values(), nil)
}

// PaymentMethods lists payout methods and their form requirements.
func (s *WithdrawFiatService) PaymentMethods(ctx context.Context, walletID string, p WithdrawFiatFlowParams) (*PaymentMethodDiscovery, error) {
	return call[PaymentMethodDiscovery](ctx, s.c, http.MethodGet, "/v2/wallets/"+seg(walletID)+"/withdraw/fiat/payment-methods", p.values(), nil)
}

// PaymentMethodRequirements returns the JSON Schema of fields needed for paymentMethod.
func (s *WithdrawFiatService) PaymentMethodRequirements(ctx context.Context, walletID, paymentMethod string, p WithdrawFiatFlowParams) (*PaymentMethodRequirements, error) {
	q := p.values()
	q.Set("paymentMethod", paymentMethod)
	return call[PaymentMethodRequirements](ctx, s.c, http.MethodGet, "/v2/wallets/"+seg(walletID)+"/withdraw/fiat/payment-method-requirements", q, nil)
}

// ResolvePaymentMethod resolves a dynamic payment-method field.
func (s *WithdrawFiatService) ResolvePaymentMethod(ctx context.Context, walletID string, r ResolvePaymentMethodRequest) (*PaymentMethodResolution, error) {
	if err := checkFlow(r.SessionID, r.AssetID, r.Currency, r.Amount); err != nil {
		return nil, err
	}
	return call[PaymentMethodResolution](ctx, s.c, http.MethodPost, "/v2/wallets/"+seg(walletID)+"/withdraw/fiat/payment-method/resolve", nil, r)
}

// Quote prices a withdrawal from the master wallet.
func (s *WithdrawFiatService) Quote(ctx context.Context, walletID string, r WithdrawFiatQuoteRequest) (*WithdrawFiatQuote, error) {
	if err := checkFlow(r.SessionID, r.AssetID, r.Currency, r.Amount); err != nil {
		return nil, err
	}
	return call[WithdrawFiatQuote](ctx, s.c, http.MethodPost, "/v2/wallets/"+seg(walletID)+"/withdraw/fiat/quote", nil, r)
}

// Execute sends a fiat withdrawal from the master wallet.
func (s *WithdrawFiatService) Execute(ctx context.Context, walletID string, r WithdrawFiatExecuteRequest) (*FiatTransaction, error) {
	if err := checkFlow(r.SessionID, r.AssetID, r.Currency, r.Amount); err != nil {
		return nil, err
	}
	return call[FiatTransaction](ctx, s.c, http.MethodPost, "/v2/wallets/"+seg(walletID)+"/withdraw/fiat/execute", nil, r)
}

// ---- child address endpoints ----

// AddressQuote prices a withdrawal from a child address.
func (s *WithdrawFiatService) AddressQuote(ctx context.Context, walletID, addressID string, r WithdrawFiatQuoteRequest) (*WithdrawFiatQuote, error) {
	if err := checkFlow(r.SessionID, r.AssetID, r.Currency, r.Amount); err != nil {
		return nil, err
	}
	return call[WithdrawFiatQuote](ctx, s.c, http.MethodPost, "/v2/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/withdraw/fiat/quote", nil, r)
}

// AddressExecute sends a fiat withdrawal from a child address.
func (s *WithdrawFiatService) AddressExecute(ctx context.Context, walletID, addressID string, r WithdrawFiatExecuteRequest) (*FiatTransaction, error) {
	if err := checkFlow(r.SessionID, r.AssetID, r.Currency, r.Amount); err != nil {
		return nil, err
	}
	return call[FiatTransaction](ctx, s.c, http.MethodPost, "/v2/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/withdraw/fiat/execute", nil, r)
}
