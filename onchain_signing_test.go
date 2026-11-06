package blockradar

import (
	"context"
	"testing"
)

func TestSigningRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx := context.Background()
	_, _ = c.Signing.Message(ctx, "W", SignMessageRequest{})
	_, _ = c.Signing.TypedData(ctx, "W", SignTypedDataRequest{})
	_, _ = c.Signing.Transaction(ctx, "W", SignTransactionRequest{})
	_, _ = c.Signing.Broadcast(ctx, "W", SignTransactionRequest{})
	_, _ = c.Signing.AddressMessage(ctx, "W", "A", SignMessageRequest{})
	_, _ = c.Signing.AddressTypedData(ctx, "W", "A", SignTypedDataRequest{})
	_, _ = c.Signing.AddressTransaction(ctx, "W", "A", SignTransactionRequest{})
	_, _ = c.Signing.AddressBroadcast(ctx, "W", "A", SignTransactionRequest{})
	assertRoutes(t, hits(), []string{
		"POST /v1/wallets/W/signing/message",
		"POST /v1/wallets/W/signing/typed-data",
		"POST /v1/wallets/W/signing/transaction",
		"POST /v1/wallets/W/signing/broadcast",
		"POST /v1/wallets/W/addresses/A/signing/message",
		"POST /v1/wallets/W/addresses/A/signing/typed-data",
		"POST /v1/wallets/W/addresses/A/signing/transaction",
		"POST /v1/wallets/W/addresses/A/signing/broadcast",
	})
}

func TestSignMessageDecodesSignature(t *testing.T) {
	c, got := server(t, 200, signMessageFixture)
	res, err := c.Signing.Message(context.Background(), "w1", SignMessageRequest{Message: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if got.body["message"] != "hello" || len(got.body) != 1 {
		t.Fatalf("body = %v", got.body)
	}
	if res.SignedTransaction.Raw != "" || res.SignedTransaction.Signature[:10] != "0xb4a07203" {
		t.Fatalf("signed = %+v", res.SignedTransaction)
	}
	if res.Type != "SIGNED" || res.CreatedChannel != "dashboard" || res.Blockchain == nil || res.Blockchain.Slug != "tron" {
		t.Fatalf("res = %+v", res)
	}
}

func TestSignTypedData(t *testing.T) {
	c, got := server(t, 200, signTypedDataFixture)
	res, err := c.Signing.AddressTypedData(context.Background(), "w1", "a1", SignTypedDataRequest{
		Domain:  SignTypedDataDomain{Name: "USD Coin", Version: "2", ChainID: "11155111", VerifyingContract: "0xa0b8"},
		Types:   map[string][]SignTypedDataField{"Permit": {{Name: "owner", Type: "address"}}},
		Message: map[string]any{"owner": "0x742d", "value": "1000000000"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/wallets/w1/addresses/a1/signing/typed-data" {
		t.Fatalf("path = %s", got.path)
	}
	dom, _ := got.body["domain"].(map[string]any)
	if dom["chainId"] != float64(11155111) || dom["name"] != "USD Coin" {
		t.Fatalf("domain = %v", dom)
	}
	if _, ok := dom["salt"]; ok {
		t.Fatalf("salt should be omitted: %v", dom)
	}
	permit, _ := got.body["types"].(map[string]any)["Permit"].([]any)
	if len(permit) != 1 || permit[0].(map[string]any)["type"] != "address" {
		t.Fatalf("types = %v", got.body["types"])
	}
	if res.SignedTransaction.V == 0 || res.SignedTransaction.R == "" || res.SignedTransaction.Signature == "" {
		t.Fatalf("signed = %+v", res.SignedTransaction)
	}
}

func TestSignTransactionDecodesRaw(t *testing.T) {
	c, got := server(t, 200, signTransactionFixture)
	res, err := c.Signing.Transaction(context.Background(), "w1", SignTransactionRequest{
		Transaction: `{"to":"0xdead"}`, Reference: "eth-transfer-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.body["transaction"] != `{"to":"0xdead"}` || got.body["reference"] != "eth-transfer-001" {
		t.Fatalf("body = %v", got.body)
	}
	if _, ok := got.body["metadata"]; ok {
		t.Fatalf("metadata should be omitted: %v", got.body)
	}
	if res.SignedTransaction.Raw[:10] != "0x02f87383" || res.ChainID.String() != "11155111" || res.Reference != "52TQawmiqYpNiWD2Ks0P" {
		t.Fatalf("res = %+v", res)
	}
}

// Documented response examples.
const (
	signMessageFixture     = `{"data": {"amlScreening": {}, "amount": null, "amountPaid": null, "amountUSD": "0.00", "asset": null, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"configurations": null, "createdAt": "2024-05-14T11:53:33.110Z", "derivationPath": "m/44'/195'/0'/0", "id": "fa91a922-3838-45f6-8a88-a4c771e1443a", "isActive": true, "isEvmCompatible": false, "isL2": false, "name": "tron", "networkFeeModel": "native", "slug": "tron", "symbol": "trx", "tokenStandard": "TRC20", "updatedAt": "2024-11-26T13:04:13.943Z"}, "chainId": null, "confirmations": null, "confirmed": true, "createdAt": "2026-03-02T11:11:35.399Z", "createdChannel": "dashboard", "currency": "USD", "fee": null, "feeHash": null, "feeUSD": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": "0xb4a07203d5ba162c99e67cd91cb329eea1edd8ebd837ce458f7de55152a00c710b076ecc7015b1dec9c87693a4f456631858b03d2cc0af53cf92203bc33e9d421c", "id": "df8811a7-7482-4f2c-ac7c-12298b6ab802", "isAutoSweep": false, "metadata": null, "network": "testnet", "note": null, "processingProviderReference": null, "processingReason": null, "processingStatus": "PENDING", "rate": null, "rateUSD": "0.00", "reason": "I am verifying ownership of the wallet address 0x... as customer cust_123. This message was signed on 2026-03-03 to confirm my control over this wallet.", "recipientAddress": "TF8LEkxpDVnS1u4aR48xSuekGGscvc6Uv7", "reference": "xZm6Xf7xWXlqCjp6eTd", "senderAddress": "TF8LEkxpDVnS1u4aR48xSuekGGscvc6Uv7", "signedTransaction": {"signature": "0xb4a07203d5ba162c99e67cd91cb329eea1edd8ebd837ce458f7de55152a00c710b076ecc7015b1dec9c87693a4f456631858b03d2cc0af53cf92203bc33e9d421c"}, "status": "SUCCESS", "toAmount": null, "toCurrency": null, "tokenAddress": null, "type": "SIGNED", "updatedAt": "2026-03-02T11:11:35.399Z", "wallet": {"address": "TF8LEkxpDVnS1u4aR48xSuekGGscvc6Uv7", "configurations": {"addresses": {"isActive": true, "prefunding": {"assets": [{"alwaysMaintain": true, "amount": "3.5", "blockchain": "tron", "createdAt": "2025-09-02T11:58:15.807Z", "currency": "USD", "id": "c4da3644-7980-484a-8477-5abe542d91a1", "isActive": true, "name": "Tron Network Fee Funding", "symbol": "TRX", "updatedAt": "2025-09-02T11:58:15.807Z"}], "isActive": true}}, "autoSettlement": {"isActive": false, "name": "Base Master wallet swap auto ettlement", "rules": [{"destination": {"address": "0x0077777d7EBA4688BDeF3E311b846F25870A19B9", "asset": "USDC", "blockchain": "base", "isGateway": true}, "id": "13d4d7e1-c95c-4a8a-a800-75894daf3673", "isActive": true, "name": "Swap from Base USDC to Gateway", "order": "FASTEST", "slippageTolerance": "5", "source": {"assets": ["USDC", "USDT"], "blockchain": "tron", "maxAmount": "1000", "minAmount": "1"}}]}}, "createdAt": "2024-08-22T10:01:41.993Z", "derivationPath": "m/44'/195'/0'/0/0", "description": "This is Tron testnet master wallet", "id": "eefe6470-3eec-4454-bc54-3eafe3bcbb87", "isActive": true, "name": "Tron Main Wallet", "network": "testnet", "status": "ACTIVE", "updatedAt": "2025-09-02T11:58:15.808Z"}}, "message": "Message signed successfully", "statusCode": 200}`
	signTypedDataFixture   = `{"data": {"amlScreening": {}, "amount": null, "amountPaid": null, "amountUSD": "0.00", "asset": null, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"createdAt": "2024-05-14T21:53:33.095Z", "derivationPath": "m/44'/60'/0'/0", "id": "85ffc132-3972-4c9e-99a5-5cf0ccb688bf", "isActive": true, "isEvmCompatible": true, "isL2": false, "name": "ethereum", "slug": "ethereum", "symbol": "eth", "tokenStandard": "ERC20", "updatedAt": "2024-11-26T23:04:13.936Z"}, "chainId": null, "confirmations": null, "confirmed": true, "createdAt": "2025-10-07T22:58:08.766Z", "createdChannel": "dashboard", "currency": "USD", "fee": null, "feeHash": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": "0xdb095e6cbf235d630cee43e0953e60c351e46897bc4e65abfce3e975810e21335aa3918399dac1e01badb2dc8c59c171e65d0c328c92737de702da9d76b889b31b", "id": "770f9100-7338-4823-b1ce-3658fc67db09", "metadata": null, "network": "testnet", "note": null, "rate": null, "rateUSD": "0.00", "reason": null, "recipientAddress": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "reference": "OznFZsh1SMlTBgUaEpM", "senderAddress": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "signedTransaction": {"r": "0xdb095e6cbf235d630cee43e0953e60c351e46897bc4e65abfce3e975810e2133", "s": "0x5aa3918399dac1e01badb2dc8c59c171e65d0c328c92737de702da9d76b889b3", "signature": "0xdb095e6cbf235d630cee43e0953e60c351e46897bc4e65abfce3e975810e21335aa3918399dac1e01badb2dc8c59c171e65d0c328c92737de702da9d76b889b31b", "v": 27}, "status": "SUCCESS", "toAmount": null, "toCurrency": null, "tokenAddress": null, "type": "SIGNED", "updatedAt": "2025-10-07T22:58:08.766Z", "wallet": {"address": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "configurations": {"addresses": {"isActive": true, "prefunding": {"isActive": true, "rules": []}}, "autoSettlement": {"isActive": true, "rules": [{"createdAt": "2025-09-10T14:41:12.126Z", "destination": {"address": "0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22", "asset": "USDC", "blockchain": "ethereum"}, "id": "fb841522-ac48-41f5-ab77-9744571b9c46", "isActive": true, "isGateway": false, "name": "Settle to an external address ", "order": "RECOMMENDED", "slippageTolerance": "5", "source": {"assets": ["USDC"], "blockchain": "ethereum", "maxAmount": "10000", "minAmount": "1"}, "updatedAt": "2025-09-10T14:41:12.126Z"}]}}, "createdAt": "2024-08-22T19:48:56.322Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "This is ethereum testnet master wallet", "id": "d236a191-c1d4-423c-a439-54ce6542ca41", "isActive": true, "name": "Ethereum Master Wallet", "network": "testnet", "status": "ACTIVE", "updatedAt": "2025-09-15T23:06:08.009Z"}}, "message": "Typed data signed successfully", "statusCode": 200}`
	signTransactionFixture = `{"statusCode": 200, "message": "Transaction signed successfully", "data": {"id": "782942da-48b0-416b-924b-8f657ae637a7", "reference": "52TQawmiqYpNiWD2Ks0P", "senderAddress": "0xC887a3Cb8E7AbA4A77D7275AD4B242f71cbd5446", "recipientAddress": "0xC887a3Cb8E7AbA4A77D7275AD4B242f71cbd5446", "tokenAddress": null, "amount": null, "amountPaid": null, "amountUSD": "0.00", "rateUSD": "0.00", "fee": null, "feeUSD": null, "feeHash": null, "currency": "USD", "toCurrency": null, "blockNumber": null, "blockHash": null, "hash": "0xac4c73ca084608ac6b981e54db948dc80c15b4ea3ffd0c9f5781f3af7ad6fe51", "confirmations": null, "confirmed": true, "gasPrice": null, "gasUsed": null, "gasFee": null, "status": "SUCCESS", "processingStatus": "PENDING", "processingProviderReference": null, "processingReason": null, "type": "SIGNED", "createdChannel": "api", "note": null, "amlScreening": {}, "isAutoSweep": false, "assetSwept": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptSenderAddress": null, "assetSweptRecipientAddress": null, "assetSweptAmount": null, "reason": "Transaction signed", "network": "testnet", "chainId": 11155111, "metadata": null, "toAmount": null, "signedTransaction": "0x02f87383aa36a763843b9aca008459682f0082520894000000000000000000000000000000000000dead865af3107a400080c080a0b7600de5fce50c7dddea4673296e48d440aa4d62148d9645c881494ff55c76e6a044576669a19e0e6cddc14115144ce3e5b6d80c71201ef0e46b28a22aca8d7d2a", "rate": null, "createdAt": "2026-03-19T13:34:24.937Z", "updatedAt": "2026-03-19T13:34:24.937Z", "wallet": {"id": "3f9aca5c-38ee-4e1d-ab67-c084a2e37bb2", "name": "Ethereum Wallet", "description": "Yooooo", "address": "0xC887a3Cb8E7AbA4A77D7275AD4B242f71cbd5446", "derivationPath": "m/44'/60'/0'/0/0", "isActive": true, "status": "ACTIVE", "network": "testnet", "configurations": {}, "createdAt": "2024-05-15T12:09:24.777Z", "updatedAt": "2026-03-18T15:30:13.505Z"}, "asset": null, "blockchain": {"id": "85ffc132-3972-4c9e-99a5-5cf0ccb688bf", "name": "ethereum", "symbol": "eth", "slug": "ethereum", "derivationPath": "m/44'/60'/0'/0", "isEvmCompatible": true, "isL2": false, "isActive": true, "tokenStandard": "ERC20", "networkFeeModel": "native", "configurations": null, "createdAt": "2024-05-14T17:53:33.095Z", "updatedAt": "2024-11-26T20:04:13.936Z"}, "beneficiary": null}}`
)
