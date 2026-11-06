package blockradar

import (
	"context"
	"testing"
)

func TestLiquidityPoolRoutes(t *testing.T) {
	c, routes := recordRoutes(t, "")
	ctx := context.Background()
	lp := c.LiquidityPool

	calls := []func() error{
		func() error {
			_, err := lp.Create(ctx, LiquidityCreateRateRequest{FromAsset: "USDT", ToAsset: "cNGN", Rate: "1545", MinAmount: "10"})
			return err
		},
		func() error { _, err := lp.Get(ctx, "R"); return err },
		func() error { _, err := lp.Update(ctx, "R", LiquidityUpdateRateRequest{Rate: "1"}); return err },
		func() error {
			_, err := lp.Deactivate(ctx, "R", LiquidityDeactivateRateRequest{Reason: "x"})
			return err
		},
		func() error { _, err := lp.Reactivate(ctx, "R"); return err },
		func() error { _, err := lp.History(ctx, "R", PageParams{}); return err },
		func() error { _, err := lp.CheckPair(ctx, "USDT", "cNGN"); return err },
		func() error {
			_, err := lp.MarketBenchmark(ctx, LiquidityBenchmarkParams{FromAsset: "USDT", ToAsset: "cNGN"})
			return err
		},
		func() error { _, err := lp.TreasuryBalances(ctx); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, routes(), []string{
		"POST /v1/rates",
		"GET /v1/rates/R",
		"PATCH /v1/rates/R",
		"PATCH /v1/rates/R/deactivate",
		"PATCH /v1/rates/R/reactivate",
		"GET /v1/rates/R/history",
		"GET /v1/rates/check-pair",
		"GET /v1/rates/market-benchmark",
		"GET /v1/rates/treasury-balances",
	})
}

func TestLiquidityCreateBodyAndDecode(t *testing.T) {
	c, got := server(t, 201, `{"data": {"createdAt": "2026-02-19T07:50:17.042Z", "fromAsset": "BNB", "id": "d69078ef-2467-40f4-bb00-63394efe32c0", "isActive": true, "maxAmount": null, "minAmount": "0.000005", "network": "testnet", "rate": "1", "status": "active", "toAsset": "USDC", "version": 1}, "message": "Rate created successfully", "statusCode": 201}`)
	r, err := c.LiquidityPool.Create(context.Background(), LiquidityCreateRateRequest{
		FromAsset: "BNB", ToAsset: "USDC", Rate: "1", MinAmount: "0.000005",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "POST" || got.path != "/v1/rates" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	if b := mustJSON(got.body); b != `{"fromAsset":"BNB","minAmount":"0.000005","rate":"1","toAsset":"USDC"}` {
		t.Fatalf("body = %s", b)
	}
	if r.ID != "d69078ef-2467-40f4-bb00-63394efe32c0" || r.Rate != "1" || r.MaxAmount != "" || r.Version != 1 || r.CreatedAt.IsZero() {
		t.Fatalf("decoded %+v", r)
	}
}

func TestLiquidityDeactivateDecode(t *testing.T) {
	c, got := server(t, 200, `{"data": {"deactivatedAt": "2026-02-19T08:00:37.185Z", "deactivatedBy": "86964e42-79dc-4267-a2ca-3612c4b095a8", "deactivationReason": "Paused by user", "fromAsset": "USDC", "id": "cad01df3-4041-4e81-a368-534fb4de5d12", "isActive": false, "maxAmount": null, "minAmount": "0.000006", "network": "testnet", "rate": "1", "status": "deactivated", "toAsset": "BNB", "version": 2}, "message": "Rate deactivated successfully", "statusCode": 200}`)
	r, err := c.LiquidityPool.Deactivate(context.Background(), "cad01df3", LiquidityDeactivateRateRequest{Reason: "Paused by user"})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "PATCH" || mustJSON(got.body) != `{"reason":"Paused by user"}` {
		t.Fatalf("got %s %s", got.method, mustJSON(got.body))
	}
	if r.Status != "deactivated" || r.DeactivatedAt == nil || r.DeactivationReason != "Paused by user" {
		t.Fatalf("decoded %+v", r)
	}
}

func TestLiquidityMarketBenchmarkOneOrMany(t *testing.T) {
	c, got := server(t, 200, `{"data": [{"fromAsset": "USDT", "toAsset": "cNGN", "bestRate": "1547.50"}, {"fromAsset": "cNGN", "toAsset": "USDT", "bestRate": null}], "message": "Market benchmark retrieved", "statusCode": 200}`)
	bs, err := c.LiquidityPool.MarketBenchmark(context.Background(), LiquidityBenchmarkParams{Pairs: []string{"USDT:cNGN", "cNGN:USDT"}, Amount: "500"})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "amount=500&pairs=USDT%3AcNGN%2CcNGN%3AUSDT" {
		t.Fatalf("query = %q", got.query)
	}
	if len(bs) != 2 || bs[0].BestRate != "1547.50" || bs[1].BestRate != "" {
		t.Fatalf("decoded %+v", bs)
	}

	c, got = server(t, 200, `{"data": {"fromAsset": "USDT", "toAsset": "cNGN", "bestRate": "1547.50"}, "message": "Market benchmark retrieved", "statusCode": 200}`)
	bs, err = c.LiquidityPool.MarketBenchmark(context.Background(), LiquidityBenchmarkParams{FromAsset: "USDT", ToAsset: "cNGN"})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "fromAsset=USDT&toAsset=cNGN" {
		t.Fatalf("query = %q", got.query)
	}
	if len(bs) != 1 || bs[0].FromAsset != "USDT" || bs[0].BestRate != "1547.50" {
		t.Fatalf("decoded %+v", bs)
	}
}

func TestLiquidityCheckPairAndHistory(t *testing.T) {
	c, got := server(t, 200, `{"data": {"fromAsset": "USDT", "toAsset": "cNGN", "hasActiveRates": true, "activeRateCount": 2, "rates": [{"id": "d69078ef-2467-40f4-bb00-63394efe32c0", "rate": "1545.00", "minAmount": "10", "maxAmount": "1000", "version": 1, "createdAt": "2026-04-12T09:14:22.501Z"}]}, "message": "Pair check completed", "statusCode": 200}`)
	pc, err := c.LiquidityPool.CheckPair(context.Background(), "USDT", "cNGN")
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "fromAsset=USDT&toAsset=cNGN" {
		t.Fatalf("query = %q", got.query)
	}
	if !pc.HasActiveRates || pc.ActiveRateCount != 2 || pc.Rates[0].MaxAmount != "1000" {
		t.Fatalf("decoded %+v", pc)
	}

	c, _ = server(t, 200, `{"data": [{"id": "d69078ef-2467-40f4-bb00-63394efe32c0", "fromAsset": "BNB", "toAsset": "USDC", "network": "testnet", "rate": "1", "minAmount": "0.000005", "isActive": false, "maxAmount": null, "status": "deactivated", "version": 1, "previousRateId": null, "rootRateId": null, "createdBy": "86964e42-79dc-4267-a2ca-3612c4b095a8", "deactivatedBy": "86964e42-79dc-4267-a2ca-3612c4b095a8", "deactivatedAt": "2026-02-25T18:13:48.113Z", "deactivationReason": "re", "createdAt": "2026-02-19T07:50:17.042Z", "updatedAt": "2026-02-25T18:13:48.101Z"}], "message": "Rate history retrieved successfully", "meta": {"totalItems": 1, "itemCount": 1, "itemsPerPage": 10, "totalPages": 1, "currentPage": 1}, "statusCode": 200}`)
	h, err := c.LiquidityPool.History(context.Background(), "d69078ef", PageParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Data) != 1 || h.Meta.TotalItems != 1 || h.Data[0].Status != "deactivated" || h.Data[0].DeactivatedAt == nil {
		t.Fatalf("decoded %+v", h)
	}
}

func TestLiquidityTreasuryBalances(t *testing.T) {
	c, _ := server(t, 200, `{"data": [{"asset": "USDT", "totalBalance": "5000.00", "totalConvertedBalance": "5000.00", "chains": [{"blockchain": "ethereum", "balance": "3000.00", "convertedBalance": "3000.00", "walletAddress": "0xA1b2C3d4E5f6789012345678901234567890aBcD"}]}], "message": "Treasury balances retrieved successfully", "statusCode": 200}`)
	tb, err := c.LiquidityPool.TreasuryBalances(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tb) != 1 || tb[0].TotalBalance != "5000.00" || tb[0].Chains[0].Blockchain != "ethereum" {
		t.Fatalf("decoded %+v", tb)
	}
}
