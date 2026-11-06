package blockradar

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

type captured struct {
	method, path, query, apiKey string
	body                        map[string]any
}

// server answers every request with status/payload and records the last request.
func server(t *testing.T, status int, payload string) (*Client, *captured) {
	t.Helper()
	got := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path, got.query = r.Method, r.URL.EscapedPath(), r.URL.RawQuery
		got.apiKey = r.Header.Get("x-api-key")
		b, _ := io.ReadAll(r.Body)
		got.body = nil
		if len(b) > 0 {
			_ = json.Unmarshal(b, &got.body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, payload)
	}))
	t.Cleanup(srv.Close)
	return NewClient("test-key", WithBaseURL(srv.URL)), got
}

func TestRatesSendsQueryAndDecodes(t *testing.T) {
	c, got := server(t, 200, `{"status":true,"message":"ok","data":{
		"rate":"1520.5","amount":"152050","unlocked":true,
		"sessionId":"11111111-1111-1111-1111-111111111111",
		"sessionExpiresAt":"2026-11-06T12:00:00Z","provider":"p1"}}`)

	rate, err := c.WithdrawFiat.Rates(context.Background(), "w1", WithdrawFiatRatesParams{
		AssetID: "a1", Currency: "NGN", Amount: "100", AmountSide: AmountSideSource,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "GET" || got.path != "/v2/wallets/w1/withdraw/fiat/rates" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	if got.apiKey != "test-key" {
		t.Fatalf("x-api-key = %q", got.apiKey)
	}
	if got.query != "amount=100&amountSide=source&assetId=a1&currency=NGN" {
		t.Fatalf("query = %q", got.query)
	}
	if rate.Rate != "1520.5" || rate.SessionID == "" || rate.SessionExpiresAt.IsZero() {
		t.Fatalf("decoded %+v", rate)
	}
}

func TestExecuteFlattensQuoteRequest(t *testing.T) {
	c, got := server(t, 200, `{"status":true,"message":"ok","data":{"id":"tx1","reference":"r1","recipientAddress":"x","amount":"100","status":"PENDING","processingStatus":"PENDING","type":"WITHDRAW","createdAt":"2026-11-06T12:00:00Z","updatedAt":"2026-11-06T12:00:00Z","fee":null}}`)

	tx, err := c.WithdrawFiat.AddressExecute(context.Background(), "w1", "addr1", WithdrawFiatExecuteRequest{
		WithdrawFiatQuoteRequest: WithdrawFiatQuoteRequest{SessionID: "s1", PaymentMethod: "bank", PaymentMethodData: map[string]any{"accountNumber": "0123"}},
		Reference:                "order-42",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v2/wallets/w1/addresses/addr1/withdraw/fiat/execute" {
		t.Fatalf("path = %s", got.path)
	}
	want := map[string]any{"sessionId": "s1", "paymentMethod": "bank", "reference": "order-42",
		"paymentMethodData": map[string]any{"accountNumber": "0123"}}
	if b1, b2 := mustJSON(got.body), mustJSON(want); b1 != b2 {
		t.Fatalf("body = %s, want %s", b1, b2)
	}
	if tx.ID != "tx1" || tx.Fee != "" {
		t.Fatalf("decoded %+v", tx)
	}
}

func TestQuoteNeedsSessionOrRouting(t *testing.T) {
	c := NewClient("k")
	_, err := c.WithdrawFiat.Quote(context.Background(), "w1", WithdrawFiatQuoteRequest{PaymentMethod: "bank"})
	if !errors.Is(err, ErrMissingFlow) {
		t.Fatalf("err = %v", err)
	}
}

func TestErrorDetails(t *testing.T) {
	c, _ := server(t, 409, `{"statusCode":409,"message":["Customer onboarding required"],
		"details":{"code":"CUSTOMER_ONBOARDING_REQUIRED","customerId":"c1",
		"onboarding":{"caseId":null,"status":null,"providerStatus":null,
		"action":{"type":"create_case","method":"POST","path":"/v1/customers/c1/cases","body":{"provider":"p","type":"offramp"}}}},
		"requestId":"req-1"}`)

	_, err := c.WithdrawFiat.Rates(context.Background(), "w1", WithdrawFiatRatesParams{AssetID: "a", Currency: "NGN", Amount: "1"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v", err)
	}
	if apiErr.HTTPStatus != 409 || apiErr.Code() != CodeCustomerOnboardingRequired || apiErr.RequestID != "req-1" {
		t.Fatalf("got %+v", apiErr)
	}
	if apiErr.Details.Onboarding.Action.Type != "create_case" {
		t.Fatalf("action = %+v", apiErr.Details.Onboarding.Action)
	}
	if !strings.Contains(err.Error(), "Customer onboarding required") {
		t.Fatalf("Error() = %s", err)
	}
}

func TestStringMessageError(t *testing.T) {
	c, _ := server(t, 401, `{"statusCode":401,"message":"Invalid API key"}`)
	_, err := c.VirtualAccounts.Get(context.Background(), "w1", "va1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Messages[0] != "Invalid API key" {
		t.Fatalf("err = %v", err)
	}
}

func TestStatusFalseIsError(t *testing.T) {
	c, _ := server(t, 200, `{"status":false,"message":"Something failed","data":null}`)
	_, err := c.VirtualAccounts.Get(context.Background(), "w1", "va1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v", err)
	}
}

func TestListPagination(t *testing.T) {
	c, got := server(t, 200, `{"status":true,"message":"ok",
		"data":[{"id":"va1","accountNumber":"0123456789","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}],
		"meta":{"total":11,"page":1,"limit":10,"pageCount":2,"hasNextPage":true},
		"links":{"first":"f","previous":null,"next":"n","last":"l"}}`)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	page, err := c.VirtualAccounts.List(context.Background(), "w1", ListVirtualAccountsParams{
		VirtualAccountListParams: VirtualAccountListParams{StartDate: start, EndDate: start.AddDate(0, 1, 0), Limit: 10},
		IsActive:                 Bool(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "endDate=2026-02-01T00%3A00%3A00Z&isActive=true&limit=10&startDate=2026-01-01T00%3A00%3A00Z" {
		t.Fatalf("query = %s", got.query)
	}
	if len(page.Data) != 1 || !page.Meta.HasNextPage || page.Links.Previous != "" {
		t.Fatalf("page = %+v", page)
	}
}

// Every v2 endpoint from docs.blockradar.co, as "METHOD path".
var specRoutes = []string{
	"GET /v2/wallets/W/withdraw/fiat/assets",
	"GET /v2/wallets/W/withdraw/fiat/currencies",
	"GET /v2/wallets/W/withdraw/fiat/rates",
	"GET /v2/wallets/W/withdraw/fiat/payment-methods",
	"GET /v2/wallets/W/withdraw/fiat/payment-method-requirements",
	"POST /v2/wallets/W/withdraw/fiat/payment-method/resolve",
	"POST /v2/wallets/W/withdraw/fiat/quote",
	"POST /v2/wallets/W/withdraw/fiat/execute",
	"POST /v2/wallets/W/addresses/A/withdraw/fiat/quote",
	"POST /v2/wallets/W/addresses/A/withdraw/fiat/execute",
	"GET /v2/wallets/W/virtual-accounts/discovery",
	"GET /v2/wallets/W/virtual-accounts/requirements",
	"POST /v2/wallets/W/virtual-accounts",
	"GET /v2/wallets/W/virtual-accounts",
	"GET /v2/wallets/W/virtual-accounts/V",
	"PATCH /v2/wallets/W/virtual-accounts/V",
	"POST /v2/wallets/W/virtual-accounts/V/regenerate",
	"GET /v2/wallets/W/virtual-accounts/V/destination/discovery",
	"PATCH /v2/wallets/W/virtual-accounts/V/destination",
	"POST /v2/wallets/W/addresses/A/virtual-accounts",
	"GET /v2/wallets/W/addresses/A/virtual-accounts",
	"GET /v2/wallets/W/addresses/A/virtual-accounts/V",
	"PATCH /v2/wallets/W/addresses/A/virtual-accounts/V",
	"POST /v2/wallets/W/addresses/A/virtual-accounts/V/regenerate",
	"GET /v2/wallets/W/addresses/A/virtual-accounts/V/transactions",
}

func TestAllRoutesMatchSpec(t *testing.T) {
	var hits []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.Method+" "+r.URL.Path)
		_, _ = io.WriteString(w, `{"status":true,"message":"ok","data":null}`)
	}))
	defer srv.Close()
	c := NewClient("k", WithBaseURL(srv.URL))
	ctx := context.Background()
	q := WithdrawFiatQuoteRequest{SessionID: "s", PaymentMethod: "m"}
	e := WithdrawFiatExecuteRequest{WithdrawFiatQuoteRequest: q}
	va := CreateVirtualAccountRequest{Provider: "p", Currency: "NGN"}
	rg := RegenerateVirtualAccountRequest{CreateVirtualAccountRequest: va, Reason: "r"}
	wf, vs := c.WithdrawFiat, c.VirtualAccounts

	calls := []func() error{
		func() error { _, err := wf.SupportedAssets(ctx, "W", ""); return err },
		func() error { _, err := wf.Currencies(ctx, "W", ""); return err },
		func() error {
			_, err := wf.Rates(ctx, "W", WithdrawFiatRatesParams{AssetID: "a", Currency: "NGN", Amount: "1"})
			return err
		},
		func() error {
			_, err := wf.PaymentMethods(ctx, "W", WithdrawFiatFlowParams{SessionID: "s"})
			return err
		},
		func() error {
			_, err := wf.PaymentMethodRequirements(ctx, "W", "m", WithdrawFiatFlowParams{SessionID: "s"})
			return err
		},
		func() error {
			_, err := wf.ResolvePaymentMethod(ctx, "W", ResolvePaymentMethodRequest{SessionID: "s", PaymentMethod: "m"})
			return err
		},
		func() error { _, err := wf.Quote(ctx, "W", q); return err },
		func() error { _, err := wf.Execute(ctx, "W", e); return err },
		func() error { _, err := wf.AddressQuote(ctx, "W", "A", q); return err },
		func() error { _, err := wf.AddressExecute(ctx, "W", "A", e); return err },
		func() error { _, err := vs.Discovery(ctx, "W", ""); return err },
		func() error {
			_, err := vs.Requirements(ctx, "W", VirtualAccountRequirementsParams{Currency: "NGN"})
			return err
		},
		func() error { _, err := vs.Create(ctx, "W", va); return err },
		func() error { _, err := vs.List(ctx, "W", ListVirtualAccountsParams{}); return err },
		func() error { _, err := vs.Get(ctx, "W", "V"); return err },
		func() error { _, err := vs.Update(ctx, "W", "V", UpdateVirtualAccountRequest{}); return err },
		func() error { _, err := vs.Regenerate(ctx, "W", "V", rg); return err },
		func() error { _, err := vs.DestinationDiscovery(ctx, "W", "V"); return err },
		func() error {
			_, err := vs.ChangeDestination(ctx, "W", "V", ChangeVirtualAccountDestinationRequest{})
			return err
		},
		func() error { _, err := vs.AddressCreate(ctx, "W", "A", va); return err },
		func() error { _, err := vs.AddressList(ctx, "W", "A", ListVirtualAccountsParams{}); return err },
		func() error { _, err := vs.AddressGet(ctx, "W", "A", "V"); return err },
		func() error {
			_, err := vs.AddressUpdate(ctx, "W", "A", "V", UpdateVirtualAccountRequest{})
			return err
		},
		func() error { _, err := vs.AddressRegenerate(ctx, "W", "A", "V", rg); return err },
		func() error {
			_, err := vs.AddressTransactions(ctx, "W", "A", "V", VirtualAccountListParams{})
			return err
		},
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}

	want := append([]string(nil), specRoutes...)
	sort.Strings(want)
	sort.Strings(hits)
	if strings.Join(hits, "\n") != strings.Join(want, "\n") {
		t.Fatalf("routes differ\n got:\n%s\nwant:\n%s", strings.Join(hits, "\n"), strings.Join(want, "\n"))
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func jsonUnmarshal(s string, v any) error { return json.Unmarshal([]byte(s), v) }

// recordRoutes returns a client whose every call succeeds with payload
// (default: empty data) and a func listing the "METHOD /path" calls made, sorted.
func recordRoutes(t *testing.T, payload string) (*Client, func() []string) {
	t.Helper()
	if payload == "" {
		payload = `{"statusCode":200,"message":"ok","data":null}`
	}
	var hits []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.Method+" "+r.URL.EscapedPath())
		_, _ = io.WriteString(w, payload)
	}))
	t.Cleanup(srv.Close)
	return NewClient("k", WithBaseURL(srv.URL)), func() []string {
		out := append([]string(nil), hits...)
		sort.Strings(out)
		return out
	}
}

// assertRoutes fails unless got and want hold the same routes.
func assertRoutes(t *testing.T, got, want []string) {
	t.Helper()
	w := append([]string(nil), want...)
	sort.Strings(w)
	if strings.Join(got, "\n") != strings.Join(w, "\n") {
		t.Fatalf("routes differ\n got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(w, "\n"))
	}
}

func TestRetriesRateLimit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := io.ReadAll(r.Body)
		if r.Method == http.MethodPatch && string(b) != `{"walletId":"w","assetId":"a"}` {
			t.Errorf("body on attempt %d = %s", calls, b)
		}
		if calls < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"statusCode":429,"message":"Too many requests"}`)
			return
		}
		_, _ = io.WriteString(w, `{"status":true,"message":"ok","data":{"id":"va1"}}`)
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	va, err := c.VirtualAccounts.ChangeDestination(context.Background(), "w", "v", ChangeVirtualAccountDestinationRequest{WalletID: "w", AssetID: "a"})
	if err != nil || va.ID != "va1" || calls != 3 {
		t.Fatalf("va=%v err=%v calls=%d", va, err, calls)
	}

	calls = -100 // always 429
	c = NewClient("k", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err = c.VirtualAccounts.Get(context.Background(), "w", "v")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 429 {
		t.Fatalf("err = %v", err)
	}
}

func TestFiatAssetLiveShape(t *testing.T) {
	// Shape returned by the live API (wallet asset), not the flat one in the docs.
	c, _ := server(t, 200, `{"status":true,"message":"ok","data":[{"id":"wa1","isActive":true,
		"asset":{"id":"a1","name":"USD Coin","symbol":"USDC","decimals":6,"currency":"USD","network":"testnet"}}]}`)
	got, err := c.WithdrawFiat.SupportedAssets(context.Background(), "w", "")
	if err != nil || len(got) != 1 || got[0].ID != "wa1" || got[0].Info().Symbol != "USDC" {
		t.Fatalf("got %+v err %v", got, err)
	}
	flat := FiatAsset{ID: "a2", Symbol: "USDT"}
	if flat.Info().Symbol != "USDT" {
		t.Fatal("flat shape")
	}
}
