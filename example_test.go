package blockradar_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	blockradar "github.com/otyang/blockradar-go"
)

// Off-ramp: pay 100 USDC out as NGN to a bank account from the master wallet.
func Example_withdrawFiat() {
	ctx := context.Background()
	c := blockradar.NewClient(os.Getenv("BLOCKRADAR_API_KEY"))
	walletID := os.Getenv("BLOCKRADAR_WALLET_ID")

	// 1. Rate + session. The session fixes asset, currency, amount and provider.
	rate, err := c.WithdrawFiat.Rates(ctx, walletID, blockradar.WithdrawFiatRatesParams{
		AssetID:  "ae455f23-3824-4125-baab-d158315cbcbd", // from SupportedAssets
		Currency: "NGN",
		Amount:   "100",
	})
	if err != nil {
		log.Fatal(err)
	}
	session := blockradar.WithdrawFiatFlowParams{SessionID: rate.SessionID}

	// 2. Pick a payment method; its schema says which fields to collect.
	methods, err := c.WithdrawFiat.PaymentMethods(ctx, walletID, session)
	if err != nil {
		log.Fatal(err)
	}
	method := methods.PaymentMethods[0]

	// 3. Quote, then execute with the same request.
	q := blockradar.WithdrawFiatQuoteRequest{
		SessionID:     rate.SessionID,
		PaymentMethod: method.ID,
		PaymentMethodData: map[string]any{ // fields from the requirements schema
			"accountNumber": "0123456789",
			"bankCode":      "058",
		},
	}
	quote, err := c.WithdrawFiat.Quote(ctx, walletID, q)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("recipient gets", quote.AmountFiat, "NGN")

	tx, err := c.WithdrawFiat.Execute(ctx, walletID, blockradar.WithdrawFiatExecuteRequest{
		WithdrawFiatQuoteRequest: q,
		Reference:                "payout-order-42",
	})
	var apiErr *blockradar.APIError
	switch {
	case errors.As(err, &apiErr) && apiErr.Code() == blockradar.CodeWithdrawalSessionExpired:
		// Start again from Rates.
	case err != nil:
		log.Fatal(err)
	default:
		fmt.Println("withdrawal", tx.ID, tx.Status)
	}
}

// On-ramp: give a user an NGN bank account that settles into their child address.
func Example_virtualAccount() {
	ctx := context.Background()
	c := blockradar.NewClient(os.Getenv("BLOCKRADAR_API_KEY"))
	walletID, addressID := os.Getenv("BLOCKRADAR_WALLET_ID"), "child-address-id"

	req, err := c.VirtualAccounts.Requirements(ctx, walletID, blockradar.VirtualAccountRequirementsParams{Currency: "NGN"})
	if err != nil {
		log.Fatal(err)
	}

	res, err := c.VirtualAccounts.AddressCreate(ctx, walletID, addressID, blockradar.CreateVirtualAccountRequest{
		Provider:       req.Provider,
		Currency:       "NGN",
		AdditionalData: map[string]any{"bvn": "12345678901"}, // only if req.AdditionalDataRequired
		Label:          "Ada collection account",
	})
	if err != nil {
		log.Fatal(err)
	}
	va := res.VirtualAccount
	fmt.Println("pay into", va.BankName, va.AccountNumber, va.AccountName)
}
