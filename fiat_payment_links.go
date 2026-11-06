package blockradar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// PaymentLinksService covers /v1/payment_links: hosted pages that collect
// stablecoin payments.
type PaymentLinksService struct{ c *Client }

// PaymentLink is a hosted payment page.
type PaymentLink struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Slug            string         `json:"slug"`
	URL             string         `json:"url"` // e.g. https://pay.blockradar.co/<slug>; not sent on every endpoint
	Amount          Amount         `json:"amount"`
	Currency        string         `json:"currency"`
	Active          bool           `json:"active"`
	Type            string         `json:"type"`
	Network         string         `json:"network"`
	ImageURL        string         `json:"imageUrl"`
	RedirectURL     string         `json:"redirectUrl"`
	SuccessMessage  string         `json:"successMessage"`
	InactiveMessage string         `json:"inactiveMessage"`
	Configurations  map[string]any `json:"configurations"` // e.g. {"paymentlimit": 1}
	Metadata        map[string]any `json:"metadata"`
	CreatedChannel  string         `json:"createdChannel"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// PaymentLinkCreateRequest creates a payment link. Only Name is required.
// It is sent as multipart/form-data.
type PaymentLinkCreateRequest struct {
	Name            string
	Description     string
	Slug            string // URL-friendly, ^[a-zA-Z0-9-]+$
	Amount          string // fixed amount; empty lets the payer choose
	RedirectURL     string // must include http:// or https://
	SuccessMessage  string
	InactiveMessage string
	Metadata        map[string]any // string or number values; sent as a JSON string
	PaymentLimit    json.Number    // max number of payments (min 1)
	File            io.Reader      // optional image or document
	FileName        string         // e.g. "logo.png"
}

// PaymentLinkUpdateRequest changes a payment link; unset fields are left as they are.
type PaymentLinkUpdateRequest struct {
	PaymentLinkCreateRequest
	Active *bool
}

func (r PaymentLinkCreateRequest) form() (map[string]string, []File, error) {
	f := map[string]string{}
	set := func(k, v string) {
		if v != "" {
			f[k] = v
		}
	}
	set("name", r.Name)
	set("description", r.Description)
	set("slug", r.Slug)
	set("amount", r.Amount)
	set("redirectUrl", r.RedirectURL)
	set("successMessage", r.SuccessMessage)
	set("inactiveMessage", r.InactiveMessage)
	set("paymentLimit", r.PaymentLimit.String())
	if r.Metadata != nil {
		b, err := json.Marshal(r.Metadata)
		if err != nil {
			return nil, nil, fmt.Errorf("blockradar: encode metadata: %w", err)
		}
		f["metadata"] = string(b)
	}
	var files []File
	if r.File != nil {
		name := r.FileName
		if name == "" {
			name = "file"
		}
		files = append(files, File{Field: "file", Filename: name, Content: r.File})
	}
	return f, files, nil
}

// Create creates a payment link.
func (s *PaymentLinksService) Create(ctx context.Context, r PaymentLinkCreateRequest) (*PaymentLink, error) {
	if r.Name == "" {
		return nil, errors.New("blockradar: payment link Name is required")
	}
	fields, files, err := r.form()
	if err != nil {
		return nil, err
	}
	return callMultipart[PaymentLink](ctx, s.c, http.MethodPost, "/v1/payment_links", fields, files)
}

// List lists payment links.
func (s *PaymentLinksService) List(ctx context.Context) (*List[PaymentLink], error) {
	var out List[PaymentLink]
	if err := s.c.do(ctx, http.MethodGet, "/v1/payment_links", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches a payment link.
func (s *PaymentLinksService) Get(ctx context.Context, id string) (*PaymentLink, error) {
	return call[PaymentLink](ctx, s.c, http.MethodGet, "/v1/payment_links/"+seg(id), nil, nil)
}

// Update changes a payment link.
func (s *PaymentLinksService) Update(ctx context.Context, id string, r PaymentLinkUpdateRequest) (*PaymentLink, error) {
	fields, files, err := r.form()
	if err != nil {
		return nil, err
	}
	if r.Active != nil {
		fields["active"] = strconv.FormatBool(*r.Active)
	}
	return callMultipart[PaymentLink](ctx, s.c, http.MethodPatch, "/v1/payment_links/"+seg(id), fields, files)
}

// Transactions lists payments received through a payment link.
func (s *PaymentLinksService) Transactions(ctx context.Context, id string) (*List[Transaction], error) {
	var out List[Transaction]
	if err := s.c.do(ctx, http.MethodGet, "/v1/payment_links/"+seg(id)+"/transactions", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
