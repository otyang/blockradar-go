package blockradar

import (
	"context"
	"encoding/json"
	"net/http"
)

// SigningService calls the v1 endpoints in this file.
type SigningService struct{ c *Client }

// ---- requests ----

// SignMessageRequest signs a plain-text message.
type SignMessageRequest struct {
	Message  string `json:"message"`
	Metadata string `json:"metadata,omitempty"` // documented as a string
	Note     string `json:"note,omitempty"`
}

// SignTypedDataRequest signs EIP-712 typed data (EVM only).
type SignTypedDataRequest struct {
	Domain  SignTypedDataDomain             `json:"domain"`
	Types   map[string][]SignTypedDataField `json:"types"`
	Message map[string]any                  `json:"message"`
}

// SignTypedDataDomain is the EIP-712 domain. Every field is optional in EIP-712
// (e.g. Permit2 has no version); unset fields are left out of the domain.
type SignTypedDataDomain struct {
	Name              string      `json:"name,omitempty"`
	Version           string      `json:"version,omitempty"`
	ChainID           json.Number `json:"chainId,omitempty"`
	VerifyingContract string      `json:"verifyingContract,omitempty"`
	Salt              string      `json:"salt,omitempty"`
}

// SignTypedDataField is one field of an EIP-712 type, e.g. {Name: "owner", Type: "address"}.
type SignTypedDataField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// SignTransactionRequest signs (or signs and broadcasts) a raw unsigned transaction.
// Transaction is a JSON string for EVM and Tron, base64 for Solana and base64 XDR for Stellar.
type SignTransactionRequest struct {
	Transaction string         `json:"transaction"`
	Reference   string         `json:"reference,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ---- responses ----

// SignResult is the SIGNED transaction record returned by the signing endpoints.
type SignResult struct {
	Transaction
	SignedTransaction SignedTransaction `json:"signedTransaction"`
}

// SignedTransaction holds a signed raw transaction (Raw) or a message/typed-data signature.
type SignedTransaction struct {
	Raw       string `json:"-"` // set when the API returns a string
	Signature string `json:"signature,omitempty"`
	R         string `json:"r,omitempty"`
	S         string `json:"s,omitempty"`
	V         int    `json:"v,omitempty"`
}

func (t *SignedTransaction) UnmarshalJSON(b []byte) error {
	switch {
	case string(b) == "null":
		return nil
	case len(b) > 0 && b[0] == '"':
		return json.Unmarshal(b, &t.Raw)
	}
	type plain SignedTransaction
	return json.Unmarshal(b, (*plain)(t))
}

// ---- master wallet ----

// Message signs a message with the master wallet key.
func (s *SigningService) Message(ctx context.Context, walletID string, r SignMessageRequest) (*SignResult, error) {
	return call[SignResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/signing/message", nil, r)
}

// TypedData signs EIP-712 typed data with the master wallet key.
func (s *SigningService) TypedData(ctx context.Context, walletID string, r SignTypedDataRequest) (*SignResult, error) {
	return call[SignResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/signing/typed-data", nil, r)
}

// Transaction signs a raw transaction with the master wallet key without broadcasting it.
func (s *SigningService) Transaction(ctx context.Context, walletID string, r SignTransactionRequest) (*SignResult, error) {
	return call[SignResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/signing/transaction", nil, r)
}

// Broadcast signs a raw transaction with the master wallet key and queues it for broadcast (status PENDING).
func (s *SigningService) Broadcast(ctx context.Context, walletID string, r SignTransactionRequest) (*SignResult, error) {
	return call[SignResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/signing/broadcast", nil, r)
}

// ---- child address ----

// AddressMessage signs a message with a child address key.
func (s *SigningService) AddressMessage(ctx context.Context, walletID, addressID string, r SignMessageRequest) (*SignResult, error) {
	return call[SignResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/signing/message", nil, r)
}

// AddressTypedData signs EIP-712 typed data with a child address key.
func (s *SigningService) AddressTypedData(ctx context.Context, walletID, addressID string, r SignTypedDataRequest) (*SignResult, error) {
	return call[SignResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/signing/typed-data", nil, r)
}

// AddressTransaction signs a raw transaction with a child address key without broadcasting it.
func (s *SigningService) AddressTransaction(ctx context.Context, walletID, addressID string, r SignTransactionRequest) (*SignResult, error) {
	return call[SignResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/signing/transaction", nil, r)
}

// AddressBroadcast signs a raw transaction with a child address key and queues it for broadcast.
func (s *SigningService) AddressBroadcast(ctx context.Context, walletID, addressID string, r SignTransactionRequest) (*SignResult, error) {
	return call[SignResult](ctx, s.c, http.MethodPost, "/v1/wallets/"+seg(walletID)+"/addresses/"+seg(addressID)+"/signing/broadcast", nil, r)
}
