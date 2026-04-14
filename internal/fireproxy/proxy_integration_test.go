package fireproxy

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestProxyStreamSendReceiveModelMapping(t *testing.T) {
	upstreamURL, err := url.Parse("https://api.fireworks.ai")
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}
	server, err := NewServer(testConfigWithUpstream(upstreamURL, "fw_upstream_secret"), log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	server.proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		t.Fatalf("proxy error: %v", err)
	}
	server.proxy.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer fw_upstream_secret" {
			t.Fatalf("unexpected upstream auth header: %q", got)
		}
		if !strings.Contains(string(body), `"model":"accounts/fireworks/routers/kimi-k2p5-turbo"`) {
			t.Fatalf("request body missing upstream model mapping: %s", string(body))
		}
		if strings.Contains(string(body), `"model":"kimi-k2p5-turbo"`) {
			t.Fatalf("request body still contains public model id: %s", string(body))
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: &chunkedReadCloser{
				chunks: [][]byte{
					[]byte("data: {\"type\":\"response.created\",\"model\":\"accounts/fireworks/ro"),
					[]byte("uters/kimi-k2p5-turbo\"}\n\n"),
					[]byte("data: [DONE]\n"),
				},
			},
			Request: r,
		}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/inference/v1/responses", strings.NewReader(`{"model":"kimi-k2p5-turbo","input":"hi","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer fp_test_key")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"model":"kimi-k2p5-turbo"`) {
		t.Fatalf("stream body missing public model id: %s", body)
	}
	if body := rec.Body.String(); strings.Contains(body, "accounts/fireworks/routers/kimi-k2p5-turbo") {
		t.Fatalf("stream body still contains upstream model id: %s", body)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type chunkedReadCloser struct {
	chunks [][]byte
}

func (r *chunkedReadCloser) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	chunk := r.chunks[0]
	r.chunks = r.chunks[1:]
	n := copy(p, chunk)
	return n, nil
}

func (r *chunkedReadCloser) Close() error {
	return nil
}
