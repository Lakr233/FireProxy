package fireproxy

import (
	"io"
	"log"
	"net/url"
	"testing"
)

func testConfig() Config {
	return Config{
		AllowedModels: map[string]struct{}{
			"kimi-k2p5-turbo": {},
			"accounts/fireworks/routers/kimi-k2p5-turbo": {},
		},
		APIKeys: map[string]struct{}{
			"fp_test_key": {},
		},
		PublicModelID:   "kimi-k2p5-turbo",
		UpstreamModelID: "accounts/fireworks/routers/kimi-k2p5-turbo",
		UpstreamModelAliases: map[string]struct{}{
			"accounts/fireworks/routers/kimi-k2p5-turbo": {},
			"accounts/fireworks/models/kimi-k2p5":        {},
		},
		BodyLimitBytes: defaultBodyLimitBytes,
	}
}

func testConfigWithUpstream(upstreamURL *url.URL, upstreamAPIKey string) Config {
	cfg := testConfig()
	cfg.UpstreamBaseURL = upstreamURL
	cfg.UpstreamAPIKey = upstreamAPIKey
	return cfg
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	upstreamURL, err := url.Parse("https://api.fireworks.ai")
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	server, err := NewServer(testConfigWithUpstream(upstreamURL, ""), log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return server
}
