package blockradar

import (
	"context"
	"testing"
)

func TestWalletsRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx, s := context.Background(), c.Wallets
	calls := []func() error{
		func() error { _, err := s.Get(ctx, "W", false); return err },
		func() error { _, err := s.Update(ctx, "W", WalletUpdateRequest{Name: "n"}); return err },
		func() error { _, err := s.Balance(ctx, "W", "X"); return err },
		func() error { _, err := s.Balances(ctx, "W"); return err },
		func() error { _, err := s.WebhookLogs(ctx, "W", PageParams{}); return err },
		func() error { return s.RescanBlocks(ctx, "W", WalletRescanRequest{}) },
		func() error { return s.SweepAssets(ctx, "W", WalletSweepRequest{}) },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, hits(), []string{
		"GET /v1/wallets/W",
		"PATCH /v1/wallets/W",
		"GET /v1/wallets/W/balance",
		"GET /v1/wallets/W/balances",
		"GET /v1/wallets/W/webhooks",
		"POST /v1/wallets/W/rescan/blocks",
		"POST /v1/wallets/W/sweep/assets",
	})
}

func TestWalletsGetDecodes(t *testing.T) {
	c, got := server(t, 200, `{"data":{"address":"0x947514e4B803e312C312da0F1B41fEDdbe15ae7a","analytics":{"currentBalance":58.88822990007824,"numberOfAssets":2,"unsweptBalance":0},"assets":[{"asset":{"address":"0x0000000000000000000000000000000000000000","blockchain":{"createdAt":"2024-05-14T11:53:33.106Z","derivationPath":"m/44'/60'/0'/0","id":"b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa","isActive":true,"isEvmCompatible":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png","name":"BNB smart chain","slug":"bnb-smart-chain","symbol":"bnb","tokenStandard":"BEP20","updatedAt":"2024-06-14T22:36:43.716Z"},"createdAt":"2024-05-14T11:53:33.678Z","decimals":18,"id":"ef2458b5-b292-4f5a-89c4-71c8576e6272","isActive":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png","name":"Binance Coin","network":"testnet","standard":null,"symbol":"BNB","updatedAt":"2024-10-27T07:52:16.162Z"},"createdAt":"2024-08-22T09:29:11.387Z","id":"98582300-2f1d-4ef4-99f2-4361319fc776","isActive":true,"updatedAt":"2024-08-22T09:29:11.387Z"},{"asset":{"address":"0x337610d27c682E347C9cD60BD4b3b107C9d34dDd","blockchain":{"createdAt":"2024-05-14T11:53:33.106Z","derivationPath":"m/44'/60'/0'/0","id":"b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa","isActive":true,"isEvmCompatible":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png","name":"BNB smart chain","slug":"bnb-smart-chain","symbol":"bnb","tokenStandard":"BEP20","updatedAt":"2024-06-14T22:36:43.716Z"},"createdAt":"2024-05-14T11:53:33.680Z","decimals":18,"id":"1890db58-f624-4972-a72b-e25387a81520","isActive":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800082/crypto-assets/tether-usdt-logo_wx7rwh.png","name":"Tether USD","network":"testnet","standard":"BEP20","symbol":"USDT","updatedAt":"2024-06-14T22:36:43.868Z"},"createdAt":"2024-08-22T10:33:45.481Z","id":"ae455f23-3824-4125-baab-d158315cbcbd","isActive":true,"updatedAt":"2024-08-22T10:33:45.481Z"}],"blockchain":{"createdAt":"2024-05-14T11:53:33.106Z","derivationPath":"m/44'/60'/0'/0","id":"b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa","isActive":true,"isEvmCompatible":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png","name":"BNB smart chain","slug":"bnb-smart-chain","symbol":"bnb","tokenStandard":"BEP20","updatedAt":"2024-06-14T22:36:43.716Z"},"createdAt":"2024-08-22T09:29:11.387Z","derivationPath":"m/44'/60'/0'/0/0","description":"This is BNB smart chain testnet master wallet","id":"4465468a-3c36-4536-918a-91d689e18a74","isActive":true,"name":"BNB smart chain Master Wallet","network":"testnet","status":"ACTIVE","updatedAt":"2024-10-30T07:11:56.221Z"},"message":"Wallet fetched successfully","statusCode":200}`)
	w, err := c.Wallets.Get(context.Background(), "W", true)
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "showPrivateKey=true" {
		t.Fatalf("query = %q", got.query)
	}
	if w.ID != "4465468a-3c36-4536-918a-91d689e18a74" || w.Status != "ACTIVE" || w.Blockchain.Slug != "bnb-smart-chain" {
		t.Fatalf("wallet = %+v", w)
	}
	if w.Analytics.CurrentBalance != "58.88822990007824" || len(w.Assets) != 2 || w.Assets[1].Asset.Symbol != "USDT" {
		t.Fatalf("analytics/assets = %+v %+v", w.Analytics, w.Assets)
	}
}

func TestWalletsBalancesDecodes(t *testing.T) {
	c, _ := server(t, 200, `{"data":[{"asset":{"asset":{"address":"0x036CbD53842c5426634e7929541eC2318f3dCF7e","blockchain":{"createdAt":"2024-05-27T08:31:14.966Z","derivationPath":"m/44'/60'/0'/0","id":"74733889-4ecd-403e-9840-94e87c043f24","isActive":true,"isEvmCompatible":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/Base_Network_Logo_vqyh7r.png","name":"base","slug":"base","symbol":"eth","tokenStandard":null,"updatedAt":"2024-10-27T07:52:16.115Z"},"createdAt":"2024-05-27T08:31:19.442Z","decimals":6,"id":"0927dc05-7d4e-4c0a-82c6-98ceb170ab84","isActive":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800083/crypto-assets/usd-coin-usdc-logo_fs9mhv.png","name":"USD Coin","network":"testnet","standard":null,"symbol":"USDC","updatedAt":"2024-05-27T09:24:11.963Z"},"createdAt":"2024-08-22T10:34:38.785Z","id":"dde9b0ce-599d-4ee4-9a17-960f06aa50a8","isActive":true,"updatedAt":"2024-08-22T10:34:38.785Z"},"balance":"0.049","convertedBalance":"0.049"}],"message":"Balances fetched successfully","statusCode":200}`)
	bs, err := c.Wallets.Balances(context.Background(), "W")
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 1 || bs[0].Balance != "0.049" || bs[0].Asset.ID != "dde9b0ce-599d-4ee4-9a17-960f06aa50a8" || bs[0].Asset.Asset.Decimals != 6 {
		t.Fatalf("balances = %+v", bs)
	}
}

func TestWalletsWebhookLogsDecodes(t *testing.T) {
	c, got := server(t, 200, `{"data":[{"attempts":2,"createdAt":"2024-07-24T02:51:04.316Z","event":"withdraw.success","id":"ad83cf3a-ed14-4d02-aa1e-6349137a4067","payload":{"address":null,"amlScreening":{"message":"Address is not sanctioned","provider":"ofac","status":"success"},"amount":"1","amountPaid":"1","asset":{"address":"0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238","createdAt":"2024-05-14T11:53:33.682Z","decimals":6,"id":"fe04a28c-c615-4e41-8eda-f84c862864f5","isActive":true,"name":"USDC Coin","network":"testnet","standard":"ERC20","symbol":"USDC","updatedAt":"2024-06-14T22:32:12.589Z"},"assetSwept":null,"assetSweptAmount":null,"assetSweptAt":null,"assetSweptGasFee":null,"assetSweptHash":null,"assetSweptRecipientAddress":null,"assetSweptSenderAddress":null,"blockHash":"0x984eae8712e72b9221307acc92470b3bf3ea350ae6c10abc8069ff9c95ea2c37","blockNumber":6365312,"blockchain":{"createdAt":"2024-05-14T11:53:33.095Z","derivationPath":"m/44'/60'/0'/0","id":"85ffc132-3972-4c9e-99a5-5cf0ccb688bf","isActive":true,"isEvmCompatible":true,"name":"ethereum","slug":"ethereum","symbol":"eth","tokenStandard":"ERC20","updatedAt":"2024-06-14T22:32:11.983Z"},"confirmations":1,"confirmed":true,"createdAt":"2024-07-24T02:51:03.541Z","currency":"USD","gasFee":"0.000001766085518659","gasPrice":"39205397","gasUsed":"45047","hash":"0x96297a97ca20a7fd0e23516af9bc5049d6512c7e27e82a58399a59f1f6668e06","id":"6f1b03d3-5f8b-4424-ae77-10f7f866d520","metadata":null,"network":"testnet","note":null,"reason":null,"recipientAddress":"0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22","reference":"TSZiRYfUQ3","senderAddress":"0x49A204B430F6948847e20846A8d9Bbee5eE04D2A","status":"SUCCESS","type":"WITHDRAW","updatedAt":"2024-07-24T02:51:03.541Z","wallet":{"address":"0x49A204B430F6948847e20846A8d9Bbee5eE04D2A","business":{"createdAt":"2024-05-15T06:01:53.045Z","id":"7c323cfb-c1b6-4a3e-b72d-7433870812ac","name":"Blockradar Inc","sector":"infrastructure","status":"ACTIVE","updatedAt":"2024-07-10T17:09:47.057Z"},"createdAt":"2024-05-15T06:09:24.777Z","derivationPath":"m/44'/60'/0'/0/1","description":"Yooooo","id":"3f9aca5c-38ee-4e1d-ab67-c084a2e37bb2","isActive":true,"name":"updated wallet yeah","network":"testnet","updatedAt":"2024-06-07T09:45:35.631Z"}},"reference":"6f1b03d3-5f8b-4424-ae77-10f7f866d520","response":{"status":404,"text":"Not Found"},"status":"failed","updatedAt":"2024-07-24T11:57:58.700Z","url":"https://example.com/webhook"}],"message":"Webhooks fetched successfully","meta":{"currentPage":2,"itemCount":10,"itemsPerPage":10,"totalItems":99,"totalPages":10},"statusCode":200}`)
	logs, err := c.Wallets.WebhookLogs(context.Background(), "W", PageParams{Page: 2, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "limit=10&page=2" {
		t.Fatalf("query = %q", got.query)
	}
	if logs.Meta.TotalItems != 99 || len(logs.Data) != 1 {
		t.Fatalf("meta = %+v", logs.Meta)
	}
	l := logs.Data[0]
	if l.Event != "withdraw.success" || l.Attempts != 2 || string(l.Response) != `{"status":404,"text":"Not Found"}` {
		t.Fatalf("log = %+v", l)
	}
	var tx Transaction
	if err := jsonUnmarshal(string(l.Payload), &tx); err != nil || tx.Reference != "TSZiRYfUQ3" {
		t.Fatalf("payload = %v %+v", err, tx)
	}
}

func TestWalletsBodies(t *testing.T) {
	c, got := server(t, 200, `{"message":"Sweep initiated successfully","statusCode":200}`)
	ctx := context.Background()
	if err := c.Wallets.SweepAssets(ctx, "W", WalletSweepRequest{TransactionID: "T", ForceSweep: Bool(false)}); err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"forceSweep":false,"transactionId":"T"}` {
		t.Fatalf("sweep body = %s", b)
	}
	if err := c.Wallets.RescanBlocks(ctx, "W", WalletRescanRequest{TransactionHash: "0xabc"}); err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"transactionHash":"0xabc"}` {
		t.Fatalf("rescan body = %s", b)
	}
	if _, err := c.Wallets.Balance(ctx, "W", "X"); err != nil {
		t.Fatal(err)
	}
	if got.query != "assetId=X" {
		t.Fatalf("balance query = %q", got.query)
	}
	if _, err := c.Wallets.Update(ctx, "W", WalletUpdateRequest{Description: "d"}); err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"description":"d"}` {
		t.Fatalf("update body = %s", b)
	}
}
