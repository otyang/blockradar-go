package blockradar

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// SignatureHeader carries the HMAC-SHA512 (hex) of the raw webhook body,
// keyed with the wallet's API key.
const SignatureHeader = "x-blockradar-signature"

// ErrInvalidSignature is returned when a webhook signature does not match.
var ErrInvalidSignature = errors.New("blockradar: invalid webhook signature")

// VerifySignature reports whether signature matches body for apiKey.
// Pass the raw request body, not a re-encoded one.
func VerifySignature(apiKey string, body []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(apiKey))
	mac.Write(body)
	want := mac.Sum(nil)
	got, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return false
	}
	return hmac.Equal(want, got)
}

// WebhookEvent is a webhook delivery: {"event": "deposit.success", "data": {...}}.
// Data is kept raw; decode it into the type that fits the event, e.g.
// Transaction for deposit.* and withdraw.* events.
type WebhookEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// ParseWebhook reads r's body, checks its signature and decodes the event.
// It returns ErrInvalidSignature if the signature is missing or wrong.
//
//	http.HandleFunc("/webhooks/blockradar", func(w http.ResponseWriter, r *http.Request) {
//		ev, err := blockradar.ParseWebhook(r, apiKey)
//		if err != nil { http.Error(w, "bad signature", http.StatusUnauthorized); return }
//		w.WriteHeader(http.StatusOK)
//		if ev.Event == "deposit.success" { var tx blockradar.Transaction; json.Unmarshal(ev.Data, &tx) ... }
//	})
func ParseWebhook(r *http.Request, apiKey string) (*WebhookEvent, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 5<<20))
	if err != nil {
		return nil, err
	}
	if !VerifySignature(apiKey, body, r.Header.Get(SignatureHeader)) {
		return nil, ErrInvalidSignature
	}
	var ev WebhookEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}
