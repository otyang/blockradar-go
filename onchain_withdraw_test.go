package blockradar

import (
	"context"
	"testing"
)

func TestWithdrawRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx, r := context.Background(), WithdrawRequest{}
	_, _ = c.Withdraw.Execute(ctx, "W", r)
	_, _ = c.Withdraw.Sign(ctx, "W", r)
	_, _ = c.Withdraw.NetworkFee(ctx, "W", r)
	_, _ = c.Withdraw.AddressExecute(ctx, "W", "A", r)
	_, _ = c.Withdraw.AddressSign(ctx, "W", "A", r)
	_, _ = c.Withdraw.AddressNetworkFee(ctx, "W", "A", r)
	assertRoutes(t, hits(), []string{
		"POST /v1/wallets/W/withdraw",
		"POST /v1/wallets/W/withdraw/sign",
		"POST /v1/wallets/W/withdraw/network-fee",
		"POST /v1/wallets/W/addresses/A/withdraw",
		"POST /v1/wallets/W/addresses/A/withdraw/sign",
		"POST /v1/wallets/W/addresses/A/withdraw/network-fee",
	})
}

func TestWithdrawExecuteSingle(t *testing.T) {
	c, got := server(t, 200, withdrawSingleFixture)
	tx, err := c.Withdraw.Execute(context.Background(), "w1", WithdrawRequest{
		AssetID: "as1", Address: "0xabc", Amount: "10", Note: "hi",
		Metadata: map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.method != "POST" || got.path != "/v1/wallets/w1/withdraw" {
		t.Fatalf("got %s %s", got.method, got.path)
	}
	if got.body["assetId"] != "as1" || got.body["address"] != "0xabc" || got.body["amount"] != "10" || got.body["note"] != "hi" {
		t.Fatalf("body = %v", got.body)
	}
	for _, k := range []string{"assets", "reference", "recipientRouting"} {
		if _, ok := got.body[k]; ok {
			t.Fatalf("unexpected %q in body %v", k, got.body)
		}
	}
	if tx.ID != "081d6315-159f-4c38-b02a-c4708836c5bd" || tx.Amount != "10" || tx.Status != "PENDING" || tx.Type != "WITHDRAW" {
		t.Fatalf("tx = %+v", tx.Transaction)
	}
	if tx.Asset == nil || tx.Asset.Symbol != "USDC" || tx.Wallet == nil || tx.Wallet.ID != "d236a191-c1d4-423c-a439-54ce6542ca41" {
		t.Fatalf("asset/wallet = %+v %+v", tx.Asset, tx.Wallet)
	}
}

func TestWithdrawExecuteBatch(t *testing.T) {
	c, got := server(t, 200, withdrawBatchFixture)
	res, err := c.Withdraw.AddressExecute(context.Background(), "w1", "a1", WithdrawRequest{Assets: []WithdrawAsset{
		{ID: "as1", Amount: "10.5", Address: "0x1", Reference: "payout-001",
			RecipientRouting: &WithdrawRecipientRouting{Type: "memo_id", Value: "10012345"}},
		{ID: "as2", Amount: "25.0", Address: "0x2"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/wallets/w1/addresses/a1/withdraw" {
		t.Fatalf("path = %s", got.path)
	}
	if _, ok := got.body["assetId"]; ok {
		t.Fatalf("batch body has assetId: %v", got.body)
	}
	assets, _ := got.body["assets"].([]any)
	if len(assets) != 2 {
		t.Fatalf("assets = %v", got.body["assets"])
	}
	a0 := assets[0].(map[string]any)
	rr, _ := a0["recipientRouting"].(map[string]any)
	if a0["id"] != "as1" || a0["amount"] != "10.5" || a0["address"] != "0x1" || rr["type"] != "memo_id" || rr["value"] != "10012345" {
		t.Fatalf("assets[0] = %v", a0)
	}
	if len(res.Success) != 2 || res.Success[1].ID != "tx-uuid-2" || res.Success[1].Amount != "25.0" || res.Success[0].Status != "PENDING" {
		t.Fatalf("success = %+v", res.Success)
	}
}

func TestWithdrawSign(t *testing.T) {
	c, _ := server(t, 200, withdrawSignFixture)
	res, err := c.Withdraw.AddressSign(context.Background(), "w1", "a1", WithdrawRequest{AssetID: "as1", Address: "0x1", Amount: "0.005"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Type != "SIGNED" || res.SignedTransaction[:6] != "0xf8a9" || res.TokenAddress != "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd" {
		t.Fatalf("res = %+v", res)
	}
	if res.AMLScreening == nil || res.AMLScreening.Status != "success" {
		t.Fatalf("aml = %+v", res.AMLScreening)
	}

	c, _ = server(t, 200, withdrawSignBatchFixture)
	res, err = c.Withdraw.Sign(context.Background(), "w1", WithdrawRequest{Assets: []WithdrawAsset{{ID: "a", Amount: "1", Address: "x"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Success) != 2 || res.Success[1].SignedTransaction != "0xf86d..." || res.Success[0].Hash != "0xsignedhash1..." {
		t.Fatalf("success = %+v", res.Success)
	}
}

func TestWithdrawNetworkFee(t *testing.T) {
	c, _ := server(t, 200, withdrawFeeBatchFixture)
	fee, err := c.Withdraw.NetworkFee(context.Background(), "w1", WithdrawRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(fee.Fees) != 2 || fee.Fees[1].AssetID != "98582300-2f1d-4ef4-99f2-4361319fc776" || fee.Fees[1].Index != 1 ||
		fee.TotalNetworkFee != "0.00001751904" || fee.EstimatedArrivalTime != 60 || fee.NativeBalance != "0.073690520542044578" {
		t.Fatalf("fee = %+v", fee)
	}

	c, _ = server(t, 200, withdrawFeeSingleFixture)
	fee, err = c.Withdraw.AddressNetworkFee(context.Background(), "w1", "a1", WithdrawRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if fee.NetworkFee != "0.000000021112014" || fee.NativeBalance != "0.04998428208321722" {
		t.Fatalf("fee = %+v", fee)
	}
}

// Documented response examples.
const (
	withdrawSingleFixture    = `{"data": {"amlScreening": null, "amount": "10", "amountPaid": "10", "asset": {"address": "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238", "blockchain": {"createdAt": "2024-05-14T11:53:33.095Z", "derivationPath": "m/44'/60'/0'/0", "id": "85ffc132-3972-4c9e-99a5-5cf0ccb688bf", "isActive": true, "isEvmCompatible": true, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1716800081/crypto-assets/ethereum-eth-logo_idraq2.png", "name": "ethereum", "slug": "ethereum", "symbol": "eth", "tokenStandard": "ERC20", "updatedAt": "2024-06-14T22:32:11.983Z"}, "createdAt": "2024-05-14T11:53:33.682Z", "decimals": 6, "id": "fe04a28c-c615-4e41-8eda-f84c862864f5", "isActive": true, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1716800083/crypto-assets/usd-coin-usdc-logo_fs9mhv.png", "name": "USDC Coin", "network": "testnet", "standard": "ERC20", "symbol": "USDC", "updatedAt": "2024-06-14T22:32:12.589Z"}, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptResponse": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"createdAt": "2024-05-14T11:53:33.095Z", "derivationPath": "m/44'/60'/0'/0", "id": "85ffc132-3972-4c9e-99a5-5cf0ccb688bf", "isActive": true, "isEvmCompatible": true, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1716800081/crypto-assets/ethereum-eth-logo_idraq2.png", "name": "ethereum", "slug": "ethereum", "symbol": "eth", "tokenStandard": "ERC20", "updatedAt": "2024-06-14T22:32:11.983Z"}, "chainId": null, "confirmations": null, "confirmed": false, "createdAt": "2024-10-23T11:33:58.765Z", "currency": "USD", "fee": null, "feeMetadata": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": "0xded90bc7f3d98f5ff0c3a97f711717192e83e01d89ac1ff4483ae6fd9d229e6d", "id": "081d6315-159f-4c38-b02a-c4708836c5bd", "metadata": null, "network": "testnet", "note": null, "reason": null, "recipientAddress": "0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22", "senderAddress": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "status": "PENDING", "tokenAddress": null, "type": "WITHDRAW", "updatedAt": "2024-10-23T11:33:58.765Z", "wallet": {"id": "d236a191-c1d4-423c-a439-54ce6542ca41"}}, "message": "Withdrawal request initiated successfully", "statusCode": 200}`
	withdrawBatchFixture     = `{"data": {"errors": [], "success": [{"amount": "10.5", "id": "tx-uuid-1", "recipientAddress": "0xRecipient1Address", "reference": "payout-001", "senderAddress": "0xWalletAddress", "status": "PENDING"}, {"amount": "25.0", "id": "tx-uuid-2", "recipientAddress": "0xRecipient2Address", "reference": "payout-002", "senderAddress": "0xWalletAddress", "status": "PENDING"}]}, "message": "Batch withdrawal initiated (2/2 queued successfully)", "statusCode": 200}`
	withdrawSignFixture      = `{"data": {"amlScreening": {"message": "Address is not sanctioned", "provider": "ofac, fbi, tether, circle", "status": "success"}, "amount": "0.005", "amountPaid": "0.005", "amountUSD": "0.005", "asset": {"address": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "createdAt": "2025-08-01T12:36:47.206Z", "currency": "USD", "decimals": 18, "id": "80f8ba40-d2c3-487c-bb37-e4c041492e0b", "isActive": true, "isNative": false, "name": "Tether USD", "network": "testnet", "standard": "BEP20", "symbol": "USDT", "updatedAt": "2025-08-01T12:36:47.206Z"}, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"createdAt": "2025-08-01T12:36:47.166Z", "derivationPath": "m/44'/60'/0'/0", "id": "3953d0bc-4d9c-4dab-9814-bc204ea86028", "isActive": true, "isEvmCompatible": true, "isL2": false, "name": "BNB smart chain", "slug": "bnb-smart-chain", "symbol": "bnb", "tokenStandard": "BEP20", "updatedAt": "2025-08-01T12:36:47.166Z"}, "chainId": null, "confirmations": null, "confirmed": true, "createdAt": "2025-08-06T11:20:15.385Z", "currency": "USD", "fee": null, "feeHash": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": "0x9df88df3ef381c449b7d87343885be552f51d58a1e36e8721f8290171c41324a", "id": "110a10b2-6fea-4e70-aecb-49c7306f5d41", "metadata": null, "network": "testnet", "note": null, "rate": null, "rateUSD": "1", "reason": null, "recipientAddress": "0xDBAcA914f11A8882D5917e2cDD9Dbcbd6f974406", "reference": "egfq3zz8S5", "senderAddress": "0x1345c2f58e22Eb85d5B85000F7FD8c8c8a301a01", "signedTransaction": "0xf8a9038407270e0082882e94337610d27c682e347c9cd60bd4b3b107c9d34ddd80b844a9059cbb000000000000000000000000dbaca914f11a8882d5917e2cdd9dbcbd6f9744060000000000000000000000000000000000000000000000000011c37937e0800081e6a0e29c45e1de02e6fb4336afc82b8537e035ccafd2669a75992d42ef185ae08216a07febc9ff3883c560ae5d5b3cda3aa570673ad8010efc66a847c994b6f65230ea", "status": "SUCCESS", "toAmount": null, "toCurrency": null, "tokenAddress": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "type": "SIGNED", "updatedAt": "2025-08-06T11:20:15.385Z", "wallet": {"address": "0x48197e643FfE94C045E96268c3d749eFd5Cf62d6", "configurations": {"autoSweeping": {"isActive": false, "threshold": 0}, "withdrawal": {"gasless": {"isActive": false, "operator": "gt", "threshold": 0}}}, "createdAt": "2025-08-01T17:25:56.817Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "bnb", "id": "acff0ed8-0592-4942-848a-6d0a3559c279", "isActive": true, "name": "BNB Wallet", "network": "testnet", "status": "ACTIVE", "updatedAt": "2025-08-04T15:24:38.130Z"}}, "message": "Withdrawal request signed", "statusCode": 200}`
	withdrawSignBatchFixture = `{"data": {"errors": [], "success": [{"hash": "0xsignedhash1...", "id": "tx-uuid-1", "signedTransaction": "0xf86c..."}, {"hash": "0xsignedhash2...", "id": "tx-uuid-2", "signedTransaction": "0xf86d..."}]}, "message": "Batch withdrawal signed (2/2 successful)", "statusCode": 200}`
	withdrawFeeBatchFixture  = `{"data": {"errors": [], "estimatedArrivalTime": 60, "fees": [{"address": "0x0000000000000000000000000000000000000002", "amount": "0.1", "assetId": "ae455f23-3824-4125-baab-d158315cbcbd", "estimatedArrivalTime": 30, "index": 0, "networkFee": "0.00001247904", "transactionFee": "0"}, {"address": "0x0000000000000000000000000000000000000002", "amount": "0.002", "assetId": "98582300-2f1d-4ef4-99f2-4361319fc776", "estimatedArrivalTime": 30, "index": 1, "networkFee": "0.00000504", "transactionFee": "0"}], "nativeBalance": "0.073690520542044578", "nativeBalanceInUSD": "66.54475076508251527134", "totalNetworkFee": "0.00001751904", "totalNetworkFeeInUSD": "0.0158202186912", "totalTransactionFee": "0"}, "message": "Network fee fetched successfully", "statusCode": 200}`
	withdrawFeeSingleFixture = `{"data": {"nativeBalance": "0.04998428208321722", "nativeBalanceInUSD": "4950.00", "networkFee": "0.000000021112014"}, "message": "Network fee fetched successfully", "statusCode": 200}`
)
