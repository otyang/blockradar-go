package blockradar

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"net/http/httptest"
	"testing"
)

func sign(key string, body []byte) string {
	m := hmac.New(sha512.New, []byte(key))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

func TestParseWebhook(t *testing.T) {
	body := []byte(`{"event":"deposit.success","data":{"id":"tx1","amount":"7.0","status":"SUCCESS","type":"DEPOSIT"}}`)

	r := httptest.NewRequest("POST", "/hook", bytes.NewReader(body))
	r.Header.Set("X-Blockradar-Signature", sign("key", body))
	ev, err := ParseWebhook(r, "key")
	if err != nil || ev.Event != "deposit.success" {
		t.Fatalf("ev=%+v err=%v", ev, err)
	}

	r = httptest.NewRequest("POST", "/hook", bytes.NewReader(body))
	r.Header.Set("X-Blockradar-Signature", sign("other-key", body))
	if _, err := ParseWebhook(r, "key"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("err = %v", err)
	}

	r = httptest.NewRequest("POST", "/hook", bytes.NewReader(body))
	if _, err := ParseWebhook(r, "key"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("missing header: err = %v", err)
	}
}

func TestAmountAcceptsNumberAndString(t *testing.T) {
	var v struct{ A, B, C Amount }
	if err := jsonUnmarshal(`{"A":"7.0","B":58.88822990007824,"C":null}`, &v); err != nil {
		t.Fatal(err)
	}
	if v.A != "7.0" || v.B != "58.88822990007824" || v.C != "" {
		t.Fatalf("%+v", v)
	}
}
