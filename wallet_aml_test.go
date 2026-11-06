package blockradar

import (
	"context"
	"testing"
)

func TestAMLRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	if _, err := c.AML.Lookup(context.Background(), "X", "ethereum"); err != nil {
		t.Fatal(err)
	}
	assertRoutes(t, hits(), []string{"GET /v1/aml/lookup"})
}

func TestAMLLookupDecodes(t *testing.T) {
	c, got := server(t, 200, `{"data":{"isBlacklisted":false},"message":"Anti Money Laundering lookup","statusCode":200}`)
	r, err := c.AML.Lookup(context.Background(), "0x451dEFC27B45808078e875556AF06bCFdC697BA4", "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "address=0x451dEFC27B45808078e875556AF06bCFdC697BA4&blockchain=ethereum" {
		t.Fatalf("query = %q", got.query)
	}
	if r.IsBlacklisted {
		t.Fatalf("lookup = %+v", r)
	}
}
