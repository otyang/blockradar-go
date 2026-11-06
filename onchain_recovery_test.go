package blockradar

import (
	"context"
	"testing"
)

func TestAssetRecoveryRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx, r := context.Background(), RecoveryRequest{}
	_ = c.AssetRecovery.Execute(ctx, "W", r)
	_, _ = c.AssetRecovery.NetworkFee(ctx, "W", r)
	assertRoutes(t, hits(), []string{
		"POST /v1/wallets/W/salvage",
		"POST /v1/wallets/W/salvage/network-fee",
	})
}

func TestAssetRecoveryExecute(t *testing.T) {
	c, got := server(t, 200, `{"message":"Salvage asset initiated successfully","statusCode":200}`)
	err := c.AssetRecovery.Execute(context.Background(), "w1", RecoveryRequest{
		SenderAddress: "0xS", RecipientAddress: "0xR", Amount: "3", BlockchainID: "b1", TokenAddress: "0xT", CrossNetwork: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"senderAddress": "0xS", "recipientAddress": "0xR", "amount": "3", "blockchainId": "b1", "tokenAddress": "0xT", "crossNetwork": true}
	if len(got.body) != len(want) {
		t.Fatalf("body = %v", got.body)
	}
	for k, v := range want {
		if got.body[k] != v {
			t.Fatalf("body[%s] = %v, want %v", k, got.body[k], v)
		}
	}
}

func TestAssetRecoveryNetworkFee(t *testing.T) {
	c, got := server(t, 200, `{"data": {"balance": "0.1", "fee": "0.000034492", "nativeBalance": "0.0", "symbol": "USDT"}, "message": "Salvage network fee fetched successfully", "statusCode": 200}`)
	fee, err := c.AssetRecovery.NetworkFee(context.Background(), "w1", RecoveryRequest{Amount: "3"})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/wallets/w1/salvage/network-fee" {
		t.Fatalf("path = %s", got.path)
	}
	if fee.Balance != "0.1" || fee.Fee != "0.000034492" || fee.NativeBalance != "0.0" || fee.Symbol != "USDT" {
		t.Fatalf("fee = %+v", fee)
	}
}
