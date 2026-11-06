package blockradar

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// OnboardingService covers /v1/customers/{customerId}/onboarding-requests:
// verifying a customer with a fiat provider before onramp/offramp.
//
// Typical flow:
//
//	Create -> fill Requirements.Schema -> Submit -> follow hosted steps/actions -> Get or Refresh until approved
//
// Approval is Case.Status == OnboardingApproved, not a 2xx response.
// Fiat calls that need onboarding fail with an *APIError whose Details.Onboarding says what to do.
type OnboardingService struct{ c *Client }

// ---- types ----

// OnboardingCaseStatus is the state of an onboarding case.
type OnboardingCaseStatus string

const (
	OnboardingDraft                 OnboardingCaseStatus = "draft"
	OnboardingCollectingInformation OnboardingCaseStatus = "collecting_information"
	OnboardingReadyForSubmission    OnboardingCaseStatus = "ready_for_submission"
	OnboardingSubmitted             OnboardingCaseStatus = "submitted"
	OnboardingActionRequired        OnboardingCaseStatus = "action_required"
	OnboardingPendingAdminReview    OnboardingCaseStatus = "pending_admin_review"
	OnboardingInReview              OnboardingCaseStatus = "in_review"
	OnboardingApproved              OnboardingCaseStatus = "approved"
	OnboardingRejected              OnboardingCaseStatus = "rejected"
	OnboardingFailed                OnboardingCaseStatus = "failed"
	OnboardingExpired               OnboardingCaseStatus = "expired"
)

// Values of OnboardingCase.AvailableActions.
const (
	OnboardingEditAndResubmit      = "edit_and_resubmit"      // Reopen is allowed
	OnboardingRefreshProviderSteps = "refresh_provider_steps" // Refresh can renew hosted steps
)

// OnboardingCase is one onboarding request for a customer with a provider.
type OnboardingCase struct {
	ID                string               `json:"id"`
	BusinessID        string               `json:"businessId"`
	CustomerID        string               `json:"customerId"`
	ProviderID        string               `json:"providerId"`
	SubjectType       string               `json:"subjectType"` // "customer"
	Status            OnboardingCaseStatus `json:"status"`
	Mode              string               `json:"mode"`    // api_only, hosted_only, hybrid, manual
	Feature           string               `json:"feature"` // offramp, onramp_payment_order, onramp_virtual_account
	ProviderReference string               `json:"providerReference"`
	RouteIntent       map[string]any       `json:"routeIntent"`
	Data              map[string]any       `json:"data"`
	// HostedSessions are provider-hosted steps; each carries a sequenceIndex
	// for SubmitHostedSession.
	HostedSessions []map[string]any `json:"hostedSessions"`
	Reason         string           `json:"reason"`
	Metadata       map[string]any   `json:"metadata"`
	// Actions request more information; each carries an id, type, status and schema.
	Actions          []map[string]any `json:"actions"`
	AvailableActions []string         `json:"availableActions"` // OnboardingEditAndResubmit, OnboardingRefreshProviderSteps
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

// OnboardingRequirements describes what to collect next; build the Submit data from Schema.
type OnboardingRequirements struct {
	Mode        string               `json:"mode"`
	SubjectType string               `json:"subjectType"`
	Feature     string               `json:"feature"`
	Status      OnboardingCaseStatus `json:"status"`
	Schema      JSONSchema           `json:"schema"`
}

// OnboardingRequest is a case with its customer; Requirements is nil on
// operations that return only the case.
type OnboardingRequest struct {
	CustomerID   string                  `json:"customerId"`
	Case         OnboardingCase          `json:"case"`
	Requirements *OnboardingRequirements `json:"requirements,omitempty"`
}

// OnboardingSubmitResult: if Initiated is false, complete Requirements and submit again.
type OnboardingSubmitResult struct {
	OnboardingRequest
	Initiated bool `json:"initiated"`
}

// OnboardingRequestList is a page of cases.
type OnboardingRequestList struct {
	Items []OnboardingCase `json:"items"`
	Meta  ListMeta         `json:"meta"`
	Links PaginationLinks  `json:"links"`
}

// OnboardingConfirmationRequirements holds the confirmation form schema.
type OnboardingConfirmationRequirements struct {
	CustomerID string     `json:"customerId"`
	CaseID     string     `json:"caseId"`
	Schema     JSONSchema `json:"schema"`
}

// OnboardingDocumentViewURLs holds temporary preview links.
type OnboardingDocumentViewURLs struct {
	CaseID    string                      `json:"caseId"`
	Documents []OnboardingDocumentViewURL `json:"documents"`
}

type OnboardingDocumentViewURL struct {
	StorageKey string    `json:"storageKey"`
	ViewURL    string    `json:"viewUrl"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// OnboardingPreparedUploads lists upload targets, one per requested file field.
type OnboardingPreparedUploads struct {
	CaseID  string                     `json:"caseId"`
	Uploads []OnboardingPreparedUpload `json:"uploads"`
}

type OnboardingPreparedUpload struct {
	Field    string                 `json:"field"`
	Document OnboardingUploadTarget `json:"document"`
}

// OnboardingUploadTarget says how to upload a file. For Mode "proxy", call
// UploadDocument with ProxyUploadID; for "direct", send it to URL with Method and Headers.
type OnboardingUploadTarget struct {
	Mode              string            `json:"mode"` // "direct" or "proxy"
	ProxyUploadID     string            `json:"proxyUploadId"`
	URL               string            `json:"url"`
	Method            string            `json:"method"`
	Headers           map[string]string `json:"headers"`
	ProviderReference string            `json:"providerReference"`
	DocumentType      string            `json:"documentType"`
	ExpiresAt         *time.Time        `json:"expiresAt"`
	Sides             []string          `json:"sides"`  // "front", "back"
	Accept            []string          `json:"accept"` // allowed content types
}

// OnboardingUploadedDocument is the result of UploadDocument.
type OnboardingUploadedDocument struct {
	CaseID        string `json:"caseId"`
	ProxyUploadID string `json:"proxyUploadId"`
	Field         string `json:"field"`
	Document      struct {
		ProviderReference string   `json:"providerReference"`
		DocumentType      string   `json:"documentType"`
		UploadedSides     []string `json:"uploadedSides"`
	} `json:"document"`
}

// ---- params / requests ----

// CreateOnboardingRequest: omit Provider to use provider routing.
type CreateOnboardingRequest struct {
	Provider    string                 `json:"provider,omitempty"` // slug or UUID
	Type        string                 `json:"type,omitempty"`     // "onramp" or "offramp"
	RouteIntent *OnboardingRouteIntent `json:"routeIntent,omitempty"`
	CallbackURL string                 `json:"callbackUrl,omitempty"`
}

// OnboardingRouteIntent: SourceCurrencyCode and TargetCurrencyCode are required.
type OnboardingRouteIntent struct {
	SourceCurrencyCode string `json:"sourceCurrencyCode"` // e.g. "NGN"
	TargetCurrencyCode string `json:"targetCurrencyCode"` // e.g. "USDC"
	Chain              string `json:"chain,omitempty"`    // e.g. "ethereum"
	Network            string `json:"network,omitempty"`  // "mainnet" or "testnet"
}

// OnboardingListParams filters and paginates List. Zero values are omitted.
type OnboardingListParams struct {
	Page       int // 1-based
	Limit      int
	Status     OnboardingCaseStatus
	ProviderID string
	Feature    string // offramp, onramp_payment_order, onramp_virtual_account
}

func (p OnboardingListParams) values() url.Values {
	q := url.Values{}
	setInt(q, "page", p.Page)
	setInt(q, "limit", p.Limit)
	setStr(q, "status", string(p.Status))
	setStr(q, "providerId", p.ProviderID)
	setStr(q, "feature", p.Feature)
	return q
}

// OnboardingDataRequest carries a completed form built from a schema.
type OnboardingDataRequest struct {
	Data map[string]any `json:"data"`
}

// OnboardingDraftRequest carries partially completed fields.
type OnboardingDraftRequest struct {
	Data map[string]any `json:"data,omitempty"`
}

// OnboardingRefreshRequest: Force regenerates supported hosted steps.
type OnboardingRefreshRequest struct {
	Force bool `json:"force,omitempty"`
}

// OnboardingHostedSessionRequest: set Submitted to true once the customer finishes the step.
type OnboardingHostedSessionRequest struct {
	Submitted bool `json:"submitted"`
}

type OnboardingDocumentViewURLRequest struct {
	Documents []OnboardingDocumentRef `json:"documents"`
}

type OnboardingDocumentRef struct {
	StorageKey string `json:"storageKey"`
}

// OnboardingPrepareUploadsRequest: Mode is "proxy" (default) or "direct".
type OnboardingPrepareUploadsRequest struct {
	Mode  string                 `json:"mode,omitempty"`
	Files []OnboardingUploadFile `json:"files"`
}

// OnboardingUploadFile: Field is the property path from the requirements schema.
type OnboardingUploadFile struct {
	Field            string                 `json:"field"`
	FileName         string                 `json:"fileName,omitempty"`
	ContentType      string                 `json:"contentType,omitempty"`
	DocumentType     string                 `json:"documentType,omitempty"`
	Sides            []string               `json:"sides,omitempty"` // "front", "back"
	DoubleSided      bool                   `json:"doubleSided,omitempty"`
	Parts            []OnboardingUploadPart `json:"parts,omitempty"`
	MaxFileSizeBytes int                    `json:"maxFileSizeBytes,omitempty"`
}

type OnboardingUploadPart struct {
	Side        string `json:"side"` // "front" or "back"
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
}

// ---- paths ----

func onboardingPath(customerID string) string {
	return "/v1/customers/" + seg(customerID) + "/onboarding-requests"
}

func onboardingCasePath(customerID, caseID string) string {
	return onboardingPath(customerID) + "/" + seg(caseID)
}

// ---- endpoints ----

// Create starts an onboarding request; save Case.ID for later calls.
func (s *OnboardingService) Create(ctx context.Context, customerID string, r CreateOnboardingRequest) (*OnboardingRequest, error) {
	return call[OnboardingRequest](ctx, s.c, http.MethodPost, onboardingPath(customerID), nil, r)
}

// List lists a customer's onboarding requests.
func (s *OnboardingService) List(ctx context.Context, customerID string, p OnboardingListParams) (*OnboardingRequestList, error) {
	return call[OnboardingRequestList](ctx, s.c, http.MethodGet, onboardingPath(customerID), p.values(), nil)
}

// Get returns the saved case and its current requirements.
func (s *OnboardingService) Get(ctx context.Context, customerID, caseID string) (*OnboardingRequest, error) {
	return call[OnboardingRequest](ctx, s.c, http.MethodGet, onboardingCasePath(customerID, caseID), nil, nil)
}

// SaveDraft saves partial data without submitting it for review.
func (s *OnboardingService) SaveDraft(ctx context.Context, customerID, caseID string, r OnboardingDraftRequest) (*OnboardingRequest, error) {
	return call[OnboardingRequest](ctx, s.c, http.MethodPatch, onboardingCasePath(customerID, caseID), nil, r)
}

// Submit sends the completed requirements form.
func (s *OnboardingService) Submit(ctx context.Context, customerID, caseID string, r OnboardingDataRequest) (*OnboardingSubmitResult, error) {
	return call[OnboardingSubmitResult](ctx, s.c, http.MethodPost, onboardingCasePath(customerID, caseID)+"/submit", nil, r)
}

// Refresh asks the provider for a status update.
func (s *OnboardingService) Refresh(ctx context.Context, customerID, caseID string, r OnboardingRefreshRequest) (*OnboardingRequest, error) {
	return call[OnboardingRequest](ctx, s.c, http.MethodPost, onboardingCasePath(customerID, caseID)+"/refresh", nil, r)
}

// Reopen returns a case to collecting_information; only when AvailableActions has OnboardingEditAndResubmit.
func (s *OnboardingService) Reopen(ctx context.Context, customerID, caseID string) (*OnboardingRequest, error) {
	return call[OnboardingRequest](ctx, s.c, http.MethodPost, onboardingCasePath(customerID, caseID)+"/reopen", nil, nil)
}

// SubmitHostedSession records whether the customer completed a hosted step.
// sequenceIndex is the step's returned sequenceIndex, not its array position.
func (s *OnboardingService) SubmitHostedSession(ctx context.Context, customerID, caseID string, sequenceIndex int, r OnboardingHostedSessionRequest) (*OnboardingRequest, error) {
	path := onboardingCasePath(customerID, caseID) + "/hosted-sessions/" + strconv.Itoa(sequenceIndex) + "/submission"
	return call[OnboardingRequest](ctx, s.c, http.MethodPatch, path, nil, r)
}

// SubmitAction answers an action that asks for more information.
func (s *OnboardingService) SubmitAction(ctx context.Context, customerID, caseID, actionID string, r OnboardingDataRequest) (*OnboardingRequest, error) {
	path := onboardingCasePath(customerID, caseID) + "/actions/" + seg(actionID) + "/submissions"
	return call[OnboardingRequest](ctx, s.c, http.MethodPost, path, nil, r)
}

// ConfirmationRequirements returns the confirmation form schema, when the workflow needs one.
func (s *OnboardingService) ConfirmationRequirements(ctx context.Context, customerID, caseID string) (*OnboardingConfirmationRequirements, error) {
	return call[OnboardingConfirmationRequirements](ctx, s.c, http.MethodGet, onboardingCasePath(customerID, caseID)+"/confirmation-requirements", nil, nil)
}

// SubmitConfirmation sends the completed confirmation form.
func (s *OnboardingService) SubmitConfirmation(ctx context.Context, customerID, caseID string, r OnboardingDataRequest) (*OnboardingRequest, error) {
	return call[OnboardingRequest](ctx, s.c, http.MethodPost, onboardingCasePath(customerID, caseID)+"/confirmation", nil, r)
}

// PrepareUploads returns upload targets for the file fields in the requirements schema.
func (s *OnboardingService) PrepareUploads(ctx context.Context, customerID, caseID string, r OnboardingPrepareUploadsRequest) (*OnboardingPreparedUploads, error) {
	return call[OnboardingPreparedUploads](ctx, s.c, http.MethodPost, onboardingCasePath(customerID, caseID)+"/uploads/prepare", nil, r)
}

// UploadDocument sends a proxy upload. Field names are set for you; leave back
// zero when only the front is requested.
func (s *OnboardingService) UploadDocument(ctx context.Context, customerID, caseID, uploadID string, front, back File) (*OnboardingUploadedDocument, error) {
	front.Field, back.Field = "front", "back"
	path := onboardingCasePath(customerID, caseID) + "/uploads/" + seg(uploadID)
	return callMultipart[OnboardingUploadedDocument](ctx, s.c, http.MethodPost, path, nil, []File{front, back})
}

// DocumentViewURLs creates temporary preview links for stored documents.
func (s *OnboardingService) DocumentViewURLs(ctx context.Context, customerID, caseID string, r OnboardingDocumentViewURLRequest) (*OnboardingDocumentViewURLs, error) {
	return call[OnboardingDocumentViewURLs](ctx, s.c, http.MethodPost, onboardingCasePath(customerID, caseID)+"/documents/view-url", nil, r)
}
