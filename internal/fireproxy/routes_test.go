package fireproxy

import (
	"net/http"
	"testing"
)

func TestMatchRoute(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		want    routeKind
		wantID  string
		wantErr bool
	}{
		{name: "create response", method: http.MethodPost, path: "/inference/v1/responses", want: routeCreateResponse},
		{name: "list models", method: http.MethodGet, path: "/inference/v1/models", want: routeListModels},
		{name: "chat completion", method: http.MethodPost, path: "/inference/v1/chat/completions", want: routeCreateChatCompletion},
		{name: "list responses", method: http.MethodGet, path: "/inference/v1/responses", want: routeListResponses},
		{name: "get response", method: http.MethodGet, path: "/inference/v1/responses/resp_123", want: routeGetResponse, wantID: "resp_123"},
		{name: "delete response", method: http.MethodDelete, path: "/inference/v1/responses/resp_123", want: routeDeleteResponse, wantID: "resp_123"},
		{name: "completion", method: http.MethodPost, path: "/inference/v1/completions", want: routeCreateCompletion},
		{name: "reject legacy openai path", method: http.MethodPost, path: "/v1/chat/completions", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotID, err := matchRoute(tt.method, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want || gotID != tt.wantID {
				t.Fatalf("got (%s, %q), want (%s, %q)", got, gotID, tt.want, tt.wantID)
			}
		})
	}
}
