package blockradar

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// ContractsService calls the v1 endpoints in this file.
type ContractsService struct{ c *Client }

// ---- requests ----

// ContractCall is one smart contract method call.
type ContractCall struct {
	Address    string         `json:"address"` // contract address
	Method     string         `json:"method"`
	Parameters []any          `json:"parameters"` // use []any{} for no arguments
	ABI        []any          `json:"abi"`        // ABI entries of the contract or method
	Reference  string         `json:"reference,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ContractRequest is a single call (set ContractCall) or a batch (set Calls, max 20).
type ContractRequest struct {
	*ContractCall
	Calls []ContractCall `json:"calls,omitempty"`
	Code  string         `json:"code,omitempty"` // 2FA code, required for batch writes
}

// ---- responses ----

// ContractBatchError is a failed item of a batch.
type ContractBatchError struct {
	Index  int    `json:"index"`
	Method string `json:"method"`
	Error  string `json:"error"`
}

// ContractReadResult holds a single read's Value, or Success/Errors for a batch.
type ContractReadResult struct {
	Value   json.RawMessage      `json:"-"` // single call: the returned data as-is, e.g. "10052335235393043"
	Success []ContractReadItem   `json:"success"`
	Errors  []ContractBatchError `json:"errors"`
}

// ContractReadItem is one successful read of a batch.
type ContractReadItem struct {
	Index  int             `json:"index"`
	Method string          `json:"method"`
	Data   json.RawMessage `json:"data"`
}

// UnmarshalJSON treats an object with "success" or "errors" as a batch result.
func (r *ContractReadResult) UnmarshalJSON(b []byte) error {
	if bytes.HasPrefix(bytes.TrimSpace(b), []byte("{")) {
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(b, &probe); err == nil {
			_, s := probe["success"]
			_, e := probe["errors"]
			if s || e {
				type plain ContractReadResult
				return json.Unmarshal(b, (*plain)(r))
			}
		}
	}
	r.Value = append(json.RawMessage(nil), b...)
	return nil
}

// ContractTransaction is a contract write record; SignedTransaction is set for sign-only.
type ContractTransaction struct {
	Transaction
	SignedTransaction string `json:"signedTransaction"`
	Index             int    `json:"index"`  // batch items only
	Method            string `json:"method"` // batch items only
}

// ContractWriteResult is the transaction (single call) or Success/Errors (batch).
type ContractWriteResult struct {
	ContractTransaction
	Success []ContractTransaction `json:"success"`
	Errors  []ContractBatchError  `json:"errors"`
}

// ContractNetworkFee is a contract call fee estimate; Fees and TotalFee are set for batches.
type ContractNetworkFee struct {
	Balance              Amount               `json:"balance"`
	Fee                  Amount               `json:"fee"`
	Fees                 []ContractFee        `json:"fees"`
	TotalFee             Amount               `json:"totalFee"`
	EstimatedArrivalTime int                  `json:"estimatedArrivalTime"` // seconds
	Errors               []ContractBatchError `json:"errors"`
}

// ContractFee is the fee estimate for one call of a batch.
type ContractFee struct {
	Index                int    `json:"index"`
	Method               string `json:"method"`
	Balance              Amount `json:"balance"`
	Fee                  Amount `json:"fee"`
	NetworkFee           Amount `json:"networkFee"`
	NetworkFeeInUSD      Amount `json:"networkFeeInUSD"`
	NativeBalance        Amount `json:"nativeBalance"`
	NativeBalanceInUSD   Amount `json:"nativeBalanceInUSD"`
	TransactionFee       Amount `json:"transactionFee"`
	EstimatedArrivalTime int    `json:"estimatedArrivalTime"`
}

// ---- master wallet ----

// Read calls a view method with the master wallet.
func (s *ContractsService) Read(ctx context.Context, walletID string, r ContractRequest) (*ContractReadResult, error) {
	return call[ContractReadResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/contracts/read", nil, r)
}

// Write executes a contract method from the master wallet.
func (s *ContractsService) Write(ctx context.Context, walletID string, r ContractRequest) (*ContractWriteResult, error) {
	return call[ContractWriteResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/contracts/write", nil, r)
}

// WriteSign signs a contract write from the master wallet without broadcasting it.
func (s *ContractsService) WriteSign(ctx context.Context, walletID string, r ContractRequest) (*ContractWriteResult, error) {
	return call[ContractWriteResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/contracts/write/sign", nil, r)
}

// NetworkFee estimates the fee of a master wallet contract write.
func (s *ContractsService) NetworkFee(ctx context.Context, walletID string, r ContractRequest) (*ContractNetworkFee, error) {
	return call[ContractNetworkFee](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/contracts/network-fee", nil, r)
}

// ---- child address ----

// AddressRead calls a view method with a child address.
func (s *ContractsService) AddressRead(ctx context.Context, walletID, addressID string, r ContractRequest) (*ContractReadResult, error) {
	return call[ContractReadResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/contracts/read", nil, r)
}

// AddressWrite executes a contract method from a child address.
func (s *ContractsService) AddressWrite(ctx context.Context, walletID, addressID string, r ContractRequest) (*ContractWriteResult, error) {
	return call[ContractWriteResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/contracts/write", nil, r)
}

// AddressWriteSign signs a contract write from a child address without broadcasting it.
func (s *ContractsService) AddressWriteSign(ctx context.Context, walletID, addressID string, r ContractRequest) (*ContractWriteResult, error) {
	return call[ContractWriteResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/contracts/write/sign", nil, r)
}

// AddressNetworkFee estimates the fee of a child address contract write.
func (s *ContractsService) AddressNetworkFee(ctx context.Context, walletID, addressID string, r ContractRequest) (*ContractNetworkFee, error) {
	// The docs list this path without the /v1 prefix; every sibling uses /v1.
	return call[ContractNetworkFee](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/contracts/network-fee", nil, r)
}
