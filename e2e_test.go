//go:build e2e

// End-to-end test against the Blockradar testnet. Unlike the sandbox test,
// this one WRITES: it creates a child address and moves test coins.
//
//	BLOCKRADAR_API_KEY=test_...            testnet API key
//	BLOCKRADAR_E2E_WALLET=<wallet id>      testnet master wallet holding native gas + USDC
//	BLOCKRADAR_E2E_WEBHOOK_LOG=<file>      optional: JSONL written by a webhook receiver
//	                                       (one object per delivery: {"event","valid","body"})
//	BLOCKRADAR_E2E_AMOUNT=1                optional: USDC to move (default 1)
//
//	go test -tags e2e -run E2E -v -timeout 20m ./...
package blockradar

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestE2E(t *testing.T) {
	key, walletID := os.Getenv("BLOCKRADAR_API_KEY"), os.Getenv("BLOCKRADAR_E2E_WALLET")
	if key == "" || walletID == "" {
		t.Skip("set BLOCKRADAR_API_KEY and BLOCKRADAR_E2E_WALLET")
	}
	amount := envOr("BLOCKRADAR_E2E_AMOUNT", "1")
	hookLog := os.Getenv("BLOCKRADAR_E2E_WEBHOOK_LOG")
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Minute)
	defer cancel()
	c := NewClient(key)
	run := time.Now().UTC().Format("20060102-150405")

	wallet, err := c.Wallets.Get(ctx, walletID, false)
	must(t, "Wallets.Get", err)
	if wallet.Network != "testnet" {
		t.Fatalf("refusing to run on %s wallet %s", wallet.Network, wallet.ID)
	}
	chain := wallet.Blockchain.Slug
	t.Logf("wallet %s (%s) %s", wallet.Name, chain, wallet.Address)

	// ---- address lifecycle (no funds needed) ----
	var child *Address
	t.Run("address lifecycle", func(t *testing.T) {
		child, err = c.Addresses.Generate(ctx, walletID, AddressGenerateRequest{
			Name: "sdk-e2e-" + run, Metadata: map[string]any{"source": "blockradar-go e2e", "run": run},
		})
		must(t, "Addresses.Generate", err)
		t.Logf("child address %s (%s)", child.Address, child.ID)

		got, err := c.Addresses.Get(ctx, walletID, child.ID, false)
		must(t, "Addresses.Get", err)
		if got.Address != child.Address {
			t.Fatalf("Get returned %s, want %s", got.Address, child.Address)
		}
		upd, err := c.Addresses.Update(ctx, walletID, child.ID, AddressUpdateRequest{Name: "sdk-e2e-" + run + "-renamed"})
		must(t, "Addresses.Update", err)
		if upd.Name != "sdk-e2e-"+run+"-renamed" {
			t.Errorf("Update name = %q", upd.Name)
		}
		v, err := c.Addresses.Validate(ctx, child.Address, chain)
		must(t, "Addresses.Validate", err)
		t.Logf("validate: %+v", *v)
	})
	if child == nil {
		t.Fatal("no child address; cannot continue")
	}

	t.Run("whitelist add and remove", func(t *testing.T) {
		b := make([]byte, 20) // fresh random EVM address each run
		_, _ = rand.Read(b)
		ext := "0x" + hex.EncodeToString(b)
		wl, err := c.Addresses.Whitelist(ctx, walletID, AddressWhitelistRequest{Address: ext, Name: "sdk-e2e-" + run})
		if refused(t, "Addresses.Whitelist", err) {
			return
		}
		t.Logf("whitelisted %s as %s (type %s)", wl.Address, wl.ID, wl.Type)
		_, err = c.Addresses.Unwhitelist(ctx, walletID, wl.ID)
		refused(t, "Addresses.Unwhitelist", err)
	})

	t.Run("sign message", func(t *testing.T) {
		res, err := c.Signing.Message(ctx, walletID, SignMessageRequest{Message: "blockradar-go e2e " + run})
		if refused(t, "Signing.Message", err) {
			return
		}
		t.Logf("signed: status=%s signature=%s raw=%s", res.Status, short(res.SignedTransaction.Signature), short(res.SignedTransaction.Raw))
	})

	// ---- funds ----
	usdc, native, haveUSDC, haveGas := fundedAssets(ctx, t, c, walletID, amount)
	if !haveUSDC {
		t.Logf("NOT FUNDED: send testnet USDC and %s (gas) to %s on %s, then rerun.", nativeSymbol(wallet), wallet.Address, chain)
		t.Skip("no USDC; transfer and webhook steps skipped")
	}
	t.Logf("funded: %s USDC, %s %s", usdc.Balance, native.Balance, nativeSymbol(wallet))

	req := WithdrawRequest{
		AssetID:   usdc.Asset.ID,
		Address:   child.Address,
		Amount:    amount,
		Reference: "sdk-e2e-" + run,
		Note:      "blockradar-go e2e",
		Metadata:  map[string]any{"run": run},
	}

	t.Run("fee estimate and sign-only", func(t *testing.T) {
		fee, err := c.Withdraw.NetworkFee(ctx, walletID, req)
		must(t, "Withdraw.NetworkFee", err)
		t.Logf("network fee %s (≈$%s), arrival ~%ds", fee.NetworkFee, fee.NetworkFeeInUSD, fee.EstimatedArrivalTime)

		signReq := req
		signReq.Reference = "sdk-e2e-sign-" + run
		signed, err := c.Withdraw.Sign(ctx, walletID, signReq) // signed, not broadcast
		if !refused(t, "Withdraw.Sign", err) {
			t.Logf("sign-only: status=%s signedTx=%s", signed.Status, short(signed.SignedTransaction))
		}
	})

	if !haveGas {
		t.Logf("NO GAS: send testnet %s to %s on %s to run the transfer steps.", nativeSymbol(wallet), wallet.Address, chain)
		t.Skip("no native gas; transfer and webhook steps skipped")
	}

	var withdrawal *WithdrawResult
	t.Run("withdraw master -> child", func(t *testing.T) {
		withdrawal, err = c.Withdraw.Execute(ctx, walletID, req)
		must(t, "Withdraw.Execute", err)
		t.Logf("withdraw %s status=%s", withdrawal.ID, withdrawal.Status)

		final := pollTx(ctx, t, "withdrawal", func() (*TransactionDetail, error) {
			return c.Transactions.Get(ctx, walletID, withdrawal.ID)
		})
		if final.Status != "SUCCESS" {
			t.Fatalf("withdrawal ended %s: %s", final.Status, final.Reason)
		}
		t.Logf("withdrawal SUCCESS hash=%s fee=%s", final.Hash, final.GasFee)
		withdrawal.Hash = final.Hash
	})
	if withdrawal == nil || withdrawal.Hash == "" {
		t.Fatal("withdrawal did not complete")
	}

	var deposit *TransactionDetail
	t.Run("deposit seen on child address", func(t *testing.T) {
		deadline := time.Now().Add(10 * time.Minute)
		for time.Now().Before(deadline) {
			list, err := c.Addresses.Transactions(ctx, walletID, child.ID, TransactionListParams{Type: "DEPOSIT", Limit: 10})
			must(t, "Addresses.Transactions", err)
			for i := range list.Data {
				tx := list.Data[i]
				if strings.EqualFold(tx.Hash, withdrawal.Hash) && tx.Status == "SUCCESS" {
					deposit = &tx
				}
			}
			if deposit != nil {
				break
			}
			sleep(ctx, 10*time.Second)
		}
		if deposit == nil {
			t.Fatal("no successful DEPOSIT with the withdrawal hash on the child address within 10m")
		}
		t.Logf("deposit %s %s %s confirmed=%v swept=%v", deposit.ID, deposit.Amount, deposit.Asset.Symbol, deposit.Confirmed, deposit.AssetSwept)

		one, err := c.Addresses.Transaction(ctx, walletID, child.ID, deposit.ID)
		must(t, "Addresses.Transaction", err)
		if one.ID != deposit.ID {
			t.Errorf("Addresses.Transaction id = %s", one.ID)
		}
		bals, err := c.Addresses.Balances(ctx, walletID, child.ID)
		must(t, "Addresses.Balances", err)
		t.Logf("child balances now: %s (0 is expected if auto-sweep moved it to the master)", balancesText(bals))
	})

	t.Run("webhooks", func(t *testing.T) {
		logs, err := c.Transactions.WebhookLogs(ctx, walletID, withdrawal.ID)
		if !refused(t, "Transactions.WebhookLogs", err) {
			t.Logf("Blockradar delivery log for the withdrawal: %d attempts", len(logs))
		}
		if hookLog == "" {
			t.Skip("BLOCKRADAR_E2E_WEBHOOK_LOG not set; skipping delivery checks")
		}
		want := map[string]string{"withdraw.success": withdrawal.ID, "deposit.success": deposit.ID}
		got := waitWebhooks(ctx, t, hookLog, want, 5*time.Minute)
		for ev := range want {
			if !got[ev] {
				t.Errorf("no validly signed %s webhook for %s", ev, want[ev])
			}
		}

		before := countWebhooks(t, hookLog, "withdraw.success", withdrawal.ID)
		_, err = c.Transactions.ResendWebhook(ctx, walletID, withdrawal.ID)
		must(t, "Transactions.ResendWebhook", err)
		deadline := time.Now().Add(3 * time.Minute)
		for time.Now().Before(deadline) && countWebhooks(t, hookLog, "withdraw.success", withdrawal.ID) <= before {
			sleep(ctx, 5*time.Second)
		}
		if countWebhooks(t, hookLog, "withdraw.success", withdrawal.ID) <= before {
			t.Error("resent withdraw.success webhook did not arrive within 3m")
		} else {
			t.Log("resend delivered")
		}
	})
}

// ---- helpers ----

func fundedAssets(ctx context.Context, t *testing.T, c *Client, walletID, amount string) (usdc, native Balance, haveUSDC, haveGas bool) {
	bals, err := c.Wallets.Balances(ctx, walletID)
	must(t, "Wallets.Balances", err)
	need, _ := strconv.ParseFloat(amount, 64)
	for _, b := range bals {
		v, _ := strconv.ParseFloat(string(b.Balance), 64)
		a := b.Asset.Asset
		switch {
		case a.Symbol == "USDC" && v >= need && !haveUSDC:
			usdc, haveUSDC = b, true
		case a.IsNative || a.Address == "0x0000000000000000000000000000000000000000" || a.Standard == "":
			if v > 0 {
				native, haveGas = b, true
			}
		}
	}
	return usdc, native, haveUSDC, haveGas
}

func nativeSymbol(w *Wallet) string {
	if w.Blockchain != nil {
		return strings.ToUpper(w.Blockchain.Symbol)
	}
	return "native coin"
}

func pollTx(ctx context.Context, t *testing.T, what string, get func() (*TransactionDetail, error)) *TransactionDetail {
	t.Helper()
	deadline := time.Now().Add(10 * time.Minute)
	for {
		tx, err := get()
		must(t, what+" poll", err)
		if tx.Status == "SUCCESS" || tx.Status == "FAILED" || tx.Status == "CANCELLED" {
			return tx
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s still %s after 10m", what, tx.Status)
		}
		t.Logf("%s: %s, waiting…", what, tx.Status)
		sleep(ctx, 10*time.Second)
	}
}

type hookRecord struct {
	Event string          `json:"event"`
	Valid bool            `json:"valid"`
	Error string          `json:"error"`
	Body  json.RawMessage `json:"body"`
}

func readWebhooks(t *testing.T, path string) []hookRecord {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	must(t, "open webhook log", err)
	defer f.Close()
	var out []hookRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	for sc.Scan() {
		var r hookRecord
		if json.Unmarshal(sc.Bytes(), &r) == nil {
			out = append(out, r)
		}
	}
	return out
}

// webhookTxID decodes the delivery with the SDK types and returns data.id.
func webhookTxID(t *testing.T, r hookRecord) string {
	var ev WebhookEvent
	if json.Unmarshal(r.Body, &ev) != nil {
		return ""
	}
	var tx Transaction
	if err := json.Unmarshal(ev.Data, &tx); err != nil {
		t.Errorf("%s webhook: data does not decode into Transaction: %v", ev.Event, err)
		return ""
	}
	return tx.ID
}

func countWebhooks(t *testing.T, path, event, txID string) int {
	n := 0
	for _, r := range readWebhooks(t, path) {
		if r.Event == event && r.Valid && webhookTxID(t, r) == txID {
			n++
		}
	}
	return n
}

func waitWebhooks(ctx context.Context, t *testing.T, path string, want map[string]string, wait time.Duration) map[string]bool {
	got := map[string]bool{}
	deadline := time.Now().Add(wait)
	for {
		for _, r := range readWebhooks(t, path) {
			id := webhookTxID(t, r)
			for ev, txID := range want {
				if r.Event == ev && id == txID {
					if !r.Valid {
						t.Errorf("%s webhook for %s arrived with an INVALID signature: %s", ev, txID, r.Error)
						continue
					}
					if !got[ev] {
						t.Logf("webhook %s for %s received, signature valid", ev, txID)
					}
					got[ev] = true
				}
			}
		}
		if len(got) == len(want) || time.Now().After(deadline) {
			return got
		}
		sleep(ctx, 5*time.Second)
	}
}

func must(t *testing.T, what string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

// refused logs API refusals (feature off, not supported on this chain) without failing;
// anything else fails. It reports whether the call did not succeed.
func refused(t *testing.T, what string, err error) bool {
	t.Helper()
	var apiErr *APIError
	switch {
	case err == nil:
		return false
	case errors.As(err, &apiErr):
		t.Logf("%s refused by API: %v", what, err)
	default:
		t.Errorf("%s: %v", what, err)
	}
	return true
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func short(s string) string {
	if len(s) > 24 {
		return s[:24] + "…"
	}
	return s
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func balancesText(bs []Balance) string {
	var p []string
	for _, b := range bs {
		p = append(p, fmt.Sprintf("%s %s", b.Balance, b.Asset.Asset.Symbol))
	}
	return strings.Join(p, ", ")
}
