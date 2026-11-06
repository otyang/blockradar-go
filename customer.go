package blockradar

import (
	"context"
	"net/http"
	"time"
)

// CustomersService manages the business's customers (/v1/customers), the
// subjects of onboarding and fiat flows.
type CustomersService struct{ c *Client }

// CustomerType says whether a customer is a person or a company.
type CustomerType string

const (
	CustomerTypeIndividual CustomerType = "individual"
	CustomerTypeBusiness   CustomerType = "business"
)

// CustomerStatus says whether a customer is active.
type CustomerStatus string

const (
	CustomerStatusActive   CustomerStatus = "ACTIVE"
	CustomerStatusInactive CustomerStatus = "INACTIVE"
)

// Customer is an end user or company owned by the authenticated business.
type Customer struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Email          string         `json:"email"`
	Phone          string         `json:"phone"`
	Type           CustomerType   `json:"type"` // may be empty
	Status         CustomerStatus `json:"status"`
	Network        string         `json:"network"` // "mainnet" or "testnet"
	AdditionalData map[string]any `json:"additionalData"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// CreateCustomerRequest: Name, Email and Type are required.
type CreateCustomerRequest struct {
	Name  string       `json:"name"`
	Email string       `json:"email"`
	Type  CustomerType `json:"type"`
	Phone string       `json:"phone,omitempty"`
}

// UpdateCustomerRequest changes only the fields that are set.
type UpdateCustomerRequest struct {
	Name   string         `json:"name,omitempty"`
	Email  string         `json:"email,omitempty"`
	Type   CustomerType   `json:"type,omitempty"`
	Phone  string         `json:"phone,omitempty"`
	Status CustomerStatus `json:"status,omitempty"`
}

func customerPath(customerID string) string { return "/v1/customers/" + seg(customerID) }

// Create creates a customer.
func (s *CustomersService) Create(ctx context.Context, r CreateCustomerRequest) (*Customer, error) {
	return call[Customer](ctx, s.c, http.MethodPost, "/v1/customers", nil, r)
}

// List lists the business's customers, newest first.
func (s *CustomersService) List(ctx context.Context) ([]Customer, error) {
	return getSlice[Customer](ctx, s.c, "/v1/customers", nil)
}

// Get returns one customer.
func (s *CustomersService) Get(ctx context.Context, customerID string) (*Customer, error) {
	return call[Customer](ctx, s.c, http.MethodGet, customerPath(customerID), nil, nil)
}

// Update changes a customer's details or status. Customers cannot be deleted.
func (s *CustomersService) Update(ctx context.Context, customerID string, r UpdateCustomerRequest) (*Customer, error) {
	return call[Customer](ctx, s.c, http.MethodPatch, customerPath(customerID), nil, r)
}
