package blockradar

import (
	"context"
	"testing"
)

func TestRewardsRoutes(t *testing.T) {
	c, routes := recordRoutes(t, "")
	ctx := context.Background()
	rw := c.Rewards
	fee := RewardNetworkFeeRequest{AssetID: "a", Amount: "1"}

	calls := []func() error{
		func() error { _, err := rw.Business(ctx); return err },
		func() error { _, err := rw.Products(ctx, RewardProductsParams{}); return err },
		func() error { _, err := rw.SupportedChains(ctx); return err },
		func() error { _, err := rw.Transactions(ctx, RewardTransactionsParams{}); return err },
		func() error { _, err := rw.Wallet(ctx, "W", RewardWalletParams{}); return err },
		func() error { _, err := rw.WalletAssets(ctx, "W", RewardAssetsParams{}); return err },
		func() error {
			_, err := rw.Deposit(ctx, "W", RewardDepositRequest{AssetID: "a", Amount: "1"})
			return err
		},
		func() error { _, err := rw.DepositNetworkFee(ctx, "W", fee); return err },
		func() error {
			_, err := rw.Withdraw(ctx, "W", RewardWithdrawRequest{AssetID: "a", Amount: "max"})
			return err
		},
		func() error { _, err := rw.WithdrawNetworkFee(ctx, "W", fee); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, routes(), []string{
		"GET /v1/rewards",
		"GET /v1/rewards/products",
		"GET /v1/rewards/supported-chains",
		"GET /v1/rewards/transactions",
		"GET /v1/wallets/W/rewards",
		"GET /v1/wallets/W/rewards/assets",
		"POST /v1/wallets/W/rewards/deposit",
		"POST /v1/wallets/W/rewards/deposit/network-fee",
		"POST /v1/wallets/W/rewards/withdraw",
		"POST /v1/wallets/W/rewards/withdraw/network-fee",
	})
}

const rewardPositionJSON = `{"type":"defi","productId":"5d0cc05d-d792-4cfe-a189-4a552b2c917a","productKey":"aave-base-usdc","provider":"aave","apy":"2.41","balance":"4.365017","balanceInUSD":"4.365017","totalDeposited":"4.360000","totalDepositedInUSD":"4.360000","earnings":"0.005017","earningsInUSD":"0.005017","todayEarnings":"0.000187","weeklyEarnings":"0.001312","lifetimeEarnings":"0.026492","lifetimeEarningsInUSD":"0.026492","asset":{"id":"2f6298ec-81ca-44f4-a8f2-cc045a46d28c","isActive":true,"asset":{"id":"b1deda8d-7609-4895-b584-e8ef1f13e354","name":"USD Coin","symbol":"USDC","decimals":6,"currency":"USD","network":"mainnet","blockchain":{"name":"base","slug":"base","symbol":"eth"}}}}`

func TestRewardsWalletListAndSingle(t *testing.T) {
	c, got := server(t, 200, `{"message":"Wallet rewards retrieved successfully","statusCode":200,"data":[`+rewardPositionJSON+`]}`)
	ps, err := c.Rewards.Wallet(context.Background(), "w1", RewardWalletParams{AddressID: "ad1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/wallets/w1/rewards" || got.query != "addressId=ad1" {
		t.Fatalf("got %s?%s", got.path, got.query)
	}
	p := ps[0]
	if len(ps) != 1 || p.ProductKey != "aave-base-usdc" || p.APY != "2.41" || p.LifetimeEarnings != "0.026492" ||
		p.Asset.ID != "2f6298ec-81ca-44f4-a8f2-cc045a46d28c" || p.Asset.Asset.Blockchain.Slug != "base" {
		t.Fatalf("decoded %+v", ps)
	}

	// With assetId the API returns a single object.
	c, got = server(t, 200, `{"message":"ok","statusCode":200,"data":`+rewardPositionJSON+`}`)
	ps, err = c.Rewards.Wallet(context.Background(), "w1", RewardWalletParams{AssetID: "as1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "assetId=as1" || len(ps) != 1 || ps[0].Balance != "4.365017" {
		t.Fatalf("query %q decoded %+v", got.query, ps)
	}
}

func TestRewardsDepositBodyAndDecode(t *testing.T) {
	c, got := server(t, 200, `{"message":"Rewards Deposit initiated successfully","statusCode":200,"data":{"id":"ebbe9bd1-311f-4a69-8847-046aec033413","reference":null,"amount":"1","amountPaid":"1","status":"PENDING","type":"REWARD_DEPOSIT","createdAt":"2026-06-03T13:43:47.533Z"}}`)
	tx, err := c.Rewards.Deposit(context.Background(), "w1", RewardDepositRequest{
		AssetID: "as1", Amount: "1", ProductID: "p1", Metadata: map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "POST" || got.path != "/v1/wallets/w1/rewards/deposit" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	if b := mustJSON(got.body); b != `{"amount":"1","assetId":"as1","metadata":{"k":"v"},"productId":"p1"}` {
		t.Fatalf("body = %s", b)
	}
	if tx.ID != "ebbe9bd1-311f-4a69-8847-046aec033413" || tx.Type != "REWARD_DEPOSIT" || tx.Reference != "" || tx.CreatedAt.IsZero() {
		t.Fatalf("decoded %+v", tx)
	}
}

func TestRewardsWithdrawBody(t *testing.T) {
	c, got := server(t, 200, `{"statusCode":200,"message":"ok","data":{"id":"47ae3a92-4d36-4807-af9e-b83a4eb08d69","status":"PENDING","type":"REWARD_WITHDRAW"}}`)
	_, err := c.Rewards.Withdraw(context.Background(), "w1", RewardWithdrawRequest{AssetID: "as1", Amount: "max", AddressID: "ad1", Address: "0xabc"})
	if err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"address":"0xabc","addressId":"ad1","amount":"max","assetId":"as1"}` {
		t.Fatalf("body = %s", b)
	}
}

func TestRewardsNetworkFee(t *testing.T) {
	c, got := server(t, 200, `{"message":"Network fee fetched successfully","statusCode":200,"data":{"networkFee":"0.00006779052","networkFeeInUSD":"0.14371725","nativeBalance":"0.000929309612767174","nativeBalanceInUSD":"1.97015496","estimatedArrivalTime":30,"transactionFee":"0"}}`)
	fee, err := c.Rewards.WithdrawNetworkFee(context.Background(), "w1", RewardNetworkFeeRequest{AssetID: "as1", Amount: "max"})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/wallets/w1/rewards/withdraw/network-fee" || mustJSON(got.body) != `{"amount":"max","assetId":"as1"}` {
		t.Fatalf("got %s %s", got.path, mustJSON(got.body))
	}
	if fee.NetworkFeeInUSD != "0.14371725" || fee.EstimatedArrivalTime != 30 || fee.TransactionFee != "0" {
		t.Fatalf("decoded %+v", fee)
	}
}

func TestRewardsProductsAndAssets(t *testing.T) {
	c, got := server(t, 200, `{"message":"Earn products retrieved successfully","statusCode":200,"data":[{"productId":"5d0cc05d-d792-4cfe-a189-4a552b2c917a","productKey":"aave-base-usdc","strategyId":"aave-base-usdc","provider":"aave","type":"defi","name":"Aave V3 USDC","blockchain":"base","network":"mainnet","apy":"2.41","apyUpdatedAt":"2026-09-01T10:00:00.000Z","minDeposit":null,"depositsEnabled":true,"providerPaused":{"deposits":false,"withdrawals":false},"isActive":true,"createdAt":"2026-08-31T10:00:00.000Z","asset":{"id":"b1deda8d-7609-4895-b584-e8ef1f13e354","name":"USD Coin","symbol":"USDC","decimals":6,"currency":"USD","network":"mainnet"}}]}`)
	prods, err := c.Rewards.Products(context.Background(), RewardProductsParams{Blockchain: "base", Asset: "USDC"})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "asset=USDC&blockchain=base" {
		t.Fatalf("query = %q", got.query)
	}
	if len(prods) != 1 || prods[0].Name != "Aave V3 USDC" || prods[0].APYUpdatedAt == nil || prods[0].MinDeposit != "" || !prods[0].DepositsEnabled || prods[0].Asset.Symbol != "USDC" {
		t.Fatalf("decoded %+v", prods)
	}

	c, got = server(t, 200, `{"message":"Wallet reward assets retrieved successfully","statusCode":200,"data":[{"id":"2f6298ec-81ca-44f4-a8f2-cc045a46d28c","productId":"8f2f6a4d-90c1-4a52-b9be-52f10c8f6f21","productKey":"compound-base-usdc","strategyId":"compound-base-usdc","provider":"compound","type":"defi","apy":"4.43","depositsEnabled":true,"providerPaused":{"deposits":false,"withdrawals":true},"isActive":true,"asset":{"id":"b1deda8d-7609-4895-b584-e8ef1f13e354","name":"USD Coin","symbol":"USDC","decimals":6}}]}`)
	as, err := c.Rewards.WalletAssets(context.Background(), "w1", RewardAssetsParams{Type: "defi"})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "type=defi" || len(as) != 1 || as[0].Provider != "compound" || !as[0].ProviderPaused.Withdrawals {
		t.Fatalf("query %q decoded %+v", got.query, as)
	}
}

func TestRewardsSupportedChainsAndTransactions(t *testing.T) {
	c, _ := server(t, 200, `{"message":"Supported reward chains retrieved successfully","statusCode":200,"data":{"all":["ethereum","base"],"byType":{"defi":["ethereum","base"]},"byProvider":{"aave":["ethereum","base"],"compound":["base"]}}}`)
	sc, err := c.Rewards.SupportedChains(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.All) != 2 || len(sc.ByType["defi"]) != 2 || sc.ByProvider["compound"][0] != "base" {
		t.Fatalf("decoded %+v", sc)
	}

	c, got := server(t, 200, `{"message":"Business reward transactions retrieved successfully","statusCode":200,"data":[{"id":"7a7f2845-12bb-4117-8d43-ec1e4ccafb10","reference":"nByk7CjmaKfH9lqPAQQw","amount":"0.2","amountPaid":"0.2","status":"SUCCESS","type":"REWARD_WITHDRAW","createdAt":"2026-06-03T23:09:58.284Z","asset":{"symbol":"USDC","decimals":6,"blockchain":{"slug":"base","name":"base"}}}]}`)
	txs, err := c.Rewards.Transactions(context.Background(), RewardTransactionsParams{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "limit=20" || len(txs) != 1 || txs[0].Type != "REWARD_WITHDRAW" || txs[0].Asset.Blockchain.Slug != "base" {
		t.Fatalf("query %q decoded %+v", got.query, txs)
	}
}
