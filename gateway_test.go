package blockradar

import (
	"context"
	"testing"
)

func TestGatewayRoutes(t *testing.T) {
	c, routes := recordRoutes(t, "")
	ctx := context.Background()
	g := c.Gateway
	dep := GatewayDepositRequest{Amount: "1"}
	wd := GatewayWithdrawRequest{Amount: "1", Blockchain: "base", Address: "0x1"}

	calls := []func() error{
		func() error { _, err := g.Wallets(ctx, PageParams{}); return err },
		func() error { _, err := g.Balance(ctx); return err },
		func() error { _, err := g.Transactions(ctx, PageParams{}); return err },
		func() error { _, err := g.Deposit(ctx, "W", dep); return err },
		func() error { _, err := g.DepositNetworkFee(ctx, "W", dep); return err },
		func() error { _, err := g.Withdraw(ctx, wd); return err },
		func() error { _, err := g.WithdrawNetworkFee(ctx, wd); return err },
		func() error { _, err := g.AddressBalance(ctx, "A"); return err },
		func() error { _, err := g.AddressTransactions(ctx, "A", PageParams{}); return err },
		func() error { _, err := g.AddressDeposit(ctx, "A", "W", dep); return err },
		func() error { _, err := g.AddressDepositNetworkFee(ctx, "A", "W", dep); return err },
		func() error { _, err := g.AddressWithdraw(ctx, "A", wd); return err },
		func() error { _, err := g.AddressWithdrawNetworkFee(ctx, "A", wd); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, routes(), []string{
		"GET /v1/gateway/wallets",
		"GET /v1/gateway/balance",
		"GET /v1/gateway/Transactions",
		"POST /v1/gateway/deposit/W",
		"POST /v1/gateway/deposit/W/network-fee",
		"POST /v1/gateway/withdraw",
		"POST /v1/gateway/withdraw/network-fee",
		"GET /v1/addresses/A/gateway/balance",
		"GET /v1/addresses/A/gateway/transactions",
		"POST /v1/addresses/A/gateway/deposit/W",
		"POST /v1/addresses/A/gateway/deposit/W/network-fee",
		"POST /v1/addresses/A/gateway/withdraw",
		"POST /v1/addresses/A/gateway/withdraw/network-fee",
	})
}

func TestGatewayWithdrawBodyAndDecode(t *testing.T) {
	c, got := server(t, 200, `{"data": {"amlScreening": {"message": "Address is not sanctioned", "provider": "ofac, fbi, tether, circle", "status": "success"}, "amount": "1", "amountPaid": "1", "amountUSD": "1", "asset": {"address": "0x036CbD53842c5426634e7929541eC2318f3dCF7e", "createdAt": "2024-05-27T14:31:19.442Z", "currency": "USD", "decimals": 6, "id": "0927dc05-7d4e-4c0a-82c6-98ceb170ab84", "isActive": true, "isNative": false, "name": "USD Coin", "network": "testnet", "standard": null, "symbol": "USDC", "updatedAt": "2025-04-17T04:50:20.019Z"}, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"createdAt": "2024-05-27T14:31:14.966Z", "derivationPath": "m/44'/60'/0'/0", "id": "74733889-4ecd-403e-9840-94e87c043f24", "isActive": true, "isEvmCompatible": true, "isL2": true, "name": "base", "slug": "base", "symbol": "eth", "tokenStandard": null, "updatedAt": "2024-11-26T20:04:13.945Z"}, "chainId": 84532, "confirmations": null, "confirmed": false, "createdAt": "2025-08-17T01:48:37.989Z", "currency": "USD", "fee": null, "feeHash": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": null, "id": "c5e9ec3c-c7b1-4d84-a76e-056b8246f461", "metadata": null, "network": "testnet", "note": "Withdrawal of 1 USDC from gateway smart wallet 0x0077777d7EBA4688BDeF3E311b846F25870A19B9", "rate": null, "rateUSD": "1", "reason": "Balance Before: 9.749252 USDC | Network Fee: 2.005999 USDC | Gasless: false", "recipientAddress": "0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22", "reference": "tcApNlezq", "senderAddress": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "signedTransaction": null, "status": "PENDING", "toAmount": null, "toCurrency": null, "tokenAddress": null, "type": "GATEWAY_WITHDRAW", "updatedAt": "2025-08-17T01:48:37.989Z", "wallet": {"address": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "configurations": null, "createdAt": "2024-08-22T16:05:46.167Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "This is Base wallet", "id": "e17dea25-6c01-4b3a-a4ae-51f411bf69cd", "isActive": true, "name": "Base Wallet", "network": "testnet", "status": "ACTIVE", "updatedAt": "2024-08-22T16:05:46.167Z"}}, "message": "Withdraw initiated successfully", "statusCode": 200}`)

	tx, err := c.Gateway.AddressWithdraw(context.Background(), "addr1", GatewayWithdrawRequest{
		Amount: "1", Blockchain: "base", Address: "0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22",
		Reference: "tcApNlezq", Metadata: map[string]any{"user_id": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "POST" || got.path != "/v1/addresses/addr1/gateway/withdraw" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	want := map[string]any{"amount": "1", "blockchain": "base", "address": "0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22",
		"reference": "tcApNlezq", "metadata": map[string]any{"user_id": 1}}
	if b1, b2 := mustJSON(got.body), mustJSON(want); b1 != b2 {
		t.Fatalf("body = %s, want %s", b1, b2)
	}
	if tx.ID != "c5e9ec3c-c7b1-4d84-a76e-056b8246f461" || tx.Type != "GATEWAY_WITHDRAW" || tx.Status != "PENDING" ||
		tx.Amount != "1" || tx.Fee != "" || tx.Asset.Symbol != "USDC" || tx.Wallet.Name != "Base Wallet" ||
		tx.Blockchain.Slug != "base" || tx.ChainID.String() != "84532" {
		t.Fatalf("decoded %+v", tx)
	}
}

func TestGatewayDepositBody(t *testing.T) {
	c, got := server(t, 200, `{"statusCode":200,"message":"ok","data":{"id":"tx1","type":"GATEWAY_DEPOSIT"}}`)
	_, err := c.Gateway.Deposit(context.Background(), "w1", GatewayDepositRequest{Amount: "10"})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/gateway/deposit/w1" || mustJSON(got.body) != `{"amount":"10"}` {
		t.Fatalf("got %s %s", got.path, mustJSON(got.body))
	}
}

func TestGatewayNetworkFeeNumberAndString(t *testing.T) {
	c, _ := server(t, 200, `{"data": {"estimatedArrivalTime": 10, "nativeBalance": "0.044395941712804974", "nativeBalanceInUSD": "0.0443915021186336935026", "networkFee": "0.00000013083861216", "networkFeeInUSD": "0.000000130825528298784", "transactionFee": 2.005999}, "message": "Withdraw network fees calculated successfully", "statusCode": 200}`)
	fee, err := c.Gateway.WithdrawNetworkFee(context.Background(), GatewayWithdrawRequest{Amount: "1", Blockchain: "base", Address: "0x1"})
	if err != nil {
		t.Fatal(err)
	}
	if fee.TransactionFee != "2.005999" || fee.NetworkFee != "0.00000013083861216" || fee.EstimatedArrivalTime != 10 {
		t.Fatalf("decoded %+v", fee)
	}
}

func TestGatewayAddressTransactionsPaged(t *testing.T) {
	c, got := server(t, 200, `{"data": [{"address": {"address": "0xb2893f617baB937d340ED5e97Ab15f70bb45667a", "configurations": {"aml": {"message": "Address is not sanctioned", "provider": "ofac, fbi, tether, circle", "status": "success"}, "autoSettlement": {"isActive": false, "rules": []}, "disableAutoSweep": true, "enableGaslessWithdraw": true, "showPrivateKey": false}, "createdAt": "2025-12-11T02:16:04.119Z", "derivationPath": "m/44'/60'/0'/0/691", "id": "d7c354eb-3b79-49a6-8562-a3c48beccfa8", "isActive": true, "metadata": null, "name": "Child Address Gateway", "network": "testnet", "type": "INTERNAL", "updatedAt": "2025-12-11T02:16:04.119Z"}, "amount": "0.1", "blockNumber": 48843272, "chainId": 43113, "confirmations": 3, "confirmed": true, "createdAt": "2025-12-11T03:04:25.043Z", "fee": null, "gasFee": "0.000000000000270586", "hash": "0x898ac7555a7a73279f5e1c2b9aa95b8a948cc79bf50bdaa26fe29ec10274281d", "id": "f2b7e8ca-a275-4a3a-a86c-1edc112ce4ee", "status": "SUCCESS", "type": "GATEWAY_WITHDRAW", "updatedAt": "2025-12-11T03:05:11.064Z"}], "message": "Gateway transactions fetched successfully", "meta": {"currentPage": 1, "itemCount": 2, "itemsPerPage": 10, "totalItems": 2, "totalPages": 1}, "statusCode": 200}`)
	list, err := c.Gateway.AddressTransactions(context.Background(), "a1", PageParams{Page: 2, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "limit=10&page=2" {
		t.Fatalf("query = %q", got.query)
	}
	if len(list.Data) != 1 || list.Meta.TotalItems != 2 || list.Data[0].Address.Name != "Child Address Gateway" || list.Data[0].BlockNumber != 48843272 {
		t.Fatalf("decoded %+v", list)
	}
}

func TestGatewayWalletsAndBalance(t *testing.T) {
	c, got := server(t, 200, `{"data": [{"address": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "assets": [{"asset": {"address": "0x0000000000000000000000000000000000000000", "createdAt": "2024-05-27T14:31:19.440Z", "currency": "ETH", "decimals": 18, "id": "ec9c982f-5d83-46e7-bda5-386100e795fc", "isActive": true, "isNative": true, "name": "Ethereum", "network": "testnet", "standard": null, "symbol": "ETH", "updatedAt": "2025-04-17T04:50:20.013Z"}, "createdAt": "2024-08-22T16:05:46.167Z", "id": "ff99a04e-b279-4933-9075-29f0ae12eea0", "isActive": true, "updatedAt": "2024-08-22T16:05:46.167Z"}], "balance": "0.2", "blockchain": {"name": "base", "slug": "base", "symbol": "eth"}, "configurations": null, "convertedBalance": "0.19998", "createdAt": "2024-08-22T16:05:46.167Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "This is Base wallet", "id": "e17dea25-6c01-4b3a-a4ae-51f411bf69cd", "isActive": true, "name": "Base Wallet", "network": "testnet", "status": "ACTIVE", "updatedAt": "2024-08-22T16:05:46.167Z"}], "message": "Wallets fetched successfully", "statusCode": 200}`)
	ws, err := c.Gateway.Wallets(context.Background(), PageParams{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "limit=5" {
		t.Fatalf("query = %q", got.query)
	}
	if len(ws) != 1 || ws[0].ID != "e17dea25-6c01-4b3a-a4ae-51f411bf69cd" || ws[0].Balance != "0.2" ||
		ws[0].ConvertedBalance != "0.19998" || ws[0].Assets[0].Asset.Symbol != "ETH" {
		t.Fatalf("decoded %+v", ws)
	}

	c, _ = server(t, 200, `{"data": {"asset": {"createdAt": "2024-05-27T14:31:19.442Z", "currency": "USD", "decimals": 6, "id": "0927dc05-7d4e-4c0a-82c6-98ceb170ab84", "isActive": true, "isNative": false, "name": "USD Coin", "network": "testnet", "symbol": "USDC", "updatedAt": "2025-04-17T04:50:20.019Z"}, "balance": "9.749252", "convertedBalance": "9.7482770748"}, "message": "Balance fetched successfully", "statusCode": 200}`)
	bal, err := c.Gateway.Balance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if bal.Balance != "9.749252" || bal.Asset.Symbol != "USDC" || bal.Asset.Decimals != 6 {
		t.Fatalf("decoded %+v", bal)
	}
}
