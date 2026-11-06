package blockradar

import (
	"context"
	"testing"
)

func TestTransactionsRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx, s := context.Background(), c.Transactions
	calls := []func() error{
		func() error { _, err := s.List(ctx, "W", TransactionListParams{}); return err },
		func() error { _, err := s.Get(ctx, "W", "X"); return err },
		func() error { _, err := s.WebhookLogs(ctx, "W", "X"); return err },
		func() error { _, err := s.Cancel(ctx, "W", "X"); return err },
		func() error { _, err := s.Retry(ctx, "W", "X"); return err },
		func() error { return s.SweepAssets(ctx, "W", TransactionSweepRequest{TransactionID: "X"}) },
		func() error { _, err := s.ResendWebhook(ctx, "W", "X"); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, hits(), []string{
		"GET /v1/wallets/W/transactions",
		"GET /v1/wallets/W/transactions/X",
		"GET /v1/wallets/W/transactions/X/webhooks",
		"POST /v1/wallets/W/transactions/cancel",
		"POST /v1/wallets/W/transactions/retry",
		"POST /v1/wallets/W/transactions/sweep/assets",
		"POST /v1/wallets/W/transactions/webhooks/resend",
	})
}

func TestTransactionsListDecodes(t *testing.T) {
	c, got := server(t, 200, `{"analytics":{"depositsCount":38,"depositsSumIn24Hours":0,"transactionsCount":64,"transactionsSumIn24Hours":0,"withdrawsCount":22,"withdrawsSumIn24Hours":0},"data":[{"address":null,"amlScreening":{"message":"Address is not sanctioned","provider":"ofac","status":"success"},"amount":"5","amountPaid":"5","asset":{"address":"0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238","createdAt":"2024-05-14T11:53:33.682Z","decimals":6,"id":"fe04a28c-c615-4e41-8eda-f84c862864f5","isActive":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800083/crypto-assets/usd-coin-usdc-logo_fs9mhv.png","name":"USDC Coin","network":"testnet","standard":"ERC20","symbol":"USDC","updatedAt":"2024-06-14T22:32:12.589Z"},"assetSwept":null,"assetSweptAmount":null,"assetSweptAt":null,"assetSweptGasFee":null,"assetSweptHash":null,"assetSweptRecipientAddress":null,"assetSweptSenderAddress":null,"beneficiary":null,"blockHash":"0x39e1a06ddf5940b457043c43b6c0271274503771d1ae440aa18cc438faf84c17","blockNumber":6626690,"blockchain":{"createdAt":"2024-05-14T11:53:33.095Z","derivationPath":"m/44'/60'/0'/0","id":"85ffc132-3972-4c9e-99a5-5cf0ccb688bf","isActive":true,"isEvmCompatible":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800081/crypto-assets/ethereum-eth-logo_idraq2.png","name":"ethereum","slug":"ethereum","symbol":"eth","tokenStandard":"ERC20","updatedAt":"2024-06-14T22:32:11.983Z"},"chainId":null,"confirmations":6,"confirmed":true,"createdAt":"2024-09-03T18:19:18.301Z","currency":"USD","fee":null,"gasFee":"0.00060714270605711","gasPrice":"13477983130","gasUsed":"45047","hash":"0xf071b647856716e354c378d208522943cb4e37339e6679c99b9a03f2db4a2025","id":"1d9b9b89-7326-482d-a048-67540e9d129e","metadata":null,"network":"testnet","note":null,"reason":null,"recipientAddress":"0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22","reference":"WGYa4Qqq01","senderAddress":"0x947514e4B803e312C312da0F1B41fEDdbe15ae7a","status":"SUCCESS","type":"WITHDRAW","updatedAt":"2024-09-03T18:20:40.077Z"}],"message":"Transactions fetched successfully","meta":{"currentPage":1,"itemCount":64,"itemsPerPage":100,"totalItems":64,"totalPages":1},"statusCode":200}`)
	l, err := c.Transactions.List(context.Background(), "W", TransactionListParams{
		Hash: "0xf0", Type: "WITHDRAW", Order: "DESC", Assets: []string{"USDC"}, Page: 2, Limit: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "assets=USDC&hash=0xf0&limit=50&order=DESC&page=2&type=WITHDRAW" {
		t.Fatalf("query = %q", got.query)
	}
	if l.Analytics.DepositsCount != 38 || l.Analytics.TransactionsCount != 64 || l.Meta.TotalItems != 64 || len(l.Data) != 1 {
		t.Fatalf("list = %+v", l)
	}
	tx := l.Data[0]
	if tx.ID != "1d9b9b89-7326-482d-a048-67540e9d129e" || tx.Amount != "5" || tx.Address != nil || tx.Asset.Symbol != "USDC" || tx.AMLScreening.Provider != "ofac" {
		t.Fatalf("tx = %+v", tx.Transaction)
	}
}

func TestTransactionsCancelDecodes(t *testing.T) {
	c, got := server(t, 200, `{"data":{"address":null,"amlScreening":{},"amount":"0.1","amountPaid":"0.1","amountUSD":"0.1","asset":{"address":"0x55d398326f99059fF775485246999027B3197955","createdAt":"2024-05-14T16:53:33.671Z","currency":"USD","decimals":18,"id":"e572fb27-3e52-4dd0-8452-f9a681be03f3","isActive":true,"isNative":false,"name":"Tether USD","network":"mainnet","standard":"BEP20","symbol":"USDT","updatedAt":"2025-04-17T03:50:19.952Z"},"assetSwept":null,"assetSweptAmount":null,"assetSweptAt":null,"assetSweptGasFee":null,"assetSweptHash":null,"assetSweptRecipientAddress":null,"assetSweptSenderAddress":null,"blockHash":null,"blockNumber":null,"blockchain":{"createdAt":"2024-05-14T16:53:33.106Z","derivationPath":"m/44'/60'/0'/0","id":"b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa","isActive":true,"isEvmCompatible":true,"isL2":false,"name":"BNB smart chain","slug":"bnb-smart-chain","symbol":"bnb","tokenStandard":"BEP20","updatedAt":"2024-11-26T19:04:13.941Z"},"chainId":null,"confirmations":null,"confirmed":false,"createdAt":"2025-09-23T15:14:30.894Z","createdChannel":"dashboard","currency":"USD","fee":"0.00003","feeHash":"0xf41d394f91dfb1c07890bac1cf5d408f6c0edb526ed371b481ab46caf0acb1d2","gasFee":null,"gasPrice":null,"gasUsed":null,"hash":null,"id":"5f65c56e-58d9-4b26-a2fd-98b5b9cbd26b","metadata":null,"network":"mainnet","note":" Withdrawal of 0.1 USDT from 0xD2b6be31932E0294F2ebD14a008C3f1E05B47BC4 to 0xb55c054D8eE75224E1033e6eC775B4F62D942b43 on BNB smart chain","rate":null,"rateUSD":"1","reason":"Withdrawal of 0.1 from 0xD2b6be31932E0294F2ebD14a008C3f1E05B47BC4 USDT to 0xb55c054D8eE75224E1033e6eC775B4F62D942b43 on BNB smart chain","recipientAddress":"0xb55c054D8eE75224E1033e6eC775B4F62D942b43","reference":"OjAhIPve6kAX3CQwPVkg","senderAddress":"0xD2b6be31932E0294F2ebD14a008C3f1E05B47BC4","signedTransaction":null,"status":"CANCELLED","toAmount":null,"toCurrency":null,"tokenAddress":"0x55d398326f99059fF775485246999027B3197955","type":"WITHDRAW","updatedAt":"2025-09-23T15:28:02.993Z"},"message":"Transaction cancelled","statusCode":200}`)
	tx, err := c.Transactions.Cancel(context.Background(), "W", "960661a2")
	if err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"id":"960661a2"}` {
		t.Fatalf("body = %s", b)
	}
	if tx.Status != "CANCELLED" || tx.Fee != "0.00003" || tx.FeeHash == "" || tx.CreatedChannel != "dashboard" || tx.BlockNumber != 0 || tx.RateUSD != "1" {
		t.Fatalf("tx = %+v", tx)
	}
}

func TestTransactionsRetryDecodes(t *testing.T) {
	c, _ := server(t, 200, `{"data":{"amlScreening":{"message":"sender address is not sanctioned","provider":"ofac","status":"passed"},"amount":"0.01","asset":{"address":"0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238","createdAt":"2024-05-14T14:53:33.682Z","decimals":6,"id":"fe04a28c-c615-4e41-8eda-f84c862864f5","isActive":true,"name":"USDC Coin","network":"testnet","symbol":"USDC","updatedAt":"2024-05-27T12:24:11.951Z"},"assetSwept":true,"assetSweptAt":"2024-06-06T06:00:41.053Z","blockHash":"0x8733defb28554d62c57ea0ddfd3cf4b9ee8e52128a0176cbfd2508ecad71bc4a","blockNumber":6049098,"blockchain":{"createdAt":"2024-05-14T14:53:33.095Z","derivationPath":"m/44'/60'/0'/0","id":"85ffc132-3972-4c9e-99a5-5cf0ccb688bf","isActive":true,"isEvmCompatible":true,"name":"ethereum","slug":"ethereum","symbol":"eth","updatedAt":"2024-06-05T11:10:50.577Z"},"confirmations":1,"confirmed":true,"createdAt":"2024-06-06T08:15:02.079Z","gasPrice":"0.000000002999993809","gasUsed":62147,"hash":"0x02e2400c5858af6be3a0c060f0185a31a3a859243dededbead4a81f258ba31d0","id":"f3730f9a-a0bd-4fe4-9fad-90cf42c2a997","metadata":{"id":"0x"},"network":"testnet","note":null,"reason":"Funds swept successfully","recipientAddress":"0xbdF23C259346d326BA8847A4FCed31E1c9030746","reference":"EClNLYLcG2","senderAddress":"0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22","status":"PENDING","type":"WITHDRAW","updatedAt":"2024-06-06T09:00:41.068Z"},"message":"Transaction retried","statusCode":200}`)
	tx, err := c.Transactions.Retry(context.Background(), "W", "X")
	if err != nil {
		t.Fatal(err)
	}
	if tx.GasUsed != "62147" || tx.Status != "PENDING" || tx.AssetSweptAt == nil || tx.Metadata["id"] != "0x" {
		t.Fatalf("tx = %+v", tx.Transaction)
	}
}

func TestTransactionsWebhookLogsDecodes(t *testing.T) {
	c, _ := server(t, 200, `{"data":[{"attempts":1,"createdAt":"2024-07-23T10:56:42.073Z","event":"deposit.success","id":"d6b0c629-2887-43f2-b812-f61e215c7ce8","payload":{"address":{"address":"0xd09693EbfDE205cDcDD830160f15BC8FdBf08654","configurations":{"aml":{"message":"Address is not sanctioned","provider":"ofac","status":"success"}},"createdAt":"2024-07-23T09:10:50.267Z","derivationPath":"m/44'/60'/0'/0/48","id":"06823986-5f7d-42c3-9d10-f42cba7d95ea","isActive":true,"metadata":{"id":"slack3 0x0e01 0x0e01"},"name":"slack3 0x0e01","network":"testnet","type":"INTERNAL","updatedAt":"2024-07-23T09:10:50.267Z"},"amlScreening":{"message":"Address is not sanctioned","provider":"ofac","status":"success"},"amount":"0.3","amountPaid":"0.3","asset":{"address":"0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238","createdAt":"2024-05-14T11:53:33.682Z","decimals":6,"id":"fe04a28c-c615-4e41-8eda-f84c862864f5","isActive":true,"name":"USDC Coin","network":"testnet","standard":"ERC20","symbol":"USDC","updatedAt":"2024-06-14T22:32:12.589Z"},"assetSwept":null,"assetSweptAmount":null,"assetSweptAt":null,"assetSweptGasFee":null,"assetSweptHash":null,"assetSweptRecipientAddress":null,"assetSweptSenderAddress":null,"blockHash":"0xee3dd3a7109190eb5b9cca683810f54ce29fa18495f6660341c2c3f318ae2260","blockNumber":6361538,"blockchain":{"createdAt":"2024-05-14T11:53:33.095Z","derivationPath":"m/44'/60'/0'/0","id":"85ffc132-3972-4c9e-99a5-5cf0ccb688bf","isActive":true,"isEvmCompatible":true,"name":"ethereum","slug":"ethereum","symbol":"eth","tokenStandard":"ERC20","updatedAt":"2024-06-14T22:32:11.983Z"},"confirmations":1,"confirmed":true,"createdAt":"2024-07-23T10:56:40.914Z","currency":"USD","gasFee":"0.001013543448643627","gasPrice":"16305658853","gasUsed":"62159","hash":"0xccc0b8f2e3fd99779c331fabb3421f12aa67765d2dc876eeda95e9e2456826c3","id":"21ecf6a9-4e41-4ec6-8c1a-958cec64bbe3","metadata":{"id":"slack3 0x0e01 0x0e01"},"network":"testnet","note":null,"reason":null,"recipientAddress":"0xd09693EbfDE205cDcDD830160f15BC8FdBf08654","reference":"kihJcwTH9","senderAddress":"0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22","status":"SUCCESS","type":"DEPOSIT","updatedAt":"2024-07-23T10:56:40.914Z","wallet":{"address":"0x49A204B430F6948847e20846A8d9Bbee5eE04D2A","business":{"createdAt":"2024-05-15T06:01:53.045Z","id":"7c323cfb-c1b6-4a3e-b72d-7433870812ac","name":"Blockradar Inc","sector":"infrastructure","status":"ACTIVE","updatedAt":"2024-07-10T17:09:47.057Z"},"createdAt":"2024-05-15T06:09:24.777Z","derivationPath":"m/44'/60'/0'/0/1","description":"Yooooo","id":"3f9aca5c-38ee-4e1d-ab67-c084a2e37bb2","isActive":true,"name":"updated wallet yeah","network":"testnet","updatedAt":"2024-06-07T09:45:35.631Z"}},"reference":"21ecf6a9-4e41-4ec6-8c1a-958cec64bbe3","response":"Ok","status":"success","updatedAt":"2024-07-23T10:56:42.073Z","url":"https://example.com/webhook"}],"message":"Transaction webhook logs fetched successfully","statusCode":200}`)
	logs, err := c.Transactions.WebhookLogs(context.Background(), "W", "X")
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Event != "deposit.success" || string(logs[0].Response) != `"Ok"` || logs[0].Status != "success" {
		t.Fatalf("logs = %+v", logs)
	}
}

func TestTransactionsSweepBody(t *testing.T) {
	c, got := server(t, 200, `{"message":"Sweep initiated successfully","statusCode":200}`)
	if err := c.Transactions.SweepAssets(context.Background(), "W", TransactionSweepRequest{TransactionID: "T"}); err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"transactionId":"T"}` {
		t.Fatalf("body = %s", b)
	}
}

func TestTransactionsAnalyticsLiveShape(t *testing.T) {
	// Analytics as returned by the live API (field names differ from the docs).
	c, _ := server(t, 200, `{"message":"ok","statusCode":200,"data":[],
		"analytics":{"totalDepositsSum":20.0226,"totalWithdrawsSum":0,"totalTransactionsSum":21.02373,"totalNetworkFee":0,
		"totalTransactionsCount":5,"totalWithdrawsCount":0,"totalDepositsCount":1,"totalSignedCount":4},
		"meta":{"totalItems":1,"itemCount":1,"itemsPerPage":5,"totalPages":1,"currentPage":1}}`)
	got, err := c.Transactions.List(context.Background(), "w", TransactionListParams{})
	if err != nil {
		t.Fatal(err)
	}
	a := got.Analytics
	if a.TotalDepositsCount != 1 || a.TotalDepositsSum != "20.0226" || a.TotalSignedCount != 4 || got.Meta.TotalItems != 1 {
		t.Fatalf("analytics = %+v", a)
	}
}
