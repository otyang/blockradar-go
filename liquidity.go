package blockradar

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LiquidityPoolService covers /v1/rates: the exchange rates you quote as a
// liquidity provider for asset pairs (e.g. USDT -> cNGN).
type LiquidityPoolService struct{ c *Client }

// ---- types ----

// LiquidityRate is a rate you offer for a pair within an amount band.
// Updates create a new version linked by PreviousRateID / RootRateID.
type LiquidityRate struct {
	ID                 string     `json:"id"`
	FromAsset          string     `json:"fromAsset"`
	ToAsset            string     `json:"toAsset"`
	Network            string     `json:"network"`
	Rate               Amount     `json:"rate"`
	MinAmount          Amount     `json:"minAmount"`
	MaxAmount          Amount     `json:"maxAmount"` // "" when unbounded
	IsActive           bool       `json:"isActive"`
	Status             string     `json:"status"` // "active" or "deactivated"
	Version            int        `json:"version"`
	PreviousRateID     string     `json:"previousRateId"`
	RootRateID         string     `json:"rootRateId"`
	CreatedBy          string     `json:"createdBy"`
	DeactivatedBy      string     `json:"deactivatedBy"`
	DeactivatedAt      *time.Time `json:"deactivatedAt"`
	DeactivationReason string     `json:"deactivationReason"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// LiquidityPairCheck lists your active rates for one pair.
type LiquidityPairCheck struct {
	FromAsset       string          `json:"fromAsset"`
	ToAsset         string          `json:"toAsset"`
	HasActiveRates  bool            `json:"hasActiveRates"`
	ActiveRateCount int             `json:"activeRateCount"`
	Rates           []LiquidityRate `json:"rates"`
}

// LiquidityBenchmark is the best competing rate for a pair.
type LiquidityBenchmark struct {
	FromAsset string `json:"fromAsset"`
	ToAsset   string `json:"toAsset"`
	BestRate  Amount `json:"bestRate"`
}

// LiquidityTreasuryBalance is your balance of one asset used by active rates.
type LiquidityTreasuryBalance struct {
	Asset                 string                   `json:"asset"`
	TotalBalance          Amount                   `json:"totalBalance"`
	TotalConvertedBalance Amount                   `json:"totalConvertedBalance"`
	Chains                []LiquidityTreasuryChain `json:"chains"`
}

// LiquidityTreasuryChain is the per-blockchain part of a LiquidityTreasuryBalance.
type LiquidityTreasuryChain struct {
	Blockchain       string `json:"blockchain"`
	Balance          Amount `json:"balance"`
	ConvertedBalance Amount `json:"convertedBalance"`
	WalletAddress    string `json:"walletAddress"`
}

// ---- requests / params ----

// LiquidityCreateRateRequest: all fields but MaxAmount are required. Assets are symbols.
type LiquidityCreateRateRequest struct {
	FromAsset string `json:"fromAsset"`
	ToAsset   string `json:"toAsset"`
	Rate      string `json:"rate"`
	MinAmount string `json:"minAmount"`
	MaxAmount string `json:"maxAmount,omitempty"`
}

// LiquidityUpdateRateRequest is a partial update; empty fields are left unchanged.
type LiquidityUpdateRateRequest struct {
	Rate      string `json:"rate,omitempty"`
	MinAmount string `json:"minAmount,omitempty"`
}

// LiquidityDeactivateRateRequest: Reason is required.
type LiquidityDeactivateRateRequest struct {
	Reason string `json:"reason"`
}

// LiquidityBenchmarkParams: set FromAsset and ToAsset, or Pairs (max 20, "FROM:TO").
type LiquidityBenchmarkParams struct {
	FromAsset string
	ToAsset   string
	Amount    string   // only rates whose band covers this amount
	Pairs     []string // e.g. {"USDT:cNGN", "cNGN:USDT"}
}

func (p LiquidityBenchmarkParams) values() url.Values {
	q := url.Values{}
	setStr(q, "fromAsset", p.FromAsset)
	setStr(q, "toAsset", p.ToAsset)
	setStr(q, "amount", p.Amount)
	setStr(q, "pairs", strings.Join(p.Pairs, ","))
	return q
}

func rateURL(id string) string { return "/v1/rates/" + seg(id) }

// ---- endpoints ----

// Create creates a rate for an asset pair.
func (s *LiquidityPoolService) Create(ctx context.Context, r LiquidityCreateRateRequest) (*LiquidityRate, error) {
	return call[LiquidityRate](ctx, s.c, http.MethodPost, "/v1/rates", nil, r)
}

// Get returns a rate by ID.
func (s *LiquidityPoolService) Get(ctx context.Context, rateID string) (*LiquidityRate, error) {
	return call[LiquidityRate](ctx, s.c, http.MethodGet, rateURL(rateID), nil, nil)
}

// Update changes a rate's value or minimum amount (creates a new version).
func (s *LiquidityPoolService) Update(ctx context.Context, rateID string, r LiquidityUpdateRateRequest) (*LiquidityRate, error) {
	return call[LiquidityRate](ctx, s.c, http.MethodPatch, rateURL(rateID), nil, r)
}

// Deactivate stops quoting a rate.
func (s *LiquidityPoolService) Deactivate(ctx context.Context, rateID string, r LiquidityDeactivateRateRequest) (*LiquidityRate, error) {
	return call[LiquidityRate](ctx, s.c, http.MethodPatch, rateURL(rateID)+"/deactivate", nil, r)
}

// Reactivate resumes quoting a deactivated rate.
func (s *LiquidityPoolService) Reactivate(ctx context.Context, rateID string) (*LiquidityRate, error) {
	return call[LiquidityRate](ctx, s.c, http.MethodPatch, rateURL(rateID)+"/reactivate", nil, nil)
}

// History lists the versions of a rate. Page/limit are not documented but the response is paginated.
func (s *LiquidityPoolService) History(ctx context.Context, rateID string, p PageParams) (*List[LiquidityRate], error) {
	var out List[LiquidityRate]
	if err := s.c.do(ctx, http.MethodGet, rateURL(rateID)+"/history", p.values(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CheckPair returns your active rates for a pair of asset symbols.
func (s *LiquidityPoolService) CheckPair(ctx context.Context, fromAsset, toAsset string) (*LiquidityPairCheck, error) {
	q := url.Values{}
	q.Set("fromAsset", fromAsset)
	q.Set("toAsset", toAsset)
	return call[LiquidityPairCheck](ctx, s.c, http.MethodGet, "/v1/rates/check-pair", q, nil)
}

// MarketBenchmark returns the best competing rates. The API sends one object
// for a single pair and an array for Pairs; both come back as a slice.
func (s *LiquidityPoolService) MarketBenchmark(ctx context.Context, p LiquidityBenchmarkParams) ([]LiquidityBenchmark, error) {
	out, err := call[oneOrMany[LiquidityBenchmark]](ctx, s.c, http.MethodGet, "/v1/rates/market-benchmark", p.values(), nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// TreasuryBalances returns your balances of every asset in your active rates.
func (s *LiquidityPoolService) TreasuryBalances(ctx context.Context) ([]LiquidityTreasuryBalance, error) {
	return getSlice[LiquidityTreasuryBalance](ctx, s.c, "/v1/rates/treasury-balances", nil)
}
