package fireproxy

import (
	"io"
	"strings"
	"testing"
)

func TestRewriteStreamModels(t *testing.T) {
	stream := io.NopCloser(strings.NewReader("data: {\"model\":\"accounts/fireworks/routers/kimi-k2p5-turbo\"}\n\ndata: [DONE]\n"))
	rewriter := newModelRewriteReadCloser(stream, testConfig())
	defer rewriter.Close()

	body, err := io.ReadAll(rewriter)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if strings.Contains(string(body), "accounts/fireworks/routers/kimi-k2p5-turbo") {
		t.Fatalf("upstream model id still present: %s", string(body))
	}
}

func TestRewriteStreamModelsAlias(t *testing.T) {
	stream := io.NopCloser(strings.NewReader("data: {\"model\":\"accounts/fireworks/models/kimi-k2p5\"}\n\ndata: [DONE]\n"))
	rewriter := newModelRewriteReadCloser(stream, testConfig())
	defer rewriter.Close()

	body, err := io.ReadAll(rewriter)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if strings.Contains(string(body), "accounts/fireworks/models/kimi-k2p5") {
		t.Fatalf("upstream alias still present: %s", string(body))
	}
	if !strings.Contains(string(body), "kimi-k2p5-turbo") {
		t.Fatalf("public model id missing: %s", string(body))
	}
}
