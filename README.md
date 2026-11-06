# blockradar-go

Go client for the [Blockradar API](https://docs.blockradar.co).

- **v2** endpoints for Withdraw Fiat and Virtual Accounts (the only areas Blockradar has a v2 for).
- **v1** endpoints for everything else: wallets, addresses, transactions, withdrawals, swaps, signing, smart contracts, customers, deposit fiat, gateway, rewards, and more.
- Webhook signature verification.
- Standard library only. Go 1.22+.

```sh
go get github.com/otyang/blockradar-go
```

## Quick start

```go
import blockradar "github.com/otyang/blockradar-go"

c := blockradar.NewClient(os.Getenv("BLOCKRADAR_API_KEY"))

wallet, err := c.Wallets.Get(ctx, walletID, false)
```

The API key decides the environment (testnet or mainnet). `c.Wallets.List(ctx)` returns
every master wallet the key can access, which is the easiest way to find wallet IDs.
Every method takes a `context.Context`, so you can set timeouts and cancel requests.

Rate-limited requests (HTTP 429) are retried automatically: up to 3 times, waiting for
`Retry-After` if the server sends it (at most 60s), otherwise 1s, 2s, 4s.

Options:

```go
c := blockradar.NewClient(key,
    blockradar.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
    blockradar.WithMaxRetries(5),                    // 0 disables 429 retries
    blockradar.WithBaseURL("http://localhost:8080"), // tests / mocks
)
```

## How it's organised

`Client` has one field per API area: `c.Wallets`, `c.Addresses`, `c.Withdraw`, `c.WithdrawFiat`, ...
Methods acting on a child address instead of the master wallet have an `Address` prefix:

```go
c.Withdraw.Execute(ctx, walletID, req)                     // from the master wallet
c.Withdraw.AddressExecute(ctx, walletID, addressID, req)   // from a child address
```

Responses come back unwrapped: you get the `data` object, not the `{status, message, data}` envelope.
Paginated endpoints return `*List[T]` (v1) or `*Page[T]` (v2) with the paging metadata.

Amounts are strings (`"100.50"`), never floats. In v1 responses they use the `Amount` type,
because the API sometimes sends them as JSON numbers.

## Off-ramp: stablecoin → bank account (v2)

```go
// 1. Rate + session. Keep the session ID for every later call.
rate, err := c.WithdrawFiat.Rates(ctx, walletID, blockradar.WithdrawFiatRatesParams{
    AssetID: usdcAssetID, Currency: "NGN", Amount: "100",
})

// 2. Payment methods and the form fields each one needs (JSON Schema).
methods, err := c.WithdrawFiat.PaymentMethods(ctx, walletID,
    blockradar.WithdrawFiatFlowParams{SessionID: rate.SessionID})

// 3. Quote, then execute with the same request.
q := blockradar.WithdrawFiatQuoteRequest{
    SessionID:         rate.SessionID,
    PaymentMethod:     methods.PaymentMethods[0].ID,
    PaymentMethodData: map[string]any{"accountNumber": "0123456789", "bankCode": "058"},
}
quote, err := c.WithdrawFiat.Quote(ctx, walletID, q)
tx, err := c.WithdrawFiat.Execute(ctx, walletID, blockradar.WithdrawFiatExecuteRequest{
    WithdrawFiatQuoteRequest: q,
    Reference:                "order-42",
})
```

## On-ramp: a bank account that settles into a wallet (v2)

```go
reqs, err := c.VirtualAccounts.Requirements(ctx, walletID,
    blockradar.VirtualAccountRequirementsParams{Currency: "NGN"})

res, err := c.VirtualAccounts.AddressCreate(ctx, walletID, addressID, blockradar.CreateVirtualAccountRequest{
    Provider: reqs.Provider,
    Currency: "NGN",
    AdditionalData: map[string]any{"bvn": "12345678901"}, // when reqs.AdditionalDataRequired
})
fmt.Println(res.VirtualAccount.BankName, res.VirtualAccount.AccountNumber)
```

## Errors

Any non-2xx response is a `*blockradar.APIError`:

```go
var apiErr *blockradar.APIError
if errors.As(err, &apiErr) {
    log.Println(apiErr.HTTPStatus, apiErr.Messages, apiErr.RequestID)

    switch apiErr.Code() {
    case blockradar.CodeWithdrawalSessionExpired:
        // start again from Rates
    case blockradar.CodeCustomerOnboardingRequired:
        // apiErr.Details.Onboarding.Action says what to call next
    }
}
```

## Webhooks

Blockradar signs each webhook with HMAC-SHA512 of the raw body, keyed with your API key,
in the `x-blockradar-signature` header.

```go
http.HandleFunc("/webhooks/blockradar", func(w http.ResponseWriter, r *http.Request) {
    ev, err := blockradar.ParseWebhook(r, apiKey)
    if err != nil {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }
    w.WriteHeader(http.StatusOK) // acknowledge first, then process

    switch ev.Event {
    case "deposit.success":
        var tx blockradar.Transaction
        _ = json.Unmarshal(ev.Data, &tx)
        // credit the user
    }
})
```

Blockradar retries failed deliveries 5 times over about 2.5 hours, so make your handler idempotent
(e.g. key on the transaction ID).

## Endpoints

Methods prefixed with `Address` act on a child address; the rest act on the master wallet or the account.

| Client field | API | Methods |
|---|---|---|
| `c.WithdrawFiat` | v2 | SupportedAssets, Currencies, Rates, PaymentMethods, PaymentMethodRequirements, ResolvePaymentMethod, Quote, Execute, AddressQuote, AddressExecute |
| `c.VirtualAccounts` | v2 | Discovery, Requirements, Create, List, Get, Update, Regenerate, DestinationDiscovery, ChangeDestination, AddressCreate, AddressList, AddressGet, AddressUpdate, AddressRegenerate, AddressTransactions |
| `c.Wallets` | v1 | List, Get, Update, Balance, Balances, WebhookLogs, RescanBlocks, SweepAssets |
| `c.Addresses` | v1 | Generate, List, Get, Update, Balance, Balances, Transactions, Transaction, Whitelist, Unwhitelist, Validate |
| `c.Assets` | v1 | List, Add, Update, Remove |
| `c.Transactions` | v1 | List, Get, WebhookLogs, Cancel, Retry, SweepAssets, ResendWebhook |
| `c.Withdraw` | v1 | Execute, Sign, NetworkFee, AddressExecute, AddressSign, AddressNetworkFee |
| `c.Swap` | v1 | Quote, Execute, AddressQuote, AddressExecute |
| `c.Signing` | v1 | Message, TypedData, Transaction, Broadcast, AddressMessage, AddressTypedData, AddressTransaction, AddressBroadcast |
| `c.Contracts` | v1 | Read, Write, WriteSign, NetworkFee, AddressRead, AddressWrite, AddressWriteSign, AddressNetworkFee |
| `c.AssetRecovery` | v1 | Execute, NetworkFee |
| `c.Customers` | v1 | Create, List, Get, Update |
| `c.Onboarding` | v1 | Create, List, Get, SaveDraft, Submit, Refresh, Reopen, SubmitHostedSession, SubmitAction, ConfirmationRequirements, SubmitConfirmation, PrepareUploads, UploadDocument, DocumentViewURLs |
| `c.DepositFiat` | v1 | SupportedAssets, Currencies, Rails, ResolvePaymentOrderRequirements, Quote, CreateOrder, ListOrders, GetOrder, CancelOrder, RefreshOrder, AddressQuote, AddressCreateOrder, AddressListOrders |
| `c.PaymentLinks` | v1 | Create, List, Get, Update, Transactions |
| `c.AutoSettlements` | v1 | Get, Update, ListRules, GetRule, CreateRule, UpdateRule, DeleteRule, CopyRules, AddressGet, AddressUpdate, AddressListRules, AddressGetRule, AddressCreateRule, AddressUpdateRule, AddressDeleteRule |
| `c.Gateway` | v1 | Wallets, Balance, Transactions, Deposit, DepositNetworkFee, Withdraw, WithdrawNetworkFee, AddressBalance, AddressTransactions, AddressDeposit, AddressDepositNetworkFee, AddressWithdraw, AddressWithdrawNetworkFee |
| `c.LiquidityPool` | v1 | Create, Get, Update, Deactivate, Reactivate, History, CheckPair, MarketBenchmark, TreasuryBalances |
| `c.Rewards` | v1 | Business, Products, SupportedChains, Transactions, Wallet, WalletAssets, Deposit, DepositNetworkFee, Withdraw, WithdrawNetworkFee |
| `c.Misc` | v1 | Assets, Rates, Blockchains |
| `c.AML` | v1 | Lookup |

## Tests

```sh
go test ./...
```

These run against a local fake server. For each area, a test calls every method and checks
the HTTP method and path against the list in Blockradar's OpenAPI docs.

### Sandbox check

An opt-in, read-only test calls the real API with a testnet key, across every wallet the key can see:

```sh
BLOCKRADAR_API_KEY=test_... go test -tags integration -run TestSandbox -v ./...
```

It only reads data and asks for fee estimates and quotes; it never sends a withdrawal, swap or
anything else that moves funds. Responses the client can't decode fail the test. Refusals from
the API (feature not enabled, mainnet-only, ...) are listed but don't fail it.

### End-to-end check (moves testnet funds)

Creates a child address, sends USDC from the master wallet to it, waits for the deposit, and
checks the webhooks. It refuses to run against a mainnet wallet.

```sh
BLOCKRADAR_API_KEY=test_... \
BLOCKRADAR_E2E_WALLET=<testnet wallet id holding native gas + USDC> \
BLOCKRADAR_E2E_WEBHOOK_LOG=/path/to/webhooks.jsonl \  # optional, written by your webhook receiver
go test -tags e2e -run E2E -v -timeout 20m ./...
```

## Where the live API differs from the docs

Found by running the sandbox check:

- `DepositFiat.ListOrders` returns `data: {items, meta, analytics}`, not a flat array.
  `PaymentOrderPage` follows the live shape.
- v2 `WithdrawFiat.SupportedAssets` returns wallet assets (`{id, isActive, asset: {...}}`), not flat assets.
  `FiatAsset` accepts both; use `asset.Info()` to read the token details.
- `GET /v1/wallets` (list wallets) works but isn't in the reference. It's exposed as `Wallets.List`.
- `Transactions.List` analytics use `totalDepositsCount`, `totalDepositsSum`, … rather than the documented
  `depositsCount`, …. `TransactionAnalytics` has both sets of fields.
- Withdraw network-fee estimates also return `networkFeeInUSD` and `transactionFee`.

Other doc quirks:

- The docs list the child-address smart-contract network-fee endpoint without the `/v1` prefix;
  this client uses `/v1/...` like its sibling endpoints.
- `GET /v1/gateway/Transactions` is documented with a capital `T` and is called exactly as documented.
- The legacy v1 Withdraw Fiat and v1 Virtual Accounts endpoints are not included; use the v2 ones.
