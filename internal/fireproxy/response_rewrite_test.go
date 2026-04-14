package fireproxy

import (
	"strings"
	"testing"
)

func TestRewriteJSONModels(t *testing.T) {
	body := []byte(`{"model":"accounts/fireworks/routers/kimi-k2p5-turbo","output":[{"model":"accounts/fireworks/routers/kimi-k2p5-turbo"}]}`)
	rewritten, err := rewriteJSONModels(body, testConfig())
	if err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}
	if strings.Contains(string(rewritten), "accounts/fireworks/routers/kimi-k2p5-turbo") {
		t.Fatalf("upstream model id still present: %s", string(rewritten))
	}
}

func TestRewriteJSONModelsPreservesReasoningContent(t *testing.T) {
	body := []byte(`{"object":"chat.completion.chunk","choices":[{"delta":{"reasoning_content":"hello"}}]}`)
	rewritten, err := rewriteJSONModels(body, testConfig())
	if err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}
	if !strings.Contains(string(rewritten), `"reasoning_content":"hello"`) {
		t.Fatalf("reasoning content missing: %s", string(rewritten))
	}
	if strings.Contains(string(rewritten), `"content":"hello"`) {
		t.Fatalf("unexpected content rewrite: %s", string(rewritten))
	}
}
