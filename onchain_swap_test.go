package blockradar

import (
	"context"
	"testing"
)

func TestSwapRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx, r := context.Background(), SwapRequest{}
	_, _ = c.Swap.Quote(ctx, "W", r)
	_, _ = c.Swap.Execute(ctx, "W", r)
	_, _ = c.Swap.AddressQuote(ctx, "W", "A", r)
	_, _ = c.Swap.AddressExecute(ctx, "W", "A", r)
	assertRoutes(t, hits(), []string{
		"POST /v1/wallets/W/swaps/quote",
		"POST /v1/wallets/W/swaps/execute",
		"POST /v1/wallets/W/addresses/A/swaps/quote",
		"POST /v1/wallets/W/addresses/A/swaps/execute",
	})
}

func TestSwapQuote(t *testing.T) {
	c, got := server(t, 200, `{"data": {"amount": "999.208698", "fromAmount": "1000", "toAmount": "999.208698", "requestedAmount": "1000", "amountSide": "source", "recipientRouting": {"status": "none", "message": "No recipient routing value will be used. Confirm the destination accepts direct deposits without one."}, "estimatedArrivalTime": 30, "impact": "-0.08", "impactInUSD": "-0.791264", "minAmount": "989.208", "nativeBalance": "0.000857699568686543", "nativeBalanceInUSD": "1.56912705292928295678", "networkFee": "0.000090678936279694", "networkFeeInUSD": "0.16589348676624898524", "rate": "0.999208698", "slippage": "1.00", "transactionFee": 0}, "message": "Swap quote fetched successfully", "statusCode": 200}`)
	q, err := c.Swap.Quote(context.Background(), "w1", SwapRequest{FromAssetID: "f", ToAssetID: "t", Amount: "1000"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.body) != 3 || got.body["fromAssetId"] != "f" || got.body["toAssetId"] != "t" || got.body["amount"] != "1000" {
		t.Fatalf("body = %v", got.body)
	}
	if q.ToAmount != "999.208698" || q.TransactionFee != "0" || q.EstimatedArrivalTime != 30 || q.Slippage != "1.00" {
		t.Fatalf("quote = %+v", q)
	}
	if q.RecipientRouting == nil || q.RecipientRouting.Status != "none" {
		t.Fatalf("routing = %+v", q.RecipientRouting)
	}
}

func TestSwapExecute(t *testing.T) {
	c, got := server(t, 200, swapExecuteFixture)
	tx, err := c.Swap.AddressExecute(context.Background(), "w1", "a1", SwapRequest{
		FromAssetID: "f", ToAssetID: "t", Amount: "0.5", AmountSide: AmountSideTarget, Order: "FASTEST",
		RecipientAddress: "GABC", RecipientRouting: &WithdrawRecipientRouting{Type: "none"},
		Reference: "ref", Metadata: map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/wallets/w1/addresses/a1/swaps/execute" || got.body["amountSide"] != "target" || got.body["order"] != "FASTEST" || got.body["reference"] != "ref" {
		t.Fatalf("got %s %v", got.path, got.body)
	}
	if rr, _ := got.body["recipientRouting"].(map[string]any); len(rr) != 1 || rr["type"] != "none" {
		t.Fatalf("recipientRouting = %v", got.body["recipientRouting"])
	}
	if tx.Type != "SWAP" || tx.Rate != "0.848566" || tx.ToAmount != "0.424283" || tx.Hash != "" {
		t.Fatalf("tx = %+v", tx)
	}
	if tx.ToAsset == nil || tx.ToAsset.ID != "ee7318c8-3a61-4437-88a0-de56b97f2a89" || tx.ToBlockchain == nil || tx.ToBlockchain.Slug != "optimism" ||
		tx.ToWallet == nil || tx.ToWallet.Name != "Optimism Wallet" || tx.Asset == nil || tx.Asset.Symbol != "USDC" {
		t.Fatalf("to = %+v %+v %+v", tx.ToAsset, tx.ToBlockchain, tx.ToWallet)
	}
}

// Documented response example (wallet.business and wallet.apiCredential removed).
const swapExecuteFixture = `{"data": {"addressId": null, "amlScreening": null, "amount": "0.5", "amountPaid": null, "asset": {"address": "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", "blockchain": {"createdAt": "2024-05-14T17:53:33.106Z", "derivationPath": "m/44'/60'/0'/0", "id": "b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa", "isActive": true, "isEvmCompatible": true, "isL2": false, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png", "name": "BNB smart chain", "slug": "bnb-smart-chain", "symbol": "bnb", "tokenStandard": "BEP20", "updatedAt": "2024-11-26T19:04:13.941Z"}, "createdAt": "2024-05-14T17:53:33.672Z", "decimals": 18, "id": "f31328ee-2db9-43ab-81ea-9a91db72957a", "isActive": true, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1716800083/crypto-assets/usd-coin-usdc-logo_fs9mhv.png", "name": "USD Coin", "network": "mainnet", "standard": "BEP20", "symbol": "USDC", "updatedAt": "2024-06-15T04:36:43.855Z"}, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptResponse": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"createdAt": "2024-05-14T17:53:33.106Z", "derivationPath": "m/44'/60'/0'/0", "id": "b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa", "isActive": true, "isEvmCompatible": true, "isL2": false, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png", "name": "BNB smart chain", "slug": "bnb-smart-chain", "symbol": "bnb", "tokenStandard": "BEP20", "updatedAt": "2024-11-26T19:04:13.941Z"}, "chainId": null, "confirmations": null, "confirmed": false, "createdAt": "2025-04-01T15:00:28.604Z", "currency": null, "fee": null, "feeMetadata": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": null, "id": "59b8f26e-0a5d-4679-8686-f5032e4344d2", "metadata": null, "network": "mainnet", "note": null, "rate": "0.848566", "reason": null, "recipientAddress": "0xD2b6be31932E0294F2ebD14a008C3f1E05B47BC4", "senderAddress": "0xD2b6be31932E0294F2ebD14a008C3f1E05B47BC4", "status": "PENDING", "toAmount": "0.424283", "toAsset": {"address": "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85", "blockchain": {"createdAt": "2024-10-07T22:09:49.784Z", "derivationPath": "m/44'/60'/0'/0", "id": "1c9ca9df-325f-4b60-a7ec-b28d2235e0b7", "isActive": true, "isEvmCompatible": true, "isL2": true, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1728312943/crypto-assets/optimism-ethereum-op-logo_h62d1i.png", "name": "optimism", "slug": "optimism", "symbol": "eth", "tokenStandard": null, "updatedAt": "2024-11-26T19:04:13.947Z"}, "createdAt": "2024-10-08T19:12:41.930Z", "decimals": 6, "id": "ee7318c8-3a61-4437-88a0-de56b97f2a89", "isActive": true, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1716800083/crypto-assets/usd-coin-usdc-logo_fs9mhv.png", "name": "USD Coin", "network": "mainnet", "standard": null, "symbol": "USDC", "updatedAt": "2024-10-08T19:12:41.930Z"}, "toBlockchain": {"createdAt": "2024-10-07T22:09:49.784Z", "derivationPath": "m/44'/60'/0'/0", "id": "1c9ca9df-325f-4b60-a7ec-b28d2235e0b7", "isActive": true, "isEvmCompatible": true, "isL2": true, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1728312943/crypto-assets/optimism-ethereum-op-logo_h62d1i.png", "name": "optimism", "slug": "optimism", "symbol": "eth", "tokenStandard": null, "updatedAt": "2024-11-26T19:04:13.947Z"}, "toWallet": {"address": "0xD2b6be31932E0294F2ebD14a008C3f1E05B47BC4", "configurations": null, "createdAt": "2024-10-08T19:15:28.390Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "This is Optimism Wallet", "id": "f25dfef5-253a-4cb8-ab00-80f2e799f780", "isActive": true, "name": "Optimism Wallet", "network": "mainnet", "status": "ACTIVE", "updatedAt": "2024-10-08T19:15:28.390Z"}, "tokenAddress": null, "type": "SWAP", "updatedAt": "2025-04-01T15:00:28.604Z", "wallet": {"address": "0xD2b6be31932E0294F2ebD14a008C3f1E05B47BC4", "blockchain": {"createdAt": "2024-05-14T17:53:33.106Z", "derivationPath": "m/44'/60'/0'/0", "id": "b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa", "isActive": true, "isEvmCompatible": true, "isL2": false, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png", "name": "BNB smart chain", "slug": "bnb-smart-chain", "symbol": "bnb", "tokenStandard": "BEP20", "updatedAt": "2024-11-26T19:04:13.941Z"}, "configurations": null, "createdAt": "2024-08-24T19:49:15.617Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "This is BNB Smart Chain Mainnet Wallet", "id": "23628012-2b69-4ed7-9627-63fb3f547381", "isActive": true, "name": "BNB Smart Chain Mainnet Wallet", "network": "mainnet", "status": "ACTIVE", "updatedAt": "2024-08-24T19:49:15.617Z"}}, "message": "Swap transaction created successfully", "statusCode": 200}`
