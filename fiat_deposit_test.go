package blockradar

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDepositFiatRoutes(t *testing.T) {
	c, routes := recordRoutes(t, "")
	ctx := context.Background()
	d := c.DepositFiat
	q := DepositFiatQuoteRequest{SessionID: "s"}
	o := PaymentOrderCreateRequest{SessionID: "s"}

	calls := []func() error{
		func() error { _, err := d.SupportedAssets(ctx, "W", ""); return err },
		func() error { _, err := d.Currencies(ctx, "W", "a", ""); return err },
		func() error {
			_, err := d.Rails(ctx, "W", DepositFiatRailsParams{AssetID: "a", Currency: "NGN", Amount: "1"})
			return err
		},
		func() error {
			_, err := d.ResolvePaymentOrderRequirements(ctx, "W", PaymentOrderResolveRequest{SessionID: "s"})
			return err
		},
		func() error { _, err := d.Quote(ctx, "W", q); return err },
		func() error { _, err := d.CreateOrder(ctx, "W", o); return err },
		func() error { _, err := d.ListOrders(ctx, "W", PaymentOrderListParams{}); return err },
		func() error { _, err := d.GetOrder(ctx, "W", "O"); return err },
		func() error { _, err := d.CancelOrder(ctx, "W", "O"); return err },
		func() error { _, err := d.RefreshOrder(ctx, "W", "O"); return err },
		func() error { _, err := d.AddressQuote(ctx, "W", "A", q); return err },
		func() error { _, err := d.AddressCreateOrder(ctx, "W", "A", o); return err },
		func() error { _, err := d.AddressListOrders(ctx, "W", "A", PaymentOrderListParams{}); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, routes(), []string{
		"GET /v1/wallets/W/deposit/fiat/assets",
		"GET /v1/wallets/W/deposit/fiat/currencies",
		"GET /v1/wallets/W/deposit/fiat/rails",
		"POST /v1/wallets/W/deposit/fiat/payment-order-requirements/resolve",
		"POST /v1/wallets/W/deposit/fiat/quote",
		"POST /v1/wallets/W/deposit/fiat/orders",
		"GET /v1/wallets/W/deposit/fiat/orders",
		"GET /v1/wallets/W/deposit/fiat/orders/O",
		"POST /v1/wallets/W/deposit/fiat/orders/O/cancel",
		"POST /v1/wallets/W/deposit/fiat/orders/O/refresh",
		"POST /v1/wallets/W/addresses/A/deposit/fiat/quote",
		"POST /v1/wallets/W/addresses/A/deposit/fiat/orders",
		"GET /v1/wallets/W/addresses/A/deposit/fiat/orders",
	})
}

const paymentOrderExample = `{"id": "80ea45ab-38b9-4506-b213-52da8bc7e825", "type": "deposit", "rail": "one_time_virtual_account", "reference": "dep_20260719_001", "providerReference": "provider_order_839201", "transactionId": null, "status": "pending", "currency": "NGN", "amount": "100.00", "amountFiat": "155000.00", "feeFiat": "1500.00", "amountFiatPayable": "156500.00", "amountFiatReceived": null, "amountSettled": null, "expiresAt": "2026-07-19T13:00:00.000Z", "paymentInstructions": {"type": "one_time_virtual_account", "bank": {"name": "Example Bank", "code": "999", "accountName": "Blockradar / Acme Ltd", "accountNumber": "0123456789"}, "paymentLink": null, "qrCode": null, "expiresAt": "2026-07-19T13:00:00.000Z", "metadata": null}, "metadata": {"checkoutId": "checkout_123"}, "provider": {"id": "1128dd40-2c94-4c16-a6bf-fbbc6afee52c", "slug": "example-provider", "name": "Example Provider"}, "wallet": {"id": "4465468a-3c36-4536-918a-91d689e18a74", "address": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a"}, "address": null, "asset": {"id": "ae455f23-3824-4125-baab-d158315cbcbd", "symbol": "USDC", "name": "USD Coin"}, "customer": {"id": "18d2f159-e0cd-4a45-b735-e9dfe0f7520a"}, "createdAt": "2026-07-19T12:30:00.000Z", "updatedAt": "2026-07-19T12:30:00.000Z"}`

func TestDepositFiatCreateOrder(t *testing.T) {
	c, got := server(t, 201, `{"message": "Payment order created successfully", "data": `+paymentOrderExample+`, "status": true}`)

	o, err := c.DepositFiat.AddressCreateOrder(context.Background(), "w1", "a1", PaymentOrderCreateRequest{
		SessionID:           "s1",
		Reference:           "dep_20260719_001",
		ValidityTimeMinutes: "30",
		Customer:            &PaymentOrderCustomer{Name: "Ada Okafor", Email: "ada@example.com"},
		AdditionalData:      map[string]any{"accountIdentifier": "9000000010"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "POST" || got.path != "/v1/wallets/w1/addresses/a1/deposit/fiat/orders" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	want := map[string]any{"sessionId": "s1", "reference": "dep_20260719_001", "validityTimeMinutes": 30,
		"customer":       map[string]any{"name": "Ada Okafor", "email": "ada@example.com"},
		"additionalData": map[string]any{"accountIdentifier": "9000000010"}}
	if b1, b2 := mustJSON(got.body), mustJSON(want); b1 != b2 {
		t.Fatalf("body = %s, want %s", b1, b2)
	}
	if o.Rail != PaymentOrderRailOneTimeVirtualAccount || o.PaymentInstructions.Bank.AccountNumber != "0123456789" ||
		o.PaymentInstructions.QRCode != nil || o.Address != nil || o.Customer.ID == "" ||
		o.TransactionID != "" || o.ExpiresAt == nil || o.Provider.Slug != "example-provider" ||
		o.Metadata["checkoutId"] != "checkout_123" {
		t.Fatalf("decoded %+v", o)
	}
}

func TestDepositFiatQuote(t *testing.T) {
	c, got := server(t, 200, `{"message": "Fiat deposit quote retrieved", "data": {"amount": "100.00", "amountFiat": "155000.00", "feeFiat": "1500.00", "amountFiatPayable": "156500.00", "rate": "1550.00", "unlocked": true, "sessionId": "13a36c1c-f4a6-4e49-a582-2ecded2559bc", "sessionExpiresAt": "2026-09-15T15:20:00.000Z", "additionalDataRequirements": {"schema": {"$schema": "http://json-schema.org/draft-07/schema#", "type": "object", "title": "Bank account details", "properties": {"institutionIdentifier": {"type": "string", "title": "Institution"}}, "required": ["institutionIdentifier"], "additionalProperties": false}, "data": null}}, "status": true}`)

	qt, err := c.DepositFiat.Quote(context.Background(), "w1", DepositFiatQuoteRequest{
		AssetID: "a1", Currency: "NGN", Amount: "100.00", Rail: PaymentOrderRailQRCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"assetId": "a1", "currency": "NGN", "amount": "100.00", "rail": "qr_code"}
	if b1, b2 := mustJSON(got.body), mustJSON(want); b1 != b2 {
		t.Fatalf("body = %s, want %s", b1, b2)
	}
	if qt.AmountFiatPayable != "156500.00" || qt.SessionID == "" || qt.SessionExpiresAt.IsZero() ||
		len(qt.AdditionalDataRequirements.Schema) == 0 || qt.AdditionalDataRequirements.Data != nil {
		t.Fatalf("decoded %+v", qt)
	}
}

func TestDepositFiatNeedsSessionOrRouting(t *testing.T) {
	c := NewClient("k")
	ctx := context.Background()
	// Rail is part of the routing fields for deposits.
	_, err := c.DepositFiat.Quote(ctx, "w1", DepositFiatQuoteRequest{AssetID: "a", Currency: "NGN", Amount: "1"})
	if !errors.Is(err, ErrMissingFlow) {
		t.Fatalf("Quote err = %v", err)
	}
	_, err = c.DepositFiat.CreateOrder(ctx, "w1", PaymentOrderCreateRequest{Reference: "r"})
	if !errors.Is(err, ErrMissingFlow) {
		t.Fatalf("CreateOrder err = %v", err)
	}
	if _, err := c.DepositFiat.Rails(ctx, "w1", DepositFiatRailsParams{AssetID: "a"}); err == nil {
		t.Fatal("Rails without Currency/Amount should fail")
	}
}

func TestDepositFiatListOrders(t *testing.T) {
	// Shape observed on the live API (the docs show a flat "data" array).
	c, got := server(t, 200, `{"message": "Successful", "status": true, "data": {"items": [`+paymentOrderExample+`], "meta": {"totalItems": 1, "itemCount": 1, "itemsPerPage": 10, "totalPages": 1, "currentPage": 1}, "analytics": {"totalOrdersCount": 1, "totalPendingOrdersCount": 1, "totalPaidOrdersCount": 0, "totalProcessingOrdersCount": 0, "totalExpiredOrdersCount": 0, "totalFailedOrdersCount": 0, "totalCancelledOrdersCount": 0}}}`)

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	page, err := c.DepositFiat.ListOrders(context.Background(), "w1", PaymentOrderListParams{
		Status: "pending", Rail: PaymentOrderRailPaymentLink, Currency: "NGN", Search: "dep",
		StartDate: start, EndDate: start.AddDate(0, 1, 0), Page: 2, Limit: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/wallets/w1/deposit/fiat/orders" {
		t.Fatalf("path = %s", got.path)
	}
	if got.query != "currency=NGN&endDate=2026-08-01T00%3A00%3A00Z&limit=50&page=2&rail=payment_link&search=dep&startDate=2026-07-01T00%3A00%3A00Z&status=pending" {
		t.Fatalf("query = %s", got.query)
	}
	if len(page.Items) != 1 || page.Meta.TotalItems != 1 || page.Analytics.TotalPendingOrdersCount != 1 {
		t.Fatalf("page = %+v", page)
	}
}

func TestDepositFiatDiscovery(t *testing.T) {
	c, got := server(t, 200, `{"message": "Successful", "data": {"rails": [{"rail": "one_time_virtual_account", "sourceCurrencyCode": "NGN", "targetCurrencyCode": "USDC", "minAmount": "5000.00", "maxAmount": "5000000.00", "provider": {"slug": "example-provider", "name": "Example Provider"}}]}, "status": true}`)

	rails, err := c.DepositFiat.Rails(context.Background(), "w1", DepositFiatRailsParams{AssetID: "a1", Currency: "NGN", Amount: "100", Provider: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "amount=100&assetId=a1&currency=NGN&provider=p" {
		t.Fatalf("query = %s", got.query)
	}
	if len(rails) != 1 || rails[0].MaxAmount != "5000000.00" || rails[0].Provider.Name != "Example Provider" {
		t.Fatalf("rails = %+v", rails)
	}

	c, got = server(t, 200, `{"message": "Currencies fetched successfully", "data": [{"code": "NGN", "name": "Nigerian Naira", "shortName": "Naira", "decimals": 2, "symbol": "₦"}], "status": true}`)
	cur, err := c.DepositFiat.Currencies(context.Background(), "w1", "a1", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "assetId=a1" || len(cur) != 1 || cur[0].Symbol != "₦" {
		t.Fatalf("query = %s, currencies = %+v", got.query, cur)
	}
}
