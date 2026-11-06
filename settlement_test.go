package blockradar

import (
	"context"
	"testing"
)

func TestAutoSettlementsRoutes(t *testing.T) {
	c, routes := recordRoutes(t, "")
	ctx := context.Background()
	a := c.AutoSettlements
	rule := AutoSettlementRuleRequest{
		Source:      AutoSettlementSource{Assets: []string{"USDC"}},
		Destination: AutoSettlementDestination{Blockchain: "base", Asset: "USDC"},
	}

	calls := []func() error{
		func() error { _, err := a.Get(ctx, "W"); return err },
		func() error { _, err := a.Update(ctx, "W", false); return err },
		func() error { _, err := a.ListRules(ctx, "W"); return err },
		func() error { _, err := a.GetRule(ctx, "W", "R"); return err },
		func() error { _, err := a.CreateRule(ctx, "W", rule); return err },
		func() error { _, err := a.UpdateRule(ctx, "W", "R", AutoSettlementRuleUpdate{}); return err },
		func() error { _, err := a.DeleteRule(ctx, "W", "R"); return err },
		func() error {
			_, err := a.CopyRules(ctx, "W", AutoSettlementCopyRequest{Rules: []AutoSettlementRuleRequest{rule}, TargetWalletIDs: []string{"W2"}})
			return err
		},
		func() error { _, err := a.AddressGet(ctx, "W", "A"); return err },
		func() error { _, err := a.AddressUpdate(ctx, "W", "A", true); return err },
		func() error { _, err := a.AddressListRules(ctx, "W", "A"); return err },
		func() error { _, err := a.AddressGetRule(ctx, "W", "A", "R"); return err },
		func() error { _, err := a.AddressCreateRule(ctx, "W", "A", rule); return err },
		func() error { _, err := a.AddressUpdateRule(ctx, "W", "A", "R", rule); return err },
		func() error { _, err := a.AddressDeleteRule(ctx, "W", "A", "R"); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, routes(), []string{
		"GET /v1/wallets/W/auto-settlements",
		"PATCH /v1/wallets/W/auto-settlements",
		"GET /v1/wallets/W/auto-settlements/rules",
		"GET /v1/wallets/W/auto-settlements/rules/R",
		"POST /v1/wallets/W/auto-settlements/rules",
		"PATCH /v1/wallets/W/auto-settlements/rules/R",
		"DELETE /v1/wallets/W/auto-settlements/rules/R",
		"POST /v1/wallets/W/auto-settlements/rules/copy",
		"GET /v1/wallets/W/addresses/A/auto-settlements",
		"PATCH /v1/wallets/W/addresses/A/auto-settlements",
		"GET /v1/wallets/W/addresses/A/auto-settlements/rules",
		"GET /v1/wallets/W/addresses/A/auto-settlements/rules/R",
		"POST /v1/wallets/W/addresses/A/auto-settlements/rules",
		"PATCH /v1/wallets/W/addresses/A/auto-settlements/rules/R",
		"DELETE /v1/wallets/W/addresses/A/auto-settlements/rules/R",
	})
}

func TestAutoSettlementGetDecodes(t *testing.T) {
	c, _ := server(t, 200, `{"data": {"isActive": true, "name": "Base Master wallet swap auto ettlement", "rules": [{"id": "586db9b9-a6d8-4185-af72-395f21254eaf", "type": "swap", "isGateway": false, "isReward": false, "rewardProvider": "aave", "rewardType": "defi", "useTransactionAmount": false, "deductionPercentage": "0", "inheritance": {"mode": "selected_children", "childAddressIds": ["c1"]}, "destination": {"address": "0x0077777d7EBA4688BDeF3E311b846F25870A19B9", "asset": "USDC", "blockchain": "base", "isGateway": true}, "isActive": true, "name": "Swap from Base USDC to Gateway", "order": "FASTEST", "slippageTolerance": "5", "source": {"assets": ["USDC"], "blockchain": "base", "maxAmount": "1000", "minAmount": "1"}}]}, "message": "Auto settlements fetched successfully", "statusCode": 200}`)

	as, err := c.AutoSettlements.Get(context.Background(), "w1")
	if err != nil {
		t.Fatal(err)
	}
	if !as.IsActive || len(as.Rules) != 1 {
		t.Fatalf("decoded %+v", as)
	}
	r := as.Rules[0]
	if r.Type != "swap" || r.Inheritance.ChildAddressIDs[0] != "c1" || !r.Destination.IsGateway ||
		r.Source.MaxAmount != "1000" || r.SlippageTolerance != "5" || r.Order != "FASTEST" {
		t.Fatalf("rule %+v", r)
	}
}

func TestAutoSettlementCreateRuleBody(t *testing.T) {
	c, got := server(t, 200, `{"data": {"id": "9c1b4f77-2a3d-4f01-b6c8-77e4a1d0c512", "type": "earn", "destination": {"asset": "USDC", "blockchain": "base"}, "isActive": true, "isReward": true, "rewardProvider": "aave", "rewardType": "defi", "name": "Earn on Base USDC deposits", "slippageTolerance": "0", "useTransactionAmount": false, "deductionPercentage": "10", "source": {"assets": ["USDC"], "blockchain": "base", "maxAmount": "-1", "minAmount": "10"}, "createdAt": "2025-08-26T17:31:02.140Z", "updatedAt": "2025-08-26T17:31:02.140Z"}, "message": "Auto settlement rule created successfully", "statusCode": 201}`)

	r, err := c.AutoSettlements.CreateRule(context.Background(), "w1", AutoSettlementRuleRequest{
		Type:                "earn",
		Name:                "Earn on Base USDC deposits",
		IsActive:            Bool(false),
		IsReward:            true,
		RewardProvider:      "aave",
		DeductionPercentage: "10",
		Inheritance:         &AutoSettlementInheritance{Mode: "master_only"},
		Source:              AutoSettlementSource{Assets: []string{"USDC"}, MinAmount: "10"},
		Destination:         AutoSettlementDestination{Blockchain: "base", Asset: "USDC"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "POST" || got.path != "/v1/wallets/w1/auto-settlements/rules" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	want := map[string]any{"type": "earn", "name": "Earn on Base USDC deposits", "isActive": false,
		"isReward": true, "rewardProvider": "aave", "deductionPercentage": "10",
		"inheritance": map[string]any{"mode": "master_only"},
		"source":      map[string]any{"assets": []any{"USDC"}, "minAmount": "10"},
		"destination": map[string]any{"blockchain": "base", "asset": "USDC"}}
	if b1, b2 := mustJSON(got.body), mustJSON(want); b1 != b2 {
		t.Fatalf("body = %s, want %s", b1, b2)
	}
	if r.ID == "" || !r.IsReward || r.Source.MaxAmount != "-1" || r.CreatedAt.IsZero() {
		t.Fatalf("decoded %+v", r)
	}
}

func TestAutoSettlementPartialUpdateAndToggle(t *testing.T) {
	c, got := server(t, 200, `{"data": {"id": "r1"}, "message": "ok", "statusCode": 200}`)

	_, err := c.AutoSettlements.UpdateRule(context.Background(), "w1", "r1", AutoSettlementRuleUpdate{
		IsActive: Bool(false), UseTransactionAmount: Bool(false), SlippageTolerance: "-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "PATCH" || mustJSON(got.body) != `{"isActive":false,"slippageTolerance":"-1","useTransactionAmount":false}` {
		t.Fatalf("got %s body %s", got.method, mustJSON(got.body))
	}

	if _, err := c.AutoSettlements.AddressUpdate(context.Background(), "w1", "a1", false); err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/wallets/w1/addresses/a1/auto-settlements" || mustJSON(got.body) != `{"isActive":false}` {
		t.Fatalf("got %s body %s", got.path, mustJSON(got.body))
	}
}

func TestAutoSettlementRuleRequestRoundTrip(t *testing.T) {
	r := AutoSettlementRule{ID: "r1", Type: "swap", IsActive: false, Order: "FASTEST",
		Source:      AutoSettlementSource{Assets: []string{"USDT"}, Blockchain: "base", MinAmount: "1", MaxAmount: "1000"},
		Destination: AutoSettlementDestination{Blockchain: "optimism", Asset: "USDC"}}
	req := r.Request()
	if req.IsActive == nil || *req.IsActive || req.Order != "FASTEST" || req.Source.MaxAmount != "1000" {
		t.Fatalf("request %+v", req)
	}
}

func TestAutoSettlementCopyRules(t *testing.T) {
	c, got := server(t, 200, `{"data": {"summary": {"walletsTargeted": 2, "rulesRequested": 1, "applied": 1, "skipped": 1, "failed": 0}, "results": [{"walletId": "3f6c9a11-9a7e-4a3c-8f2b-0d51e6c4a7b2", "walletName": "Base Mainnet Wallet", "chain": "base", "applied": [{"name": "Sweep USDC to treasury"}], "skipped": []}, {"walletId": "c81e7d40-5b6a-4f9e-9c3d-2a7f10b8e654", "walletName": "Optimism Mainnet Wallet", "chain": "optimism", "applied": [], "skipped": [{"name": "Sweep USDC to treasury", "reason": "A settlement rule for USDC already exists on the optimism wallet"}]}]}, "message": "Settlement rules copied", "statusCode": 200}`)

	res, err := c.AutoSettlements.CopyRules(context.Background(), "w1", AutoSettlementCopyRequest{
		Rules: []AutoSettlementRuleRequest{{
			Type: "withdraw", Name: "Sweep USDC to treasury",
			Source:      AutoSettlementSource{Assets: []string{"USDC"}},
			Destination: AutoSettlementDestination{Blockchain: "base", Asset: "USDC", Address: "0xabc"},
		}},
		TargetWalletIDs: []string{"w2", "w3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"rules":[{"destination":{"address":"0xabc","asset":"USDC","blockchain":"base"},"name":"Sweep USDC to treasury","source":{"assets":["USDC"]},"type":"withdraw"}],"targetWalletIds":["w2","w3"]}`
	if mustJSON(got.body) != want {
		t.Fatalf("body = %s", mustJSON(got.body))
	}
	if res.Summary.Skipped != 1 || len(res.Results) != 2 || res.Results[1].Skipped[0].Reason == "" || res.Results[0].Failed != "" {
		t.Fatalf("decoded %+v", res)
	}
}
