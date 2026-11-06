package blockradar

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOnboardingRoutes(t *testing.T) {
	c, routes := recordRoutes(t, "")
	ctx := context.Background()
	o := c.Onboarding
	data := OnboardingDataRequest{Data: map[string]any{}}
	front := File{Filename: "f.jpg", Content: strings.NewReader("x")}
	calls := []func() error{
		func() error { _, err := o.Create(ctx, "C", CreateOnboardingRequest{}); return err },
		func() error { _, err := o.List(ctx, "C", OnboardingListParams{}); return err },
		func() error { _, err := o.Get(ctx, "C", "K"); return err },
		func() error { _, err := o.SaveDraft(ctx, "C", "K", OnboardingDraftRequest{}); return err },
		func() error { _, err := o.Submit(ctx, "C", "K", data); return err },
		func() error { _, err := o.Refresh(ctx, "C", "K", OnboardingRefreshRequest{}); return err },
		func() error { _, err := o.Reopen(ctx, "C", "K"); return err },
		func() error {
			_, err := o.SubmitHostedSession(ctx, "C", "K", 2, OnboardingHostedSessionRequest{Submitted: true})
			return err
		},
		func() error { _, err := o.SubmitAction(ctx, "C", "K", "ACT", data); return err },
		func() error { _, err := o.ConfirmationRequirements(ctx, "C", "K"); return err },
		func() error { _, err := o.SubmitConfirmation(ctx, "C", "K", data); return err },
		func() error {
			_, err := o.PrepareUploads(ctx, "C", "K", OnboardingPrepareUploadsRequest{})
			return err
		},
		func() error { _, err := o.UploadDocument(ctx, "C", "K", "U", front, File{}); return err },
		func() error {
			_, err := o.DocumentViewURLs(ctx, "C", "K", OnboardingDocumentViewURLRequest{})
			return err
		},
	}
	for i, f := range calls {
		if err := f(); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	const base = "/v1/customers/C/onboarding-requests"
	assertRoutes(t, routes(), []string{
		"POST " + base,
		"GET " + base,
		"GET " + base + "/K",
		"PATCH " + base + "/K",
		"POST " + base + "/K/submit",
		"POST " + base + "/K/refresh",
		"POST " + base + "/K/reopen",
		"PATCH " + base + "/K/hosted-sessions/2/submission",
		"POST " + base + "/K/actions/ACT/submissions",
		"GET " + base + "/K/confirmation-requirements",
		"POST " + base + "/K/confirmation",
		"POST " + base + "/K/uploads/prepare",
		"POST " + base + "/K/uploads/U",
		"POST " + base + "/K/documents/view-url",
	})
}

func TestOnboardingCreateBody(t *testing.T) {
	c, got := server(t, 201, `{"status":true,"message":"ok","data":{"customerId":"c1",
		"case":{"id":"k1","status":"draft","mode":"api_only","createdAt":"2026-07-22T12:00:00.000Z","updatedAt":"2026-07-22T12:00:00.000Z"},
		"requirements":{"mode":"api_only","subjectType":"customer","feature":"offramp","status":"draft",
		"schema":{"$schema":"http://json-schema.org/draft-07/schema#","type":"object"}}}}`)
	res, err := c.Onboarding.Create(context.Background(), "c1", CreateOnboardingRequest{
		Provider: "infinia", Type: "offramp",
		RouteIntent: &OnboardingRouteIntent{SourceCurrencyCode: "NGN", TargetCurrencyCode: "USDC"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"provider": "infinia", "type": "offramp",
		"routeIntent": map[string]any{"sourceCurrencyCode": "NGN", "targetCurrencyCode": "USDC"}}
	if b1, b2 := mustJSON(got.body), mustJSON(want); b1 != b2 {
		t.Fatalf("body = %s, want %s", b1, b2)
	}
	if res.Case.ID != "k1" || res.Requirements == nil || !json.Valid(res.Requirements.Schema) {
		t.Fatalf("decoded %+v", res)
	}
}

func TestOnboardingListQueryAndDecode(t *testing.T) {
	c, got := server(t, 200, `{"status":true,"message":"ok","data":{
		"items":[{"id":"k1","status":"approved","mode":"hosted_only","availableActions":["edit_and_resubmit"],
		"createdAt":"2026-07-22T12:00:00.000Z","updatedAt":"2026-07-22T12:00:00.000Z"}],
		"meta":{"totalItems":1,"itemCount":1,"itemsPerPage":10,"totalPages":1,"currentPage":1},
		"links":{"first":"f","previous":"","next":"","last":"l"}}}`)
	list, err := c.Onboarding.List(context.Background(), "c1", OnboardingListParams{
		Page: 2, Limit: 5, Status: OnboardingApproved, Feature: "offramp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.query != "feature=offramp&limit=5&page=2&status=approved" {
		t.Fatalf("query = %s", got.query)
	}
	if len(list.Items) != 1 || list.Items[0].Status != OnboardingApproved || list.Meta.TotalItems != 1 ||
		list.Items[0].AvailableActions[0] != OnboardingEditAndResubmit {
		t.Fatalf("decoded %+v", list)
	}
}

func TestOnboardingRefreshSendsEmptyObject(t *testing.T) {
	// Documented "approved" example.
	c, got := server(t, 200, `{"status": true, "message": "Onboarding case approved successfully", "data": {"customerId": "18d2f159-e0cd-4a45-b735-e9dfe0f7520a", "case": {"id": "8392013d-2481-47c9-b5e3-b61b22b8ff82", "businessId": "ec01bc89-d2d8-47fc-8d4c-0616d859915d", "customerId": "18d2f159-e0cd-4a45-b735-e9dfe0f7520a", "providerId": "1128dd40-2c94-4c16-a6bf-fbbc6afee52c", "subjectType": "customer", "status": "approved", "mode": "hosted_only", "feature": "offramp", "hostedSessions": [], "actions": [], "availableActions": [], "reason": null, "createdAt": "2026-10-31T10:00:00.000Z", "updatedAt": "2026-10-31T10:15:00.000Z"}}}`)
	res, err := c.Onboarding.Refresh(context.Background(), "c1", "k1", OnboardingRefreshRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.body == nil || len(got.body) != 0 {
		t.Fatalf("body = %v, want {}", got.body)
	}
	if res.Case.Status != OnboardingApproved || res.Case.Feature != "offramp" || res.Requirements != nil || res.Case.UpdatedAt.IsZero() {
		t.Fatalf("decoded %+v", res)
	}
}

func TestOnboardingSubmitConfirmationDecodesExample(t *testing.T) {
	c, got := server(t, 200, `{"status": true, "message": "Onboarding confirmation submitted successfully", "data": {"customerId": "18d2f159-e0cd-4a45-b735-e9dfe0f7520a", "case": {"id": "71d6b5ac-6312-43eb-b389-789d214bbda2", "businessId": "ad1a5189-f47d-4201-9f77-b01c9d9c0270", "customerId": "18d2f159-e0cd-4a45-b735-e9dfe0f7520a", "providerId": "1128dd40-2c94-4c16-a6bf-fbbc6afee52c", "subjectType": "customer", "status": "approved", "mode": "api_only", "feature": "offramp", "providerReference": "provider_customer_839201", "createdAt": "2026-07-22T12:00:00.000Z", "updatedAt": "2026-07-22T12:15:00.000Z"}}}`)
	res, err := c.Onboarding.SubmitConfirmation(context.Background(), "c1", "k1", OnboardingDataRequest{Data: map[string]any{"email": "a@b.co"}})
	if err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"data":{"email":"a@b.co"}}` {
		t.Fatalf("body = %s", b)
	}
	if res.Case.ProviderReference != "provider_customer_839201" || res.Case.BusinessID != "ad1a5189-f47d-4201-9f77-b01c9d9c0270" {
		t.Fatalf("decoded %+v", res.Case)
	}
}

func TestOnboardingConfirmationRequirementsExample(t *testing.T) {
	c, _ := server(t, 200, `{"status": true, "message": "Onboarding confirmation requirements fetched successfully", "data": {"customerId": "18d2f159-e0cd-4a45-b735-e9dfe0f7520a", "caseId": "71d6b5ac-6312-43eb-b389-789d214bbda2", "schema": {"$schema": "http://json-schema.org/draft-07/schema#", "title": "Confirmation requirements", "type": "object", "additionalProperties": false, "properties": {"email": {"type": "string", "title": "Email", "format": "email"}}, "required": ["email"]}}}`)
	res, err := c.Onboarding.ConfirmationRequirements(context.Background(), "c1", "k1")
	if err != nil {
		t.Fatal(err)
	}
	if res.CaseID != "71d6b5ac-6312-43eb-b389-789d214bbda2" || !strings.Contains(string(res.Schema), `"required": ["email"]`) {
		t.Fatalf("decoded %+v", res)
	}
}

func TestOnboardingSubmitDecodesInitiated(t *testing.T) {
	c, got := server(t, 200, `{"status":true,"message":"ok","data":{"customerId":"c1","initiated":true,
		"case":{"id":"k1","status":"submitted","mode":"api_only","createdAt":"2026-07-22T12:00:00.000Z","updatedAt":"2026-07-22T12:00:00.000Z"}}}`)
	res, err := c.Onboarding.Submit(context.Background(), "c1", "k1", OnboardingDataRequest{Data: map[string]any{"firstName": "Ada"}})
	if err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"data":{"firstName":"Ada"}}` {
		t.Fatalf("body = %s", b)
	}
	if !res.Initiated || res.Case.Status != OnboardingSubmitted || res.CustomerID != "c1" {
		t.Fatalf("decoded %+v", res)
	}
}

func TestOnboardingPrepareUploadsExample(t *testing.T) {
	c, got := server(t, 200, `{"message": "Onboarding upload targets prepared successfully", "data": {"caseId": "71d6b5ac-6312-43eb-b389-789d214bbda2", "uploads": [{"field": "identityDocument", "document": {"mode": "proxy", "proxyUploadId": "upload_01K0PZV5X8JQ4V9N0P7K2M6C3F", "method": "POST", "providerReference": "document_839201", "documentType": "passport", "expiresAt": "2026-07-22T14:30:00.000Z", "sides": ["front"], "accept": ["image/jpeg", "image/png", "application/pdf"]}}]}, "status": true}`)
	res, err := c.Onboarding.PrepareUploads(context.Background(), "c1", "k1", OnboardingPrepareUploadsRequest{
		Files: []OnboardingUploadFile{{Field: "identityDocument", DocumentType: "passport", Sides: []string{"front"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if b := mustJSON(got.body); b != `{"files":[{"documentType":"passport","field":"identityDocument","sides":["front"]}]}` {
		t.Fatalf("body = %s", b)
	}
	d := res.Uploads[0].Document
	if res.Uploads[0].Field != "identityDocument" || d.ProxyUploadID != "upload_01K0PZV5X8JQ4V9N0P7K2M6C3F" ||
		d.Mode != "proxy" || d.ExpiresAt == nil || len(d.Accept) != 3 {
		t.Fatalf("decoded %+v", res)
	}
}

func TestOnboardingUploadDocumentMultipart(t *testing.T) {
	parts := map[string]string{}
	var path, contentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, contentType = r.URL.EscapedPath(), r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}
		for field, fhs := range r.MultipartForm.File {
			f, _ := fhs[0].Open()
			b, _ := io.ReadAll(f)
			f.Close()
			parts[field] = fhs[0].Filename + ":" + string(b)
		}
		_, _ = io.WriteString(w, `{"message": "Onboarding document uploaded successfully", "data": {"caseId": "71d6b5ac-6312-43eb-b389-789d214bbda2", "proxyUploadId": "upload_01K0PZV5X8JQ4V9N0P7K2M6C3F", "field": "identityDocument", "document": {"providerReference": "document_839201", "documentType": "passport", "uploadedSides": ["front"]}}, "status": true}`)
	}))
	defer srv.Close()
	c := NewClient("k", WithBaseURL(srv.URL))

	res, err := c.Onboarding.UploadDocument(context.Background(), "c1", "k1", "upload_1",
		File{Filename: "front.jpg", Content: strings.NewReader("FRONT")},
		File{Filename: "back.jpg", Content: strings.NewReader("BACK")})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v1/customers/c1/onboarding-requests/k1/uploads/upload_1" || !strings.HasPrefix(contentType, "multipart/form-data") {
		t.Fatalf("got %s %s", path, contentType)
	}
	if parts["front"] != "front.jpg:FRONT" || parts["back"] != "back.jpg:BACK" || len(parts) != 2 {
		t.Fatalf("parts = %v", parts)
	}
	if res.ProxyUploadID != "upload_01K0PZV5X8JQ4V9N0P7K2M6C3F" || res.Document.UploadedSides[0] != "front" {
		t.Fatalf("decoded %+v", res)
	}

	// Front only: a zero back File is not sent.
	parts = map[string]string{}
	if _, err := c.Onboarding.UploadDocument(context.Background(), "c1", "k1", "upload_1",
		File{Filename: "front.jpg", Content: strings.NewReader("FRONT")}, File{}); err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 || parts["front"] == "" {
		t.Fatalf("parts = %v", parts)
	}
}
