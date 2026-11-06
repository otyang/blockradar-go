package blockradar

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Error codes returned in APIError.Details.Code.
const (
	CodeCustomerOnboardingRequired      = "CUSTOMER_ONBOARDING_REQUIRED"
	CodeCustomerOnboardingPending       = "CUSTOMER_ONBOARDING_PENDING"
	CodeCustomerOnboardingRejected      = "CUSTOMER_ONBOARDING_REJECTED"
	CodeCustomerProviderUnavailable     = "CUSTOMER_PROVIDER_UNAVAILABLE"
	CodeCustomerIdentityUnsupported     = "CUSTOMER_IDENTITY_UNSUPPORTED"
	CodeOfframpRouteUnsupported         = "OFFRAMP_ROUTE_UNSUPPORTED"
	CodeOfframpBusinessNotEligible      = "OFFRAMP_BUSINESS_NOT_ELIGIBLE"
	CodeWithdrawalSessionExpired        = "FIAT_WITHDRAWAL_SESSION_EXPIRED"
	CodeWithdrawalSessionMismatch       = "FIAT_WITHDRAWAL_SESSION_MISMATCH"
	CodeWithdrawalSessionUnavailable    = "FIAT_WITHDRAWAL_SESSION_UNAVAILABLE"
	CodeWithdrawalSessionPricingChanged = "FIAT_WITHDRAWAL_SESSION_PRICING_CHANGED"
	CodeDepositSessionExpired           = "FIAT_DEPOSIT_SESSION_EXPIRED"
	CodeDepositSessionMismatch          = "FIAT_DEPOSIT_SESSION_MISMATCH"
	CodeDepositSessionUnavailable       = "FIAT_DEPOSIT_SESSION_UNAVAILABLE"
)

// APIError is returned for any non-2xx response (or a 2xx with "status": false).
// Use errors.As to inspect it.
type APIError struct {
	HTTPStatus int
	Messages   []string
	Details    *ErrorDetails // set on customer/fiat errors (400, 403, 409)
	RequestID  string
	Body       []byte // raw response body
}

func (e *APIError) Error() string {
	msg := strings.Join(e.Messages, "; ")
	if msg == "" {
		msg = strings.TrimSpace(string(e.Body))
	}
	if e.Code() != "" {
		return fmt.Sprintf("blockradar: HTTP %d %s: %s", e.HTTPStatus, e.Code(), msg)
	}
	return fmt.Sprintf("blockradar: HTTP %d: %s", e.HTTPStatus, msg)
}

// Code returns Details.Code, or "" if the error has no details.
func (e *APIError) Code() string {
	if e.Details == nil {
		return ""
	}
	return e.Details.Code
}

// ErrorDetails explains how to recover from a customer/fiat error.
type ErrorDetails struct {
	Code       string          `json:"code"`
	CustomerID string          `json:"customerId,omitempty"`
	Provider   *ErrorProvider  `json:"provider,omitempty"`
	Onboarding *OnboardingInfo `json:"onboarding,omitempty"`
}

type ErrorProvider struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type OnboardingInfo struct {
	CaseID         string            `json:"caseId"`
	Status         string            `json:"status"`
	ProviderStatus string            `json:"providerStatus"`
	Action         *OnboardingAction `json:"action,omitempty"`
}

// OnboardingAction is the next step: Type is "create_case", "resume_case",
// "wait" or "contact_support". Method/Path/Body describe the call to make.
type OnboardingAction struct {
	Type   string          `json:"type"`
	Method string          `json:"method,omitempty"`
	Path   string          `json:"path,omitempty"`
	Body   json.RawMessage `json:"body,omitempty"`
}

// messages decodes "message" whether it is a string or an array of strings.
type messages []string

func (m *messages) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*m = messages{s}
		return nil
	}
	var a []string
	if err := json.Unmarshal(b, &a); err != nil {
		return nil // unknown shape: keep the raw body on APIError instead
	}
	*m = a
	return nil
}

func parseAPIError(status int, raw []byte) *APIError {
	e := &APIError{HTTPStatus: status, Body: raw}
	var body struct {
		Message   messages      `json:"message"`
		Details   *ErrorDetails `json:"details"`
		RequestID string        `json:"requestId"`
	}
	if json.Unmarshal(raw, &body) == nil {
		e.Messages = body.Message
		e.Details = body.Details
		e.RequestID = body.RequestID
	}
	return e
}
