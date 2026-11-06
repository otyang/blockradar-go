package blockradar

import (
	"context"
	"testing"
)

func TestContractsRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx, r := context.Background(), ContractRequest{}
	_, _ = c.Contracts.Read(ctx, "W", r)
	_, _ = c.Contracts.Write(ctx, "W", r)
	_, _ = c.Contracts.WriteSign(ctx, "W", r)
	_, _ = c.Contracts.NetworkFee(ctx, "W", r)
	_, _ = c.Contracts.AddressRead(ctx, "W", "A", r)
	_, _ = c.Contracts.AddressWrite(ctx, "W", "A", r)
	_, _ = c.Contracts.AddressWriteSign(ctx, "W", "A", r)
	_, _ = c.Contracts.AddressNetworkFee(ctx, "W", "A", r)
	assertRoutes(t, hits(), []string{
		"POST /v1/wallets/W/contracts/read",
		"POST /v1/wallets/W/contracts/write",
		"POST /v1/wallets/W/contracts/write/sign",
		"POST /v1/wallets/W/contracts/network-fee",
		"POST /v1/wallets/W/addresses/A/contracts/read",
		"POST /v1/wallets/W/addresses/A/contracts/write",
		"POST /v1/wallets/W/addresses/A/contracts/write/sign",
		"POST /v1/wallets/W/addresses/A/contracts/network-fee", // docs omit /v1 here
	})
}

var transferABI = []any{map[string]any{
	"name": "transfer", "type": "function", "stateMutability": "nonpayable",
	"inputs":  []any{map[string]any{"name": "_to", "type": "address"}, map[string]any{"name": "_value", "type": "uint256"}},
	"outputs": []any{},
}}

func TestContractWriteSingleBody(t *testing.T) {
	c, got := server(t, 200, contractWriteFixture)
	res, err := c.Contracts.Write(context.Background(), "w1", ContractRequest{ContractCall: &ContractCall{
		Address: "0x3376", Method: "transfer", Parameters: []any{"0x2455", "10000"}, ABI: transferABI, Reference: "r1",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/wallets/w1/contracts/write" || got.body["address"] != "0x3376" || got.body["method"] != "transfer" || got.body["reference"] != "r1" {
		t.Fatalf("got %s %v", got.path, got.body)
	}
	if p, _ := got.body["parameters"].([]any); len(p) != 2 || p[1] != "10000" {
		t.Fatalf("parameters = %v", got.body["parameters"])
	}
	if abi, _ := got.body["abi"].([]any); len(abi) != 1 {
		t.Fatalf("abi = %v", got.body["abi"])
	}
	for _, k := range []string{"calls", "code", "metadata"} {
		if _, ok := got.body[k]; ok {
			t.Fatalf("unexpected %q in %v", k, got.body)
		}
	}
	if res.ID != "42e94af1-1b8c-47fd-9250-276a3c8a3a71" || res.Type != "CUSTOM_SMART_CONTRACT" || res.TokenAddress != "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd" || res.Wallet == nil || len(res.Success) != 0 {
		t.Fatalf("res = %+v", res)
	}
}

func TestContractWriteBatch(t *testing.T) {
	c, got := server(t, 200, contractWriteBatchFixture)
	cc := ContractCall{Address: "0x3376", Method: "transfer", Parameters: []any{"0x2455", "10000"}, ABI: transferABI}
	res, err := c.Contracts.AddressWrite(context.Background(), "w1", "a1", ContractRequest{Calls: []ContractCall{cc, cc}, Code: "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if got.body["code"] != "123456" || len(got.body["calls"].([]any)) != 2 {
		t.Fatalf("body = %v", got.body)
	}
	if _, ok := got.body["address"]; ok {
		t.Fatalf("batch body has top-level address: %v", got.body)
	}
	if len(res.Success) != 2 || res.Success[1].ID != "402dfdee-0282-40c9-ba76-6f35601c8f23" || res.Success[0].Status != "PENDING" || res.Success[1].Metadata["batchIndex"] != float64(1) || len(res.Errors) != 0 {
		t.Fatalf("res = %+v", res)
	}
}

func TestContractWriteSign(t *testing.T) {
	c, _ := server(t, 200, contractSignFixture)
	res, err := c.Contracts.WriteSign(context.Background(), "w1", ContractRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Type != "SIGNED" || res.SignedTransaction[:6] != "0xf8ab" || res.Reference != "xvLz7CIFoD" {
		t.Fatalf("res = %+v", res)
	}
}

func TestContractRead(t *testing.T) {
	c, _ := server(t, 200, `{"data": "10052335235393043", "message": "Contract read successfully", "statusCode": 200}`)
	res, err := c.Contracts.Read(context.Background(), "w1", ContractRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Value) != `"10052335235393043"` || res.Success != nil {
		t.Fatalf("res = %+v", res)
	}

	c, _ = server(t, 200, `{"data": {"errors": [], "success": [{"data": "16132019479847722328", "index": 0, "method": "balanceOf"}, {"data": "999999999556", "index": 1, "method": "balanceOf"}]}, "message": "Batch contract read completed (2/2 successful)", "statusCode": 200}`)
	res, err = c.Contracts.AddressRead(context.Background(), "w1", "a1", ContractRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Value != nil || len(res.Success) != 2 || res.Success[1].Index != 1 || string(res.Success[1].Data) != `"999999999556"` {
		t.Fatalf("res = %+v", res)
	}
}

func TestContractNetworkFee(t *testing.T) {
	c, _ := server(t, 200, `{"data": {"balance": "1.012299726305777896", "fee": "0.00422709309982132"}, "message": "Network fee retrieved successfully", "statusCode": 200}`)
	fee, err := c.Contracts.NetworkFee(context.Background(), "w1", ContractRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if fee.Balance != "1.012299726305777896" || fee.Fee != "0.00422709309982132" {
		t.Fatalf("fee = %+v", fee)
	}

	c, _ = server(t, 200, `{"data": {"errors": [], "estimatedArrivalTime": 60, "fees": [{"balance": "0.073661808182044578", "estimatedArrivalTime": 30, "fee": "0.00000835248", "index": 0, "method": "transfer", "nativeBalance": "0.073661808182044578", "nativeBalanceInUSD": "66.51882264263171527134", "networkFee": "0.00000835248", "networkFeeInUSD": "0.0075425400144", "transactionFee": "0"}, {"balance": "0.073661808182044578", "estimatedArrivalTime": 30, "fee": "0.00000835248", "index": 1, "method": "transfer", "nativeBalance": "0.073661808182044578", "nativeBalanceInUSD": "66.51882264263171527134", "networkFee": "0.00000835248", "networkFeeInUSD": "0.0075425400144", "transactionFee": "0"}], "totalFee": "0.00001670496"}, "message": "Batch network fee retrieved (2/2 successful)", "statusCode": 200}`)
	fee, err = c.Contracts.AddressNetworkFee(context.Background(), "w1", "a1", ContractRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(fee.Fees) != 2 || fee.Fees[1].Method != "transfer" || fee.Fees[1].NetworkFeeInUSD != "0.0075425400144" || fee.TotalFee != "0.00001670496" {
		t.Fatalf("fee = %+v", fee)
	}
}

// Documented response examples.
const (
	contractWriteFixture      = `{"data": {"amlScreening": null, "amountPaid": null, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptResponse": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"createdAt": "2024-05-14T17:53:33.106Z", "derivationPath": "m/44'/60'/0'/0", "id": "b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa", "isActive": true, "isEvmCompatible": true, "isL2": false, "logoUrl": "https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/bnb-bnb-logo_e4qdyk.png", "name": "BNB smart chain", "slug": "bnb-smart-chain", "symbol": "bnb", "tokenStandard": "BEP20", "updatedAt": "2024-11-26T19:04:13.941Z"}, "chainId": null, "confirmations": null, "confirmed": false, "createdAt": "2025-01-15T21:17:24.013Z", "currency": null, "fee": null, "feeMetadata": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": null, "id": "42e94af1-1b8c-47fd-9250-276a3c8a3a71", "metadata": null, "network": "testnet", "note": null, "reason": "{\"method\":\"transfer\",\"parameters\":[\"0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22\",\"100000\"],\"abi\":{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}}", "recipientAddress": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "response": null, "senderAddress": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "status": "PENDING", "tokenAddress": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "type": "CUSTOM_SMART_CONTRACT", "updatedAt": "2025-01-15T21:17:24.013Z", "wallet": {"address": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "business": {"id": "4b96c271-35eb-45e8-b558-6a53f95df601", "name": "Test One Inc"}, "createdAt": "2024-08-22T15:29:11.387Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "This is BNB smart chain testnet master wallet", "id": "4465468a-3c36-4536-918a-91d689e18a74", "isActive": true, "name": "BNB smart chain Master Wallet", "network": "testnet", "status": "ACTIVE", "updatedAt": "2024-10-30T13:11:56.221Z"}}, "message": "Contract write initiated successfully", "statusCode": 200}`
	contractWriteBatchFixture = `{"data": {"errors": [], "success": [{"amlScreening": {}, "amount": null, "amountPaid": null, "amountUSD": "0.00", "asset": null, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"configurations": null, "createdAt": "2024-05-14T17:53:33.106Z", "derivationPath": "m/44'/60'/0'/0", "id": "b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa", "isActive": true, "isEvmCompatible": true, "isL2": false, "name": "BNB smart chain", "networkFeeModel": "native", "slug": "bnb-smart-chain", "symbol": "bnb", "tokenStandard": "BEP20", "updatedAt": "2024-11-26T20:04:13.941Z"}, "chainId": null, "confirmations": null, "confirmed": false, "createdAt": "2026-01-11T23:29:12.163Z", "createdChannel": "dashboard", "currency": "USD", "fee": null, "feeHash": null, "feeUSD": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": null, "id": "b16a014f-da18-435a-9639-3359e0e02fd6", "isAutoSweep": false, "metadata": {"batchIndex": 0, "batchTotal": 2}, "network": "testnet", "note": "Method: transfer | Contract Address: 0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "rate": null, "rateUSD": "0.00", "reason": "{\"method\":\"transfer\",\"parameters\":[\"0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22\",\"10000\"],\"abi\":{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}}", "recipientAddress": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "reference": "QTfxp7WlKWB1CTohQoO", "senderAddress": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "signedTransaction": null, "status": "PENDING", "toAmount": null, "toCurrency": null, "tokenAddress": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "type": "CUSTOM_SMART_CONTRACT", "updatedAt": "2026-01-11T23:29:12.163Z", "wallet": {"address": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "configurations": {"addresses": {"isActive": true, "prefunding": {"isActive": true, "rules": []}}, "autoSettlement": {"isActive": true, "rules": []}, "autoSweeping": {"isActive": true}, "withdrawal": {"gasless": {"isActive": true}}}, "createdAt": "2024-08-22T15:29:11.387Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "This is BNB smart chain testnet master wallet", "id": "4465468a-3c36-4536-918a-91d689e18a74", "isActive": true, "name": "BNB Smart Chain", "network": "testnet", "status": "ACTIVE", "updatedAt": "2025-09-13T02:02:44.110Z"}}, {"amlScreening": {}, "amount": null, "amountPaid": null, "amountUSD": "0.00", "asset": null, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"configurations": null, "createdAt": "2024-05-14T17:53:33.106Z", "derivationPath": "m/44'/60'/0'/0", "id": "b80d3d5e-16f1-4d99-be5e-6dfcd27f89aa", "isActive": true, "isEvmCompatible": true, "isL2": false, "name": "BNB smart chain", "networkFeeModel": "native", "slug": "bnb-smart-chain", "symbol": "bnb", "tokenStandard": "BEP20", "updatedAt": "2024-11-26T20:04:13.941Z"}, "chainId": null, "confirmations": null, "confirmed": false, "createdAt": "2026-01-11T23:29:12.526Z", "createdChannel": "dashboard", "currency": "USD", "fee": null, "feeHash": null, "feeUSD": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": null, "id": "402dfdee-0282-40c9-ba76-6f35601c8f23", "isAutoSweep": false, "metadata": {"batchIndex": 1, "batchTotal": 2}, "network": "testnet", "note": "Method: transfer | Contract Address: 0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "rate": null, "rateUSD": "0.00", "reason": "{\"method\":\"transfer\",\"parameters\":[\"0x2455eC6700092991Ce0782365A89d5Cd89c8Fa22\",\"10000\"],\"abi\":{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}}", "recipientAddress": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "reference": "Kob7O3WokhVwFuhU1Iq", "senderAddress": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "signedTransaction": null, "status": "PENDING", "toAmount": null, "toCurrency": null, "tokenAddress": "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd", "type": "CUSTOM_SMART_CONTRACT", "updatedAt": "2026-01-11T23:29:12.526Z", "wallet": {"address": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "configurations": {"addresses": {"isActive": true, "prefunding": {"isActive": true, "rules": []}}, "autoSettlement": {"isActive": true, "rules": []}, "autoSweeping": {"isActive": true}, "withdrawal": {"gasless": {"isActive": true}}}, "createdAt": "2024-08-22T15:29:11.387Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "This is BNB smart chain testnet master wallet", "id": "4465468a-3c36-4536-918a-91d689e18a74", "isActive": true, "name": "BNB Smart Chain", "network": "testnet", "status": "ACTIVE", "updatedAt": "2025-09-13T02:02:44.110Z"}}]}, "message": "Batch contract write initiated successfully", "statusCode": 200}`
	contractSignFixture       = `{"data": {"amlScreening": {}, "amount": null, "amountPaid": null, "amountUSD": "0.00", "asset": null, "assetSwept": null, "assetSweptAmount": null, "assetSweptAt": null, "assetSweptGasFee": null, "assetSweptHash": null, "assetSweptRecipientAddress": null, "assetSweptSenderAddress": null, "blockHash": null, "blockNumber": null, "blockchain": {"createdAt": "2024-05-27T14:31:14.966Z", "derivationPath": "m/44'/60'/0'/0", "id": "74733889-4ecd-403e-9840-94e87c043f24", "isActive": true, "isEvmCompatible": true, "isL2": true, "name": "base", "slug": "base", "symbol": "eth", "tokenStandard": null, "updatedAt": "2024-11-26T20:04:13.945Z"}, "chainId": null, "confirmations": null, "confirmed": true, "createdAt": "2025-08-11T22:21:56.077Z", "currency": "USD", "fee": null, "feeHash": null, "gasFee": null, "gasPrice": null, "gasUsed": null, "hash": "0x229d811ef34a9f12454e89f12e06945dfc8122538a0af6369ec3f2a2f9aa8cbb", "id": "fe2d8c39-a6ba-41cb-99e7-05c4dcee6c94", "metadata": null, "network": "testnet", "note": null, "rate": null, "rateUSD": "0.00", "reason": "{\"method\":\"transfer\",\"parameters\":[\"0x451dEFC27B45808078e875556AF06bCFdC697BA4\",\"10000\"],\"abi\":{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}}", "recipientAddress": "0x036CbD53842c5426634e7929541eC2318f3dCF7e", "reference": "xvLz7CIFoD", "senderAddress": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "signedTransaction": "0xf8ab3e83124fc08301623d94036cbd53842c5426634e7929541ec2318f3dcf7e80b844a9059cbb000000000000000000000000451defc27b45808078e875556af06bcfdc697ba400000000000000000000000000000000000000000000000000000000000027108302948ca0c86783ba50374a959907d9174c2a0b5cd3a87d55336e6c9a6420fde0dd07303ea019a93dad742a9bed4c6d1f514f7fe26777e8f17fe090967a9719afddaa281d64", "status": "SUCCESS", "toAmount": null, "toCurrency": null, "tokenAddress": "0x036CbD53842c5426634e7929541eC2318f3dCF7e", "type": "SIGNED", "updatedAt": "2025-08-11T22:21:56.077Z", "wallet": {"address": "0x947514e4B803e312C312da0F1B41fEDdbe15ae7a", "configurations": null, "createdAt": "2024-08-22T16:05:46.167Z", "derivationPath": "m/44'/60'/0'/0/0", "description": "This is Base wallet", "id": "e17dea25-6c01-4b3a-a4ae-51f411bf69cd", "isActive": true, "name": "Base Wallet", "network": "testnet", "status": "ACTIVE", "updatedAt": "2024-08-22T16:05:46.167Z"}}, "message": "Contract write signed successfully", "statusCode": 200}`
)
