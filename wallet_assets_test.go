package blockradar

import (
	"context"
	"testing"
)

func TestAssetsRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx, s := context.Background(), c.Assets
	calls := []func() error{
		func() error { _, err := s.List(ctx, "W"); return err },
		func() error { _, err := s.Add(ctx, "W", "X"); return err },
		func() error { _, err := s.Update(ctx, "W", "X", AssetUpdateRequest{}); return err },
		func() error { _, err := s.Remove(ctx, "W", "X"); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, hits(), []string{
		"GET /v1/wallets/W/assets",
		"POST /v1/wallets/W/assets",
		"PATCH /v1/wallets/W/assets/X",
		"DELETE /v1/wallets/W/assets/X",
	})
}

func TestAssetsListDecodes(t *testing.T) {
	c, _ := server(t, 200, `{"data":[{"asset":{"address":"0x0000000000000000000000000000000000000000","blockchain":{"createdAt":"2024-05-14T11:53:33.106Z","derivationPath":"m/44'/60'/0'/0","id":"b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa","isActive":true,"isEvmCompatible":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png","name":"BNB smart chain","slug":"bnb-smart-chain","symbol":"bnb","tokenStandard":"BEP20","updatedAt":"2024-06-14T22:36:43.716Z"},"createdAt":"2024-05-14T11:53:33.678Z","decimals":18,"id":"ef2458b5-b292-4f5a-89c4-71c8576e6272","isActive":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png","name":"Binance Coin","network":"testnet","standard":null,"symbol":"BNB","updatedAt":"2024-10-27T07:52:16.162Z"},"createdAt":"2024-08-22T09:29:11.387Z","id":"98582300-2f1d-4ef4-99f2-4361319fc776","isActive":true,"updatedAt":"2024-08-22T09:29:11.387Z"}],"message":"Assets fetched successfully","statusCode":200}`)
	as, err := c.Assets.List(context.Background(), "W")
	if err != nil {
		t.Fatal(err)
	}
	if len(as) != 1 || as[0].ID != "98582300-2f1d-4ef4-99f2-4361319fc776" || as[0].Asset.Symbol != "BNB" || as[0].Asset.Blockchain.Name != "BNB smart chain" {
		t.Fatalf("assets = %+v", as)
	}
}

func TestAssetsBodies(t *testing.T) {
	c, got := server(t, 201, `{"data":{"asset":{"address":"0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238","createdAt":"2024-05-14T14:53:33.682Z","decimals":6,"id":"fe04a28c-c615-4e41-8eda-f84c862864f5","isActive":true,"name":"USDC Coin","network":"testnet","symbol":"USDC","updatedAt":"2024-05-27T12:24:11.951Z"},"createdAt":"2024-05-26T08:11:43.129Z","id":"c4c26507-207b-4070-b968-31171881c227","isActive":true,"updatedAt":"2024-06-06T09:37:55.873Z"},"message":"Asset added successfully","statusCode":200}`)
	ctx := context.Background()
	a, err := c.Assets.Add(ctx, "W", "fe04a28c")
	if err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"assetId":"fe04a28c"}` {
		t.Fatalf("add body = %s", b)
	}
	if a.ID != "c4c26507-207b-4070-b968-31171881c227" || !a.IsActive || a.Asset.Symbol != "USDC" {
		t.Fatalf("asset = %+v", a)
	}
	if _, err := c.Assets.Update(ctx, "W", "X", AssetUpdateRequest{IsActive: false}); err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"isActive":false}` {
		t.Fatalf("update body = %s", b)
	}
}
