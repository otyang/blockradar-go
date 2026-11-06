package blockradar

import (
	"context"
	"testing"
)

func TestCustomersRoutes(t *testing.T) {
	c, routes := recordRoutes(t, "")
	ctx := context.Background()
	cs := c.Customers
	calls := []func() error{
		func() error { _, err := cs.Create(ctx, CreateCustomerRequest{}); return err },
		func() error { _, err := cs.List(ctx); return err },
		func() error { _, err := cs.Get(ctx, "C"); return err },
		func() error { _, err := cs.Update(ctx, "C", UpdateCustomerRequest{}); return err },
	}
	for i, f := range calls {
		if err := f(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, routes(), []string{
		"POST /v1/customers",
		"GET /v1/customers",
		"GET /v1/customers/C",
		"PATCH /v1/customers/C",
	})
}

const customerJSON = `{"id":"18d2f159-e0cd-4a45-b735-e9dfe0f7520a","name":"Ada Lovelace",
	"email":"ada@example.com","phone":null,"type":"individual","status":"ACTIVE","network":"testnet",
	"additionalData":{"ref":"u-1"},"createdAt":"2026-07-22T12:00:00.000Z","updatedAt":"2026-07-22T12:00:00.000Z"}`

func TestCustomerCreateBodyAndDecode(t *testing.T) {
	c, got := server(t, 201, `{"statusCode":201,"message":"Customer created","data":`+customerJSON+`}`)
	cust, err := c.Customers.Create(context.Background(), CreateCustomerRequest{
		Name: "Ada Lovelace", Email: "ada@example.com", Type: CustomerTypeIndividual,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "Ada Lovelace", "email": "ada@example.com", "type": "individual"}
	if b1, b2 := mustJSON(got.body), mustJSON(want); b1 != b2 {
		t.Fatalf("body = %s, want %s", b1, b2)
	}
	if cust.ID != "18d2f159-e0cd-4a45-b735-e9dfe0f7520a" || cust.Type != CustomerTypeIndividual ||
		cust.Status != CustomerStatusActive || cust.Phone != "" || cust.AdditionalData["ref"] != "u-1" || cust.CreatedAt.IsZero() {
		t.Fatalf("decoded %+v", cust)
	}
}

func TestCustomerUpdateSendsOnlySetFields(t *testing.T) {
	c, got := server(t, 200, `{"statusCode":200,"message":"ok","data":`+customerJSON+`}`)
	if _, err := c.Customers.Update(context.Background(), "c1", UpdateCustomerRequest{Status: CustomerStatusInactive}); err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"status":"INACTIVE"}` {
		t.Fatalf("body = %s", b)
	}
}

func TestCustomerListDecodesArray(t *testing.T) {
	c, got := server(t, 200, `{"statusCode":200,"message":"ok","data":[`+customerJSON+`]}`)
	list, err := c.Customers.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "GET" || got.query != "" || len(list) != 1 || list[0].Email != "ada@example.com" {
		t.Fatalf("got %s ?%s %+v", got.method, got.query, list)
	}
}
