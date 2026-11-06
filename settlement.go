package blockradar

import (
	"context"
	"net/http"
	"time"
)

// AutoSettlementsService covers /v1/.../auto-settlements: rules that
// automatically withdraw, swap, route to Gateway or move into Earn whatever
// lands in a master wallet or child address.
type AutoSettlementsService struct{ c *Client }

// ---- types ----

// AutoSettlement is the auto-settlement switch of a wallet or address plus its rules.
type AutoSettlement struct {
	IsActive bool                 `json:"isActive"`
	Name     string               `json:"name"`
	Rules    []AutoSettlementRule `json:"rules"`
}

// AutoSettlementRule is a stored auto-settlement rule.
type AutoSettlementRule struct {
	ID                   string                     `json:"id"`
	Type                 string                     `json:"type"` // withdraw, swap, gateway, earn (empty on legacy rules)
	Name                 string                     `json:"name"`
	IsActive             bool                       `json:"isActive"`
	IsGateway            bool                       `json:"isGateway"`
	IsReward             bool                       `json:"isReward"`
	RewardProvider       string                     `json:"rewardProvider"`
	RewardType           string                     `json:"rewardType"`
	ProductKey           string                     `json:"productKey"`
	UseTransactionAmount bool                       `json:"useTransactionAmount"`
	DeductionPercentage  string                     `json:"deductionPercentage"`
	Order                string                     `json:"order"`
	SlippageTolerance    string                     `json:"slippageTolerance"`
	Inheritance          *AutoSettlementInheritance `json:"inheritance"`
	Source               AutoSettlementSource       `json:"source"`
	Destination          AutoSettlementDestination  `json:"destination"`
	CreatedAt            time.Time                  `json:"createdAt"`
	UpdatedAt            time.Time                  `json:"updatedAt"`
}

// Request copies a stored rule into a full rule body, for read-modify-write
// with AddressUpdateRule (which resets any field you leave out).
func (r AutoSettlementRule) Request() AutoSettlementRuleRequest {
	return AutoSettlementRuleRequest{
		ProductKey:           r.ProductKey,
		Type:                 r.Type,
		Name:                 r.Name,
		IsActive:             Bool(r.IsActive),
		IsGateway:            r.IsGateway,
		IsReward:             r.IsReward,
		RewardProvider:       r.RewardProvider,
		RewardType:           r.RewardType,
		UseTransactionAmount: r.UseTransactionAmount,
		DeductionPercentage:  r.DeductionPercentage,
		Order:                r.Order,
		SlippageTolerance:    r.SlippageTolerance,
		Inheritance:          r.Inheritance,
		Source:               r.Source,
		Destination:          r.Destination,
	}
}

// AutoSettlementSource says which deposits a rule settles.
type AutoSettlementSource struct {
	Assets     []string `json:"assets"` // asset symbols, e.g. ["USDC"]
	Blockchain string   `json:"blockchain,omitempty"`
	MinAmount  string   `json:"minAmount,omitempty"` // default "0"
	MaxAmount  string   `json:"maxAmount,omitempty"` // default "-1" (unlimited)
}

// AutoSettlementDestination says where settled funds go.
type AutoSettlementDestination struct {
	Blockchain string `json:"blockchain"`
	Asset      string `json:"asset"`
	Address    string `json:"address,omitempty"`   // required for withdraw rules
	IsGateway  bool   `json:"isGateway,omitempty"` // response only
}

// AutoSettlementInheritance controls which child addresses a master-wallet rule cascades to.
type AutoSettlementInheritance struct {
	Mode            string   `json:"mode"`                      // master_only, all_children, selected_children
	ChildAddressIDs []string `json:"childAddressIds,omitempty"` // required for selected_children
}

// AutoSettlementRuleRequest is a full rule: CreateRule, AddressCreateRule,
// AddressUpdateRule and CopyRules. Source and Destination are required.
// Inheritance is ignored on child-address rules.
type AutoSettlementRuleRequest struct {
	Type                 string                     `json:"type,omitempty"` // withdraw, swap, gateway, earn; always set it
	Name                 string                     `json:"name,omitempty"`
	IsActive             *bool                      `json:"isActive,omitempty"` // default true
	Order                string                     `json:"order,omitempty"`    // FASTEST, CHEAPEST, RECOMMENDED (default), NO_SLIPPAGE
	SlippageTolerance    string                     `json:"slippageTolerance,omitempty"`
	IsGateway            bool                       `json:"isGateway,omitempty"`
	IsReward             bool                       `json:"isReward,omitempty"`
	RewardProvider       string                     `json:"rewardProvider,omitempty"` // aave, compound
	RewardType           string                     `json:"rewardType,omitempty"`     // defi
	ProductKey           string                     `json:"productKey,omitempty"`
	UseTransactionAmount bool                       `json:"useTransactionAmount,omitempty"`
	DeductionPercentage  string                     `json:"deductionPercentage,omitempty"`
	Inheritance          *AutoSettlementInheritance `json:"inheritance,omitempty"`
	Source               AutoSettlementSource       `json:"source"`
	Destination          AutoSettlementDestination  `json:"destination"`
}

// AutoSettlementRuleUpdate is a partial master-wallet rule update: only set
// fields are sent. Source, Destination and Inheritance replace the whole object.
type AutoSettlementRuleUpdate struct {
	Type                 string                     `json:"type,omitempty"`
	Name                 string                     `json:"name,omitempty"`
	IsActive             *bool                      `json:"isActive,omitempty"`
	Order                string                     `json:"order,omitempty"`
	SlippageTolerance    string                     `json:"slippageTolerance,omitempty"`
	IsGateway            *bool                      `json:"isGateway,omitempty"`
	IsReward             *bool                      `json:"isReward,omitempty"`
	RewardProvider       string                     `json:"rewardProvider,omitempty"`
	RewardType           string                     `json:"rewardType,omitempty"`
	ProductKey           string                     `json:"productKey,omitempty"`
	UseTransactionAmount *bool                      `json:"useTransactionAmount,omitempty"`
	DeductionPercentage  string                     `json:"deductionPercentage,omitempty"`
	Inheritance          *AutoSettlementInheritance `json:"inheritance,omitempty"`
	Source               *AutoSettlementSource      `json:"source,omitempty"`
	Destination          *AutoSettlementDestination `json:"destination,omitempty"`
}

// AutoSettlementCopyRequest copies rules (by value) to other master wallets.
type AutoSettlementCopyRequest struct {
	Rules           []AutoSettlementRuleRequest `json:"rules"`
	TargetWalletIDs []string                    `json:"targetWalletIds"`
}

// AutoSettlementCopyResult reports what CopyRules did. It returns 200 even
// when some targets fail, so check Summary and Results.
type AutoSettlementCopyResult struct {
	Summary AutoSettlementCopySummary        `json:"summary"`
	Results []AutoSettlementCopyWalletResult `json:"results"`
}

// AutoSettlementCopySummary counts copy outcomes.
type AutoSettlementCopySummary struct {
	WalletsTargeted int `json:"walletsTargeted"`
	RulesRequested  int `json:"rulesRequested"`
	Applied         int `json:"applied"`
	Skipped         int `json:"skipped"`
	Failed          int `json:"failed"`
}

// AutoSettlementCopyWalletResult is the outcome for one target wallet.
type AutoSettlementCopyWalletResult struct {
	WalletID   string                   `json:"walletId"`
	WalletName string                   `json:"walletName"`
	Chain      string                   `json:"chain"`
	Applied    []AutoSettlementCopyRule `json:"applied"`
	Skipped    []AutoSettlementCopyRule `json:"skipped"`
	Failed     string                   `json:"failed"` // set only when the whole wallet failed
}

// AutoSettlementCopyRule names a copied or skipped rule.
type AutoSettlementCopyRule struct {
	Name   string `json:"name"`
	Reason string `json:"reason"` // why it was skipped
}

func autoSettlementPath(walletID string) string {
	return "/v1/wallets/" + seg(walletID) + "/auto-settlements"
}

func addressAutoSettlementPath(walletID, addressID string) string {
	return "/v1/wallets/" + seg(walletID) + "/addresses/" + seg(addressID) + "/auto-settlements"
}

type autoSettlementToggle struct {
	IsActive bool `json:"isActive"`
}

// ---- master wallet endpoints ----

// Get returns the wallet's auto-settlement settings and rules.
func (s *AutoSettlementsService) Get(ctx context.Context, walletID string) (*AutoSettlement, error) {
	return call[AutoSettlement](ctx, s.c, http.MethodGet, autoSettlementPath(walletID), nil, nil)
}

// Update turns auto-settlement on or off for the wallet.
func (s *AutoSettlementsService) Update(ctx context.Context, walletID string, isActive bool) (*AutoSettlement, error) {
	return call[AutoSettlement](ctx, s.c, http.MethodPatch, autoSettlementPath(walletID), nil, autoSettlementToggle{isActive})
}

// ListRules lists the wallet's rules.
func (s *AutoSettlementsService) ListRules(ctx context.Context, walletID string) ([]AutoSettlementRule, error) {
	return getSlice[AutoSettlementRule](ctx, s.c, autoSettlementPath(walletID)+"/rules", nil)
}

// GetRule fetches one of the wallet's rules.
func (s *AutoSettlementsService) GetRule(ctx context.Context, walletID, ruleID string) (*AutoSettlementRule, error) {
	return call[AutoSettlementRule](ctx, s.c, http.MethodGet, autoSettlementPath(walletID)+"/rules/"+seg(ruleID), nil, nil)
}

// CreateRule adds a rule to the wallet.
func (s *AutoSettlementsService) CreateRule(ctx context.Context, walletID string, r AutoSettlementRuleRequest) (*AutoSettlementRule, error) {
	return call[AutoSettlementRule](ctx, s.c, http.MethodPost, autoSettlementPath(walletID)+"/rules", nil, r)
}

// UpdateRule partially updates one of the wallet's rules.
func (s *AutoSettlementsService) UpdateRule(ctx context.Context, walletID, ruleID string, r AutoSettlementRuleUpdate) (*AutoSettlementRule, error) {
	return call[AutoSettlementRule](ctx, s.c, http.MethodPatch, autoSettlementPath(walletID)+"/rules/"+seg(ruleID), nil, r)
}

// DeleteRule deletes one of the wallet's rules and returns it.
func (s *AutoSettlementsService) DeleteRule(ctx context.Context, walletID, ruleID string) (*AutoSettlementRule, error) {
	return call[AutoSettlementRule](ctx, s.c, http.MethodDelete, autoSettlementPath(walletID)+"/rules/"+seg(ruleID), nil, nil)
}

// CopyRules copies rules onto other master wallets (and switches auto-settlement on there).
func (s *AutoSettlementsService) CopyRules(ctx context.Context, walletID string, r AutoSettlementCopyRequest) (*AutoSettlementCopyResult, error) {
	return call[AutoSettlementCopyResult](ctx, s.c, http.MethodPost, autoSettlementPath(walletID)+"/rules/copy", nil, r)
}

// ---- child address endpoints ----

// AddressGet returns a child address's auto-settlement settings and rules.
func (s *AutoSettlementsService) AddressGet(ctx context.Context, walletID, addressID string) (*AutoSettlement, error) {
	return call[AutoSettlement](ctx, s.c, http.MethodGet, addressAutoSettlementPath(walletID, addressID), nil, nil)
}

// AddressUpdate turns auto-settlement on or off for a child address.
func (s *AutoSettlementsService) AddressUpdate(ctx context.Context, walletID, addressID string, isActive bool) (*AutoSettlement, error) {
	return call[AutoSettlement](ctx, s.c, http.MethodPatch, addressAutoSettlementPath(walletID, addressID), nil, autoSettlementToggle{isActive})
}

// AddressListRules lists a child address's rules.
func (s *AutoSettlementsService) AddressListRules(ctx context.Context, walletID, addressID string) ([]AutoSettlementRule, error) {
	return getSlice[AutoSettlementRule](ctx, s.c, addressAutoSettlementPath(walletID, addressID)+"/rules", nil)
}

// AddressGetRule fetches one of a child address's rules.
func (s *AutoSettlementsService) AddressGetRule(ctx context.Context, walletID, addressID, ruleID string) (*AutoSettlementRule, error) {
	return call[AutoSettlementRule](ctx, s.c, http.MethodGet, addressAutoSettlementPath(walletID, addressID)+"/rules/"+seg(ruleID), nil, nil)
}

// AddressCreateRule adds a rule to a child address.
func (s *AutoSettlementsService) AddressCreateRule(ctx context.Context, walletID, addressID string, r AutoSettlementRuleRequest) (*AutoSettlementRule, error) {
	return call[AutoSettlementRule](ctx, s.c, http.MethodPost, addressAutoSettlementPath(walletID, addressID)+"/rules", nil, r)
}

// AddressUpdateRule replaces a child address's rule. Send the complete rule
// (see AutoSettlementRule.Request): omitted fields reset to defaults.
func (s *AutoSettlementsService) AddressUpdateRule(ctx context.Context, walletID, addressID, ruleID string, r AutoSettlementRuleRequest) (*AutoSettlementRule, error) {
	return call[AutoSettlementRule](ctx, s.c, http.MethodPatch, addressAutoSettlementPath(walletID, addressID)+"/rules/"+seg(ruleID), nil, r)
}

// AddressDeleteRule deletes one of a child address's rules and returns it.
func (s *AutoSettlementsService) AddressDeleteRule(ctx context.Context, walletID, addressID, ruleID string) (*AutoSettlementRule, error) {
	return call[AutoSettlementRule](ctx, s.c, http.MethodDelete, addressAutoSettlementPath(walletID, addressID)+"/rules/"+seg(ruleID), nil, nil)
}
