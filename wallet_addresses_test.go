package blockradar

import (
	"context"
	"testing"
)

func TestAddressesRoutes(t *testing.T) {
	c, hits := recordRoutes(t, "")
	ctx, s := context.Background(), c.Addresses
	calls := []func() error{
		func() error { _, err := s.Generate(ctx, "W", AddressGenerateRequest{}); return err },
		func() error { _, err := s.List(ctx, "W", PageParams{}); return err },
		func() error { _, err := s.Get(ctx, "W", "A", false); return err },
		func() error { _, err := s.Update(ctx, "W", "A", AddressUpdateRequest{}); return err },
		func() error { _, err := s.Balance(ctx, "W", "A", "X"); return err },
		func() error { _, err := s.Balances(ctx, "W", "A"); return err },
		func() error { _, err := s.Transactions(ctx, "W", "A", TransactionListParams{}); return err },
		func() error { _, err := s.Transaction(ctx, "W", "A", "X"); return err },
		func() error { _, err := s.Whitelist(ctx, "W", AddressWhitelistRequest{Address: "0x1"}); return err },
		func() error { _, err := s.Unwhitelist(ctx, "W", "A"); return err },
		func() error { _, err := s.Validate(ctx, "0x1", "ethereum"); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	assertRoutes(t, hits(), []string{
		"POST /v1/wallets/W/addresses",
		"GET /v1/wallets/W/addresses",
		"GET /v1/wallets/W/addresses/A",
		"PATCH /v1/wallets/W/addresses/A",
		"GET /v1/wallets/W/addresses/A/balance",
		"GET /v1/wallets/W/addresses/A/balances",
		"GET /v1/wallets/W/addresses/A/transactions",
		"GET /v1/wallets/W/addresses/A/transactions/X",
		"POST /v1/wallets/W/addresses/whitelist",
		"DELETE /v1/wallets/W/addresses/whitelist/A",
		"GET /v1/addresses/validate",
	})
}

func TestAddressesListDecodes(t *testing.T) {
	c, got := server(t, 200, `{"analytics":{"activeAddressesCount":93,"addressLimit":-1,"addressesCount":94,"externalAddressesCount":2,"inactiveAddressesCount":1,"internalAddressesCount":92},"data":[{"address":"0x7204855eA98E951a218A9CC69786D7Dc749d170A","blockchain":{"createdAt":"2024-05-14T11:53:33.095Z","derivationPath":"m/44'/60'/0'/0","id":"85ffc132-3972-4c9e-99a5-5cf0ccb688bf","isActive":true,"isEvmCompatible":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800081/crypto-assets/ethereum-eth-logo_idraq2.png","name":"ethereum","slug":"ethereum","symbol":"eth","tokenStandard":"ERC20","updatedAt":"2024-06-14T22:32:11.983Z"},"configurations":{"aml":{"message":"Address is not sanctioned","provider":"ofac","status":"success"},"disableAutoSweep":false,"enableGaslessWithdraw":false,"showPrivateKey":false},"createdAt":"2024-10-23T10:53:38.622Z","derivationPath":"m/44'/60'/0'/0/86","id":"259422b2-6c8f-40e0-822f-7d2d23c78323","isActive":true,"metadata":null,"name":null,"network":"testnet","type":"INTERNAL","updatedAt":"2024-10-23T10:53:38.622Z"}],"message":"Addresses fetched successfully","meta":{"currentPage":1,"itemCount":49,"itemsPerPage":100,"totalItems":49,"totalPages":1},"statusCode":200}`)
	l, err := c.Addresses.List(context.Background(), "W", PageParams{Page: 1, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "limit=100&page=1" {
		t.Fatalf("query = %q", got.query)
	}
	if l.Analytics.AddressesCount != 94 || l.Analytics.AddressLimit != -1 || l.Meta.TotalItems != 49 || len(l.Data) != 1 {
		t.Fatalf("list = %+v", l)
	}
	a := l.Data[0]
	if a.Type != "INTERNAL" || a.Configurations.AML.Provider != "ofac" || a.Blockchain.Slug != "ethereum" {
		t.Fatalf("address = %+v", a)
	}
}

func TestAddressesGetDecodes(t *testing.T) {
	c, got := server(t, 200, `{"data":{"address":"0x7204855eA98E951a218A9CC69786D7Dc749d170A","blockchain":{"createdAt":"2024-05-14T11:53:33.095Z","derivationPath":"m/44'/60'/0'/0","id":"85ffc132-3972-4c9e-99a5-5cf0ccb688bf","isActive":true,"isEvmCompatible":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800081/crypto-assets/ethereum-eth-logo_idraq2.png","name":"ethereum","slug":"ethereum","symbol":"eth","tokenStandard":"ERC20","updatedAt":"2024-06-14T22:32:11.983Z"},"configurations":{"aml":{"message":"Address is not sanctioned","provider":"ofac","status":"success"},"disableAutoSweep":false,"enableGaslessWithdraw":false,"showPrivateKey":false},"createdAt":"2024-10-23T10:53:38.622Z","derivationPath":"m/44'/60'/0'/0/86","id":"259422b2-6c8f-40e0-822f-7d2d23c78323","isActive":true,"metadata":null,"name":null,"network":"testnet","type":"INTERNAL","updatedAt":"2024-10-23T10:53:38.622Z"},"message":"Address fetched successfully","statusCode":200}`)
	a, err := c.Addresses.Get(context.Background(), "W", "A", true)
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "showPrivateKey=true" {
		t.Fatalf("query = %q", got.query)
	}
	if a.ID != "259422b2-6c8f-40e0-822f-7d2d23c78323" || a.Configurations.ShowPrivateKey || a.DerivationPath != "m/44'/60'/0'/0/86" {
		t.Fatalf("address = %+v", a)
	}
}

func TestAddressesValidateDecodes(t *testing.T) {
	c, got := server(t, 200, `{"data":{"isValid":true,"address":"0x451dEFC27B45808078e875556AF06bCFdC697BA4","blockchain":"ethereum"},"message":"Address validation","statusCode":200}`)
	v, err := c.Addresses.Validate(context.Background(), "0x451defc27b45808078e875556af06bcfdc697ba4", "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "address=0x451defc27b45808078e875556af06bcfdc697ba4&blockchain=ethereum" {
		t.Fatalf("query = %q", got.query)
	}
	if !v.IsValid || v.Address != "0x451dEFC27B45808078e875556AF06bCFdC697BA4" {
		t.Fatalf("validation = %+v", v)
	}
}

func TestAddressesTransactionDecodes(t *testing.T) {
	c, _ := server(t, 200, `{"data":{"address":{"address":"0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23","configurations":{"aml":{"message":"Address is not sanctioned","provider":"ofac, fbi, tether, circle","status":"success"}},"createdAt":"2025-05-18T16:41:34.119Z","derivationPath":null,"id":"4016e25b-77a0-46f6-9c3a-9c94e9fd9741","isActive":true,"metadata":null,"name":"External address metamask","network":"mainnet","type":"EXTERNAL","updatedAt":"2025-05-18T16:41:34.119Z"},"amlScreening":{"message":"Address is not sanctioned","provider":"ofac, fbi, tether, circle","status":"success"},"amount":"0.447014","amountPaid":"0.447014","amountUSD":"0.447014","asset":{"address":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","createdAt":"2024-05-27T14:31:19.436Z","currency":"USD","decimals":6,"id":"873b4eaa-b078-48e5-b298-b46e9a1b73cd","isActive":true,"isNative":false,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800083/crypto-assets/usd-coin-usdc-logo_fs9mhv.png","name":"USD Coin","network":"mainnet","standard":null,"symbol":"USDC","updatedAt":"2025-04-17T04:50:19.999Z"},"assetSwept":null,"assetSweptAmount":null,"assetSweptAt":null,"assetSweptGasFee":null,"assetSweptHash":null,"assetSweptRecipientAddress":null,"assetSweptSenderAddress":null,"beneficiary":null,"blockHash":"0x0a8171c1a6a1567bf8428af5dc73115c18096cfa0357599892d54e8582ac6747","blockNumber":32989728,"blockchain":{"createdAt":"2024-05-27T14:31:14.966Z","derivationPath":"m/44'/60'/0'/0","id":"74733889-4ecd-403e-9840-94e87c043f24","isActive":true,"isEvmCompatible":true,"isL2":true,"logoUrl":"https://res.cloudinary.com/blockradar/image/upload/v1716800080/crypto-assets/Base_Network_Logo_vqyh7r.png","name":"base","slug":"base","symbol":"eth","tokenStandard":null,"updatedAt":"2024-11-26T20:04:13.945Z"},"chainId":8453,"confirmations":6,"confirmed":true,"convertedAmount":0.447014,"convertedAssetSweptGasFee":null,"convertedGasFee":0.5333701926304428,"createdAt":"2025-07-17T22:13:35.305Z","currency":"USD","customer":null,"explorerUrl":"https://basescan.org","fee":null,"feeHash":null,"gasFee":"0.0001174782810805","gasPrice":"200001500","gasUsed":"587387","hash":"0x96eb0184094a932a8a449dc6a43e0e464221c95320980e2a8573697dc5f18d6e","id":"4a89bfaf-04a4-4303-92cc-c701a84a1d27","metadata":null,"network":"mainnet","note":null,"paymentLink":null,"rate":null,"rateUSD":"1","reason":"Deposit to child wallet","recipientAddress":"0xe59aaf21c4D9Cf92d9eD4537f4404BA031f83b23","reference":"WQbCFjzeVq","senderAddress":"0x09aea4b2242abC8bb4BB78D537A67a245A7bEC64","signedTransaction":null,"status":"SUCCESS","toAmount":null,"toAsset":null,"toBlockchain":null,"toCurrency":null,"toExplorerUrl":null,"toWallet":null,"tokenAddress":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","type":"DEPOSIT","updatedAt":"2025-07-25T21:02:22.665Z"},"message":"Transaction fetched successfully","statusCode":200}`)
	tx, err := c.Addresses.Transaction(context.Background(), "W", "A", "X")
	if err != nil {
		t.Fatal(err)
	}
	if tx.ID != "4a89bfaf-04a4-4303-92cc-c701a84a1d27" || tx.Amount != "0.447014" || tx.ChainID.String() != "8453" {
		t.Fatalf("tx = %+v", tx.Transaction)
	}
	if tx.AmountUSD != "0.447014" || tx.ConvertedGasFee != "0.5333701926304428" || tx.ExplorerURL != "https://basescan.org" || tx.Address.Type != "EXTERNAL" {
		t.Fatalf("detail = %+v", tx)
	}
}

func TestAddressesBodies(t *testing.T) {
	c, got := server(t, 200, `{"statusCode":200,"message":"ok","data":null}`)
	ctx := context.Background()
	if _, err := c.Addresses.Generate(ctx, "W", AddressGenerateRequest{Name: "Customer 1", Metadata: map[string]any{"user_id": 1}, DisableAutoSweep: true}); err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"disableAutoSweep":true,"metadata":{"user_id":1},"name":"Customer 1"}` {
		t.Fatalf("generate body = %s", b)
	}
	if _, err := c.Addresses.Update(ctx, "W", "A", AddressUpdateRequest{IsActive: Bool(false), EnableGaslessWithdraw: Bool(true)}); err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"enableGaslessWithdraw":true,"isActive":false}` {
		t.Fatalf("update body = %s", b)
	}
	if _, err := c.Addresses.Whitelist(ctx, "W", AddressWhitelistRequest{Address: "0x1", Name: "ext"}); err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"address":"0x1","name":"ext"}` {
		t.Fatalf("whitelist body = %s", b)
	}
	if _, err := c.Addresses.Transactions(ctx, "W", "A", TransactionListParams{Status: "SUCCESS", Assets: []string{"USDT", "USDC"}}); err != nil {
		t.Fatal(err)
	}
	if got.query != "assets=USDT%2CUSDC&status=SUCCESS" {
		t.Fatalf("transactions query = %q", got.query)
	}
}
