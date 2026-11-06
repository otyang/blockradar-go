//go:build integration

// Read-only checks against the real API. Nothing here moves funds.
//
//	BLOCKRADAR_API_KEY=test_... go test -tags integration -run Sandbox -v ./...
package blockradar

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type sandbox struct {
	t   *testing.T
	c   *Client
	ctx context.Context

	mu      sync.Mutex
	ok      []string
	refused []string
}

// check records one call: API refusals (4xx/5xx) are logged, anything else
// (network, decoding) fails the test because it means the client is wrong.
func (s *sandbox) check(name string, err error, detail string) bool {
	s.t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	var apiErr *APIError
	var netErr net.Error
	switch {
	case err == nil:
		s.ok = append(s.ok, name+"  "+detail)
		return true
	case errors.As(err, &netErr) && netErr.Timeout():
		s.refused = append(s.refused, name+"  no response from server (timeout)")
	case errors.As(err, &apiErr):
		msg := strings.Join(apiErr.Messages, "; ")
		if msg == "" {
			msg = string(apiErr.Body)
		}
		s.refused = append(s.refused, fmt.Sprintf("%s  HTTP %d %s", name, apiErr.HTTPStatus, trim(msg, 120)))
	default:
		s.t.Errorf("%s: %v", name, err)
	}
	return false
}

func trim(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func TestSandbox(t *testing.T) {
	key := os.Getenv("BLOCKRADAR_API_KEY")
	if key == "" {
		t.Skip("BLOCKRADAR_API_KEY not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	s := &sandbox{t: t, c: NewClient(key), ctx: ctx}
	c := s.c

	// ---- account-wide ----
	chains, err := c.Misc.Blockchains(ctx)
	s.check("Misc.Blockchains", err, fmt.Sprintf("%d chains", len(chains)))
	assets, err := c.Misc.Assets(ctx, MiscAssetsParams{Network: "testnet"})
	s.check("Misc.Assets", err, fmt.Sprintf("%d assets", len(assets)))
	rates, err := c.Misc.Rates(ctx, "USD", "USDC", "USDT")
	s.check("Misc.Rates", err, fmt.Sprintf("%v", rates))

	customers, err := c.Customers.List(ctx)
	s.check("Customers.List", err, fmt.Sprintf("%d customers", len(customers)))
	if len(customers) > 0 {
		cu, err := c.Customers.Get(ctx, customers[0].ID)
		if s.check("Customers.Get", err, "") {
			_ = cu
		}
		ob, err := c.Onboarding.List(ctx, customers[0].ID, OnboardingListParams{Limit: 5})
		if s.check("Onboarding.List", err, "") {
			s.ok[len(s.ok)-1] += fmt.Sprintf("%d requests", len(ob.Items))
		}
	}

	gw, err := c.Gateway.Wallets(ctx, PageParams{Page: 1, Limit: 10})
	s.check("Gateway.Wallets", err, fmt.Sprintf("%d wallets", len(gw)))
	_, err = c.Gateway.Balance(ctx)
	s.check("Gateway.Balance", err, "")
	gtx, err := c.Gateway.Transactions(ctx, PageParams{Page: 1, Limit: 5})
	if s.check("Gateway.Transactions", err, "") {
		s.ok[len(s.ok)-1] += fmt.Sprintf("%d txs", len(gtx.Data))
	}

	tb, err := c.LiquidityPool.TreasuryBalances(ctx)
	s.check("LiquidityPool.TreasuryBalances", err, fmt.Sprintf("%d balances", len(tb)))
	bm, err := c.LiquidityPool.MarketBenchmark(ctx, LiquidityBenchmarkParams{FromAsset: "USDT", ToAsset: "cNGN"})
	s.check("LiquidityPool.MarketBenchmark", err, fmt.Sprintf("%d", len(bm)))
	_, err = c.LiquidityPool.CheckPair(ctx, "USDT", "cNGN")
	s.check("LiquidityPool.CheckPair", err, "")

	pl, err := c.PaymentLinks.List(ctx)
	if s.check("PaymentLinks.List", err, "") {
		s.ok[len(s.ok)-1] += fmt.Sprintf("%d links", len(pl.Data))
		if len(pl.Data) > 0 {
			_, err = c.PaymentLinks.Get(ctx, pl.Data[0].ID)
			s.check("PaymentLinks.Get", err, "")
		}
	}

	rb, err := c.Rewards.Business(ctx)
	s.check("Rewards.Business", err, fmt.Sprintf("%d positions", len(rb)))
	rp, err := c.Rewards.Products(ctx, RewardProductsParams{})
	s.check("Rewards.Products", err, fmt.Sprintf("%d products", len(rp)))
	_, err = c.Rewards.SupportedChains(ctx)
	s.check("Rewards.SupportedChains", err, "")
	rt, err := c.Rewards.Transactions(ctx, RewardTransactionsParams{Limit: 5})
	s.check("Rewards.Transactions", err, fmt.Sprintf("%d txs", len(rt)))

	// ---- per wallet ----
	wallets, err := c.Wallets.List(ctx)
	if !s.check("Wallets.List", err, fmt.Sprintf("%d wallets", len(wallets))) || len(wallets) == 0 {
		t.Fatal("no wallets to test with")
	}
	for _, w := range wallets { // one at a time: the sandbox rate-limits bursts
		s.wallet(w)
	}

	t.Logf("\n==== OK (%d) ====\n%s", len(s.ok), strings.Join(s.ok, "\n"))
	t.Logf("\n==== Refused by API (%d) ====\n%s", len(s.refused), strings.Join(s.refused, "\n"))
}

func (s *sandbox) wallet(w Wallet) {
	c, ctx := s.c, s.ctx
	chain := ""
	if w.Blockchain != nil {
		chain = w.Blockchain.Slug
	}
	p := "[" + chain + "] "

	wd, err := c.Wallets.Get(ctx, w.ID, false)
	if s.check(p+"Wallets.Get", err, "") {
		s.ok[len(s.ok)-1] += fmt.Sprintf("%s, %d assets", wd.Name, len(wd.Assets))
	}
	bals, err := c.Wallets.Balances(ctx, w.ID)
	s.check(p+"Wallets.Balances", err, balances(bals))
	if len(w.Assets) > 0 {
		b, err := c.Wallets.Balance(ctx, w.ID, w.Assets[0].ID)
		if s.check(p+"Wallets.Balance", err, "") {
			s.ok[len(s.ok)-1] += string(b.Balance) + " " + b.Asset.Asset.Symbol
		}
	}
	wl, err := c.Wallets.WebhookLogs(ctx, w.ID, PageParams{Page: 1, Limit: 5})
	if s.check(p+"Wallets.WebhookLogs", err, "") {
		s.ok[len(s.ok)-1] += fmt.Sprintf("%d logs", len(wl.Data))
	}
	wa, err := c.Assets.List(ctx, w.ID)
	s.check(p+"Assets.List", err, fmt.Sprintf("%d assets", len(wa)))

	_, err = c.Addresses.Validate(ctx, w.Address, chain)
	s.check(p+"Addresses.Validate", err, "")
	_, err = c.AML.Lookup(ctx, w.Address, chain)
	s.check(p+"AML.Lookup", err, "")

	txs, err := c.Transactions.List(ctx, w.ID, TransactionListParams{Page: 1, Limit: 5})
	if s.check(p+"Transactions.List", err, "") {
		s.ok[len(s.ok)-1] += fmt.Sprintf("%d txs (total %d)", len(txs.Data), txs.Meta.TotalItems)
		if len(txs.Data) > 0 {
			tx := txs.Data[0]
			_, err = c.Transactions.Get(ctx, w.ID, tx.ID)
			s.check(p+"Transactions.Get", err, tx.Type+" "+tx.Status+" "+string(tx.Amount))
			_, err = c.Transactions.WebhookLogs(ctx, w.ID, tx.ID)
			s.check(p+"Transactions.WebhookLogs", err, "")
		}
	}

	addrs, err := c.Addresses.List(ctx, w.ID, PageParams{Page: 1, Limit: 5})
	if s.check(p+"Addresses.List", err, "") {
		s.ok[len(s.ok)-1] += fmt.Sprintf("%d addresses (total %d)", len(addrs.Data), addrs.Meta.TotalItems)
		if len(addrs.Data) > 0 {
			a := addrs.Data[0]
			_, err = c.Addresses.Get(ctx, w.ID, a.ID, false)
			s.check(p+"Addresses.Get", err, a.Address)
			ab, err := c.Addresses.Balances(ctx, w.ID, a.ID)
			s.check(p+"Addresses.Balances", err, balances(ab))
			if len(w.Assets) > 0 {
				_, err = c.Addresses.Balance(ctx, w.ID, a.ID, w.Assets[0].ID)
				s.check(p+"Addresses.Balance", err, "")
			}
			at, err := c.Addresses.Transactions(ctx, w.ID, a.ID, TransactionListParams{Page: 1, Limit: 5})
			if s.check(p+"Addresses.Transactions", err, "") && len(at.Data) > 0 {
				_, err = c.Addresses.Transaction(ctx, w.ID, a.ID, at.Data[0].ID)
				s.check(p+"Addresses.Transaction", err, "")
			}
			_, err = c.Gateway.AddressBalance(ctx, a.ID)
			s.check(p+"Gateway.AddressBalance", err, "")
			rw, err := c.Rewards.Wallet(ctx, w.ID, RewardWalletParams{AddressID: a.ID})
			s.check(p+"Rewards.Wallet(address)", err, fmt.Sprintf("%d positions", len(rw)))
			vl, err := c.VirtualAccounts.AddressList(ctx, w.ID, a.ID, ListVirtualAccountsParams{})
			if s.check(p+"VirtualAccounts.AddressList", err, "") {
				s.ok[len(s.ok)-1] += fmt.Sprintf("%d accounts", len(vl.Data))
			}
		}
	}

	rw, err := c.Rewards.Wallet(ctx, w.ID, RewardWalletParams{})
	s.check(p+"Rewards.Wallet", err, fmt.Sprintf("%d positions", len(rw)))
	ra, err := c.Rewards.WalletAssets(ctx, w.ID, RewardAssetsParams{})
	s.check(p+"Rewards.WalletAssets", err, fmt.Sprintf("%d assets", len(ra)))

	// fee estimates and quotes only: nothing is sent
	if len(w.Assets) > 0 {
		nf, err := c.Withdraw.NetworkFee(ctx, w.ID, WithdrawRequest{AssetID: w.Assets[0].ID, Address: w.Address, Amount: "0.0001"})
		if s.check(p+"Withdraw.NetworkFee", err, "") {
			s.ok[len(s.ok)-1] += fmt.Sprintf("fee %s", nf.NetworkFee)
		}
	}
	if from, to, ok := swapPair(w.Assets); ok {
		sq, err := c.Swap.Quote(ctx, w.ID, SwapRequest{FromAssetID: from.ID, ToAssetID: to.ID, Amount: "1"})
		if s.check(p+"Swap.Quote", err, "") {
			s.ok[len(s.ok)-1] += fmt.Sprintf("1 %s -> %s %s", from.Asset.Symbol, sq.Amount, to.Asset.Symbol)
		}
	}
	as, err := c.AutoSettlements.Get(ctx, w.ID)
	if s.check(p+"AutoSettlements.Get", err, "") {
		_ = as
	}
	ar, err := c.AutoSettlements.ListRules(ctx, w.ID)
	s.check(p+"AutoSettlements.ListRules", err, fmt.Sprintf("%d rules", len(ar)))
	da, err := c.DepositFiat.SupportedAssets(ctx, w.ID, "")
	s.check(p+"DepositFiat.SupportedAssets", err, fmt.Sprintf("%d assets", len(da)))
	if len(da) > 0 {
		dc, err := c.DepositFiat.Currencies(ctx, w.ID, da[0].ID, "")
		s.check(p+"DepositFiat.Currencies", err, fmt.Sprintf("%d currencies", len(dc)))
	}
	do, err := c.DepositFiat.ListOrders(ctx, w.ID, PaymentOrderListParams{Limit: 5})
	if s.check(p+"DepositFiat.ListOrders", err, "") {
		s.ok[len(s.ok)-1] += fmt.Sprintf("%d orders", len(do.Items))
	}

	// v2
	fa, err := c.WithdrawFiat.SupportedAssets(ctx, w.ID, "")
	s.check(p+"WithdrawFiat.SupportedAssets", err, fmt.Sprintf("%d assets", len(fa)))
	fc, err := c.WithdrawFiat.Currencies(ctx, w.ID, "")
	s.check(p+"WithdrawFiat.Currencies", err, fmt.Sprintf("%d currencies", len(fc)))
	if len(fa) > 0 && len(fc) > 0 {
		rate, err := c.WithdrawFiat.Rates(ctx, w.ID, WithdrawFiatRatesParams{AssetID: fa[0].ID, Currency: fc[0].Code, Amount: "10"})
		if s.check(p+"WithdrawFiat.Rates", err, "") {
			s.ok[len(s.ok)-1] += fmt.Sprintf("10 %s = %s %s", fa[0].Info().Symbol, rate.Amount, fc[0].Code)
			pm, err := c.WithdrawFiat.PaymentMethods(ctx, w.ID, WithdrawFiatFlowParams{SessionID: rate.SessionID})
			if s.check(p+"WithdrawFiat.PaymentMethods", err, "") {
				s.ok[len(s.ok)-1] += fmt.Sprintf("%d methods", len(pm.PaymentMethods))
				if len(pm.PaymentMethods) > 0 {
					_, err = c.WithdrawFiat.PaymentMethodRequirements(ctx, w.ID, pm.PaymentMethods[0].ID, WithdrawFiatFlowParams{SessionID: rate.SessionID})
					s.check(p+"WithdrawFiat.PaymentMethodRequirements", err, pm.PaymentMethods[0].ID)
				}
			}
		}
	}
	va, err := c.VirtualAccounts.List(ctx, w.ID, ListVirtualAccountsParams{})
	if s.check(p+"VirtualAccounts.List", err, "") {
		s.ok[len(s.ok)-1] += fmt.Sprintf("%d accounts", len(va.Data))
	}
	vd, err := c.VirtualAccounts.Discovery(ctx, w.ID, "")
	if s.check(p+"VirtualAccounts.Discovery", err, "") {
		s.ok[len(s.ok)-1] += fmt.Sprintf("%d asset groups", len(vd.Assets))
	}
}

func balances(bs []Balance) string {
	var parts []string
	for _, b := range bs {
		parts = append(parts, string(b.Balance)+" "+b.Asset.Asset.Symbol)
	}
	return strings.Join(parts, ", ")
}

// swapPair picks two wallet assets with different symbols.
func swapPair(as []WalletAsset) (WalletAsset, WalletAsset, bool) {
	for _, a := range as {
		for _, b := range as {
			if a.Asset.Symbol != b.Asset.Symbol {
				return a, b, true
			}
		}
	}
	return WalletAsset{}, WalletAsset{}, false
}
