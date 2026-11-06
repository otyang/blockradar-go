// Package blockradar is a Go client for the Blockradar API.
//
// Withdraw Fiat and Virtual Accounts use the v2 endpoints; every other
// area uses v1 (Blockradar has no v2 for them).
//
//	c := blockradar.NewClient(os.Getenv("BLOCKRADAR_API_KEY"))
//	wallet, err := c.Wallets.Get(ctx, walletID, false)
package blockradar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// maxRetryWait caps how long one 429 retry waits, whatever Retry-After says.
const maxRetryWait = 60 * time.Second

// DefaultBaseURL is the Blockradar API host. Paths include the /v1 or /v2 prefix.
const DefaultBaseURL = "https://api.blockradar.co"

// Client talks to the Blockradar API. It is safe for concurrent use.
// The API key decides the environment (testnet or mainnet).
type Client struct {
	apiKey     string
	baseURL    string
	http       *http.Client
	maxRetries int // retries on HTTP 429

	// v2
	WithdrawFiat    *WithdrawFiatService
	VirtualAccounts *VirtualAccountsService

	// v1
	Wallets         *WalletsService
	Addresses       *AddressesService
	Assets          *AssetsService
	Transactions    *TransactionsService
	Misc            *MiscService
	AML             *AMLService
	Withdraw        *WithdrawService
	Signing         *SigningService
	Swap            *SwapService
	Contracts       *ContractsService
	AssetRecovery   *AssetRecoveryService
	Customers       *CustomersService
	Onboarding      *OnboardingService
	DepositFiat     *DepositFiatService
	PaymentLinks    *PaymentLinksService
	AutoSettlements *AutoSettlementsService
	Gateway         *GatewayService
	LiquidityPool   *LiquidityPoolService
	Rewards         *RewardsService
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API host (useful for tests).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithHTTPClient sets the underlying *http.Client (timeouts, transport, tracing).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// WithMaxRetries sets how many times a rate-limited (HTTP 429) request is
// retried, with backoff. Default 3; 0 disables retries.
func WithMaxRetries(n int) Option {
	return func(c *Client) { c.maxRetries = n }
}

// NewClient returns a client authenticated with a wallet API key.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		http:       &http.Client{Timeout: 30 * time.Second},
		maxRetries: 3,
	}
	for _, o := range opts {
		o(c)
	}
	c.WithdrawFiat = &WithdrawFiatService{c: c}
	c.VirtualAccounts = &VirtualAccountsService{c: c}
	c.Wallets = &WalletsService{c: c}
	c.Addresses = &AddressesService{c: c}
	c.Assets = &AssetsService{c: c}
	c.Transactions = &TransactionsService{c: c}
	c.Misc = &MiscService{c: c}
	c.AML = &AMLService{c: c}
	c.Withdraw = &WithdrawService{c: c}
	c.Signing = &SigningService{c: c}
	c.Swap = &SwapService{c: c}
	c.Contracts = &ContractsService{c: c}
	c.AssetRecovery = &AssetRecoveryService{c: c}
	c.Customers = &CustomersService{c: c}
	c.Onboarding = &OnboardingService{c: c}
	c.DepositFiat = &DepositFiatService{c: c}
	c.PaymentLinks = &PaymentLinksService{c: c}
	c.AutoSettlements = &AutoSettlementsService{c: c}
	c.Gateway = &GatewayService{c: c}
	c.LiquidityPool = &LiquidityPoolService{c: c}
	c.Rewards = &RewardsService{c: c}
	return c
}

// envelope is the response wrapper. v2 sends {"status","message","data"};
// v1 sends {"statusCode","message","data"}. Only "data" is kept.
type envelope[T any] struct {
	Data T `json:"data"`
}

// call sends a JSON request (body may be nil) and returns the unwrapped "data" field.
func call[T any](ctx context.Context, c *Client, method, path string, q url.Values, body any) (*T, error) {
	var env envelope[T]
	if err := c.do(ctx, method, path, q, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// getSlice GETs path and returns the unwrapped "data" array.
func getSlice[T any](ctx context.Context, c *Client, path string, q url.Values) ([]T, error) {
	out, err := call[[]T](ctx, c, http.MethodGet, path, q, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// callMultipart sends a multipart/form-data request and returns the unwrapped "data" field.
func callMultipart[T any](ctx context.Context, c *Client, method, path string, fields map[string]string, files []File) (*T, error) {
	ct, body, err := multipartBody(fields, files)
	if err != nil {
		return nil, err
	}
	var env envelope[T]
	if err := c.send(ctx, method, path, nil, ct, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// do sends a request with an optional JSON body and decodes the response into out.
func (c *Client) do(ctx context.Context, method, path string, q url.Values, body, out any) error {
	if body == nil {
		return c.send(ctx, method, path, q, "", nil, out)
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("blockradar: encode request: %w", err)
	}
	return c.send(ctx, method, path, q, "application/json", b, out)
}

// send performs the HTTP call and decodes a 2xx JSON body into out.
// Non-2xx responses, and 2xx responses with "status": false, become *APIError.
// HTTP 429 (rate limited, nothing processed) is retried up to c.maxRetries times.
func (c *Client) send(ctx context.Context, method, path string, q url.Values, contentType string, body []byte, out any) error {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	var status int
	var raw []byte
	for attempt := 0; ; attempt++ {
		var err error
		var wait time.Duration
		status, raw, wait, err = c.roundTrip(ctx, method, u, contentType, body)
		if err != nil {
			return fmt.Errorf("blockradar: %s %s: %w", method, path, err)
		}
		if status != http.StatusTooManyRequests || attempt >= c.maxRetries {
			break
		}
		if wait < 0 { // no Retry-After header
			wait = time.Second << attempt // 1s, 2s, 4s, ...
		}
		if wait > maxRetryWait {
			wait = maxRetryWait
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}

	if status < 200 || status > 299 {
		return parseAPIError(status, raw)
	}

	if len(bytes.TrimSpace(raw)) == 0 { // e.g. 204 No Content
		return nil
	}
	// Status is kept raw so a non-bool "status" (e.g. "success") is not a decode error.
	var head struct {
		Status json.RawMessage `json:"status"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return fmt.Errorf("blockradar: decode response (HTTP %d): %w", status, err)
	}
	if string(head.Status) == "false" {
		return parseAPIError(status, raw)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("blockradar: decode response: %w", err)
	}
	return nil
}

// roundTrip makes one HTTP request and returns the status, body and any Retry-After delay.
func (c *Client) roundTrip(ctx context.Context, method, u, contentType string, body []byte) (int, []byte, time.Duration, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return 0, nil, 0, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, 0, fmt.Errorf("read response: %w", err)
	}
	wait := time.Duration(-1)
	if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s >= 0 {
		wait = time.Duration(s) * time.Second
	}
	return resp.StatusCode, raw, wait, nil
}

// File is a file part of a multipart upload.
type File struct {
	Field    string    // form field name, e.g. "front"
	Filename string    // e.g. "passport.jpg"
	Content  io.Reader // file bytes
}

func multipartBody(fields map[string]string, files []File) (string, []byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return "", nil, err
		}
	}
	for _, f := range files {
		if f.Content == nil {
			continue
		}
		part, err := w.CreateFormFile(f.Field, f.Filename)
		if err != nil {
			return "", nil, err
		}
		if _, err := io.Copy(part, f.Content); err != nil {
			return "", nil, fmt.Errorf("blockradar: read %s: %w", f.Field, err)
		}
	}
	if err := w.Close(); err != nil {
		return "", nil, err
	}
	return w.FormDataContentType(), buf.Bytes(), nil
}

// seg escapes a path segment (wallet, address and account IDs).
func seg(s string) string { return url.PathEscape(s) }

// query helpers: only set non-zero values.
func setStr(q url.Values, k, v string) {
	if v != "" {
		q.Set(k, v)
	}
}

func setInt(q url.Values, k string, v int) {
	if v != 0 {
		q.Set(k, strconv.Itoa(v))
	}
}

func setTime(q url.Values, k string, v time.Time) {
	if !v.IsZero() {
		q.Set(k, v.UTC().Format(time.RFC3339))
	}
}

func setBool(q url.Values, k string, v *bool) {
	if v != nil {
		q.Set(k, strconv.FormatBool(*v))
	}
}

// Bool returns a pointer to b, for optional boolean fields and filters.
func Bool(b bool) *bool { return &b }
