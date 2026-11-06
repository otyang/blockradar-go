package blockradar

import (
	"context"
	"net/http"
	"net/url"
)

// AMLService calls the v1 endpoints in this file.
type AMLService struct{ c *Client }

// AMLLookup is the result of an AML lookup.
type AMLLookup struct {
	IsBlacklisted bool `json:"isBlacklisted"`
}

// Lookup screens address on blockchain (a slug such as "ethereum") against AML lists.
func (s *AMLService) Lookup(ctx context.Context, address, blockchain string) (*AMLLookup, error) {
	q := url.Values{}
	setStr(q, "address", address)
	setStr(q, "blockchain", blockchain)
	return call[AMLLookup](ctx, s.c, http.MethodGet, "/v1/aml/lookup", q, nil)
}
