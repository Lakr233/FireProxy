package fireproxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRewriteModelInBodyResponsePayload(t *testing.T) {
	cfg := testConfig()

	valid := map[string]any{
		"model":  "kimi-k2p5-turbo",
		"input":  "hello",
		"stream": true,
		"foo":    "bar",
	}
	if err := rewriteModelInBody(cfg, valid); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if got := valid["model"]; got != "accounts/fireworks/routers/kimi-k2p5-turbo" {
		t.Fatalf("model remap failed: %v", got)
	}

	unsupportedModel := map[string]any{
		"model": "accounts/fireworks/models/other",
		"input": "hello",
	}
	if err := rewriteModelInBody(cfg, unsupportedModel); err == nil {
		t.Fatalf("expected unsupported model error")
	}
}

func TestRewriteModelInBody(t *testing.T) {
	cfg := testConfig()

	valid := map[string]any{"model": "kimi-k2p5-turbo", "prompt": "hello", "foo": "bar"}
	if err := rewriteModelInBody(cfg, valid); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if got := valid["model"]; got != "accounts/fireworks/routers/kimi-k2p5-turbo" {
		t.Fatalf("model remap failed: %v", got)
	}
}

func TestRewriteModelInBodyRequiresModel(t *testing.T) {
	if err := rewriteModelInBody(testConfig(), map[string]any{"messages": []any{}}); err == nil {
		t.Fatalf("expected missing model error")
	}
}

func TestServerRejectsInvalidAuthorization(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/inference/v1/responses", strings.NewReader(`{"model":"accounts/fireworks/routers/kimi-k2p5-turbo","input":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer bad-key")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServerAcceptsUnderscoreAuthorization(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/inference/v1/models", nil)
	req.Header.Set("Authorization", "Bearer fp_test_key")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestServerAllowsUnknownField(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/inference/v1/completions", strings.NewReader(`{"model":"kimi-k2p5-turbo","prompt":"hi","foo":"bar"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer fp_test_key")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code == http.StatusBadRequest {
		t.Fatalf("unexpected rejection: %s", rec.Body.String())
	}
}

func TestValidateListResponsesQuery(t *testing.T) {
	if err := validateListResponsesQuery(url.Values{"limit": {"20"}}); err != nil {
		t.Fatalf("valid query rejected: %v", err)
	}
	if err := validateListResponsesQuery(url.Values{"foo": {"bar"}}); err == nil {
		t.Fatalf("expected error for unknown query")
	}
}

func TestModelListEndpoint(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/inference/v1/models", nil)
	req.Header.Set("Authorization", "Bearer fp_test_key")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"id":"kimi-k2p5-turbo"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
