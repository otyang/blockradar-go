package blockradar

import (
	"context"
	"testing"
)

func TestMiscRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx, s := context.Background(), c.Misc
	calls := []func() error{
		func() error { _, err := s.Assets(ctx, MiscAssetsParams{}); return err },
		func() error { _, err := s.Rates(ctx, "USD", "ETH"); return err },
		func() error { _, err := s.Blockchains(ctx); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, hits(), []string{
		"GET /v1/assets",
		"GET /v1/assets/rates",
		"GET /v1/blockchains",
	})
}

func TestMiscAssetsDecodes(t *testing.T) {
	c, got := server(t, 200, `{"data":[{"address":"0x4Fabb145d64652a948d72533023f6E7A623C7C53","blockchain":{"createdAt":"2023-04-28T14:44:06.397Z","derivationPath":"m/44'/60'/0'/0","id":"8a864e95-4b86-423d-9ba4-f4e692ec121e","isActive":true,"isEvmCompatible":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800081/crypto-assets/ethereum-eth-logo_idraq2.png","name":"ethereum","slug":"ethereum","symbol":"eth","tokenStandard":"ERC20","updatedAt":"2024-06-14T22:43:52.987Z"},"createdAt":"2023-04-28T14:44:13.905Z","decimals":18,"id":"7a070953-694b-4ee9-b783-545a6ff66cd0","isActive":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/binance-usd-busd-logo_gumjv0.png","name":"Binance USD","network":"mainnet","standard":"ERC20","symbol":"BUSD","updatedAt":"2024-06-14T22:44:00.075Z"}],"message":"Assets retrieved successfully","statusCode":200}`)
	as, err := c.Misc.Assets(context.Background(), MiscAssetsParams{Network: "mainnet", Symbol: "USDC", BlockchainID: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "blockchainId=B&network=mainnet&symbol=USDC" {
		t.Fatalf("query = %q", got.query)
	}
	if len(as) != 1 || as[0].Symbol != "BUSD" || as[0].Blockchain.Slug != "ethereum" || as[0].Decimals != 18 {
		t.Fatalf("assets = %+v", as)
	}
}

func TestMiscRatesDecodes(t *testing.T) {
	c, got := server(t, 200, `{"data":{"BUSD":{"USD":0.9997},"DAI":{"USD":0.9998},"USDC":{"USD":0.9997},"USDT":{"USD":0.9997}},"message":"Rates retrieved successfully","statusCode":200}`)
	r, err := c.Misc.Rates(context.Background(), "USD", "USDC", "USDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "assets=USDC%2CUSDT&currency=USD" {
		t.Fatalf("query = %q", got.query)
	}
	if r["USDC"]["USD"] != "0.9997" || r["DAI"]["USD"] != "0.9998" {
		t.Fatalf("rates = %+v", r)
	}
}

func TestMiscBlockchainsDecodes(t *testing.T) {
	c, _ := server(t, 200, `{"data":[{"createdAt":"2024-05-14T11:53:33.095Z","derivationPath":"m/44'/60'/0'/0","id":"85ffc132-3972-4c9e-99a5-5cf0ccb688bf","isActive":true,"isEvmCompatible":true,"isL2":false,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800081/crypto-assets/ethereum-eth-logo_idraq2.png","name":"ethereum","slug":"ethereum","symbol":"eth","tokenStandard":"ERC20","updatedAt":"2024-11-26T13:04:13.936Z"}],"message":"Blockchains retrieved successfully","statusCode":200}`)
	bs, err := c.Misc.Blockchains(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 1 || bs[0].Slug != "ethereum" || !bs[0].IsEvmCompatible || bs[0].CreatedAt.IsZero() {
		t.Fatalf("blockchains = %+v", bs)
	}
}
