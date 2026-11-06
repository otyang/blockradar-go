package blockradar

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaymentLinksRoutes(t *testing.T) {
	c, routes := recordRoutes(t, "")
	ctx := context.Background()
	p := c.PaymentLinks

	calls := []func() error{
		func() error { _, err := p.Create(ctx, PaymentLinkCreateRequest{Name: "n"}); return err },
		func() error { _, err := p.List(ctx); return err },
		func() error { _, err := p.Get(ctx, "P"); return err },
		func() error { _, err := p.Update(ctx, "P", PaymentLinkUpdateRequest{}); return err },
		func() error { _, err := p.Transactions(ctx, "P"); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, routes(), []string{
		"POST /v1/payment_links",
		"GET /v1/payment_links",
		"GET /v1/payment_links/P",
		"PATCH /v1/payment_links/P",
		"GET /v1/payment_links/P/transactions",
	})
}

const paymentLinkExample = `{"active": true, "amount": null, "configurations": {}, "createdAt": "2025-04-30T14:00:48.522Z", "createdChannel": "dashboard", "currency": "USD", "description": "Payment link description", "id": "7dfa86ff-fd3a-45e1-8ecf-dc8a06ef65cd", "imageUrl": null, "inactiveMessage": null, "metadata": null, "name": "Payment Link", "network": "testnet", "redirectUrl": "https://www.blockradar.co/", "slug": "payment-link-10012", "successMessage": null, "type": "payment", "updatedAt": "2025-04-30T14:00:48.522Z", "url": "https://pay.blockradar.co/payment-link-10012"}`

// multipartServer parses each multipart request and keeps the last form.
func multipartServer(t *testing.T, payload string) (*Client, *struct {
	method, path string
	fields       map[string]string
	fileName     string
	file         string
}) {
	t.Helper()
	got := &struct {
		method, path string
		fields       map[string]string
		fileName     string
		file         string
	}{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path = r.Method, r.URL.Path
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}
		got.fields = map[string]string{}
		for k, v := range r.MultipartForm.Value {
			got.fields[k] = v[0]
		}
		if fh := r.MultipartForm.File["file"]; len(fh) > 0 {
			f, _ := fh[0].Open()
			b, _ := io.ReadAll(f)
			got.fileName, got.file = fh[0].Filename, string(b)
		}
		_, _ = io.WriteString(w, payload)
	}))
	t.Cleanup(srv.Close)
	return NewClient("k", WithBaseURL(srv.URL)), got
}

func TestPaymentLinkCreateMultipart(t *testing.T) {
	c, got := multipartServer(t, `{"data": `+paymentLinkExample+`, "message": "Payment Link created successfully", "statusCode": 200}`)

	link, err := c.PaymentLinks.Create(context.Background(), PaymentLinkCreateRequest{
		Name:         "Payment link name",
		Amount:       "100",
		Slug:         "payment-link",
		Metadata:     map[string]any{"orderId": "o1", "qty": 2},
		PaymentLimit: "1",
		File:         strings.NewReader("PNGDATA"),
		FileName:     "logo.png",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "POST" || got.path != "/v1/payment_links" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	want := map[string]string{"name": "Payment link name", "amount": "100", "slug": "payment-link",
		"metadata": `{"orderId":"o1","qty":2}`, "paymentLimit": "1"}
	if mustJSON(got.fields) != mustJSON(want) {
		t.Fatalf("fields = %v, want %v", got.fields, want)
	}
	if got.fileName != "logo.png" || got.file != "PNGDATA" {
		t.Fatalf("file = %q %q", got.fileName, got.file)
	}
	if link.Slug != "payment-link-10012" || link.URL == "" || link.Amount != "" || !link.Active || link.CreatedAt.IsZero() {
		t.Fatalf("decoded %+v", link)
	}
}

func TestPaymentLinkUpdateMultipart(t *testing.T) {
	c, got := multipartServer(t, `{"data": `+paymentLinkExample+`, "message": "Payment Link updated successfully", "statusCode": 200}`)

	_, err := c.PaymentLinks.Update(context.Background(), "pl1", PaymentLinkUpdateRequest{
		PaymentLinkCreateRequest: PaymentLinkCreateRequest{SuccessMessage: "Hey"},
		Active:                   Bool(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "PATCH" || got.path != "/v1/payment_links/pl1" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	if mustJSON(got.fields) != `{"active":"false","successMessage":"Hey"}` || got.file != "" {
		t.Fatalf("fields = %v file = %q", got.fields, got.file)
	}
}

func TestPaymentLinkCreateNeedsName(t *testing.T) {
	if _, err := NewClient("k").PaymentLinks.Create(context.Background(), PaymentLinkCreateRequest{}); err == nil {
		t.Fatal("want error")
	}
}

func TestPaymentLinkListAndTransactions(t *testing.T) {
	c, got := server(t, 200, `{"data": [{"active": false, "amount": "2", "configurations": {"paymentlimit": 1}, "createdAt": "2025-02-07T20:03:39.974Z", "createdChannel": "dashboard", "currency": "USD", "description": "Payment link description", "id": "84d4e096-0b71-4d0c-ab4f-8a692c67314c", "imageUrl": null, "inactiveMessage": null, "metadata": {}, "name": "Payment Link", "network": "testnet", "redirectUrl": null, "slug": "payment-link-12", "successMessage": null, "type": "payment", "updatedAt": "2025-03-04T14:16:44.830Z"}], "message": "Payment Links fetched successfully", "meta": {"currentPage": 1, "itemCount": 8, "itemsPerPage": 10, "totalItems": 8, "totalPages": 1}, "statusCode": 200}`)
	list, err := c.PaymentLinks.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "GET" || got.path != "/v1/payment_links" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	if len(list.Data) != 1 || list.Data[0].Amount != "2" || list.Meta.TotalItems != 8 || list.Data[0].Configurations["paymentlimit"] != float64(1) {
		t.Fatalf("list = %+v", list)
	}

	c, _ = server(t, 200, `{"data": [{"amlScreening": {"message": "Address is not sanctioned", "provider": "ofac, fbi, tether, circle", "status": "success"}, "amount": "0.1", "amountPaid": "0.1", "amountUSD": "0.1", "asset": {"address": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "configurations": null, "createdAt": "2024-05-14T17:53:33.680Z", "currency": "USD", "decimals": 18, "id": "1890db58-f624-4972-a72b-e25387a81520", "isActive": true, "isNative": false, "name": "Tether USD", "network": "testnet", "standard": "BEP20", "symbol": "USDT", "updatedAt": "2025-04-17T04:50:19.964Z"}, "assetSwept": true, "assetSweptAmount": "0.1", "assetSweptAt": "2025-09-04T12:24:42.450Z", "blockHash": "0x8a05", "blockNumber": 64195701, "blockchain": {"configurations": null, "createdAt": "2024-05-14T17:53:33.106Z", "derivationPath": "m/44'/60'/0'/0", "id": "b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa", "isActive": true, "isEvmCompatible": true, "isL2": false, "name": "BNB smart chain", "slug": "bnb-smart-chain", "symbol": "bnb", "tokenStandard": "BEP20", "updatedAt": "2024-11-26T20:04:13.941Z"}, "chainId": 97, "confirmations": 10, "confirmed": true, "createdAt": "2025-09-04T18:24:02.249Z", "currency": "USD", "fee": null, "gasFee": "0.000051603", "gasPrice": "1000000000", "gasUsed": "51603", "hash": "0xff17", "id": "a3c355d1-d172-4cc6-aefa-19b7ee4d5d04", "metadata": null, "network": "testnet", "note": null, "reason": "Funds swept successfully", "recipientAddress": "0xF00B", "reference": "pay-Uz3aRUMh3zug30emoULk", "senderAddress": "0x2455", "status": "SUCCESS", "type": "DEPOSIT", "updatedAt": "2025-09-04T18:24:42.451Z"}], "message": "Payment link transactions fetched successfully", "meta": {"currentPage": 1, "itemCount": 10, "itemsPerPage": 10, "totalItems": 10, "totalPages": 1}, "statusCode": 200}`)
	txs, err := c.PaymentLinks.Transactions(context.Background(), "pl1")
	if err != nil {
		t.Fatal(err)
	}
	if len(txs.Data) != 1 || txs.Data[0].Amount != "0.1" || txs.Data[0].ChainID != "97" || txs.Data[0].Asset.Symbol != "USDT" || txs.Meta.ItemCount != 10 {
		t.Fatalf("txs = %+v", txs)
	}
}
