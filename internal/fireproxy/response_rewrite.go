package fireproxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func buildModelListResponse(cfg Config) map[string]any {
	return map[string]any{
		"object": "list",
		"data": []map[string]any{{
			"id":       cfg.PublicModelID,
			"object":   "model",
			"created":  0,
			"owned_by": "fireproxy",
		}},
	}
}

func rewriteUpstreamResponse(resp *http.Response, cfg Config) error {
	if resp.Body == nil {
		return nil
	}

	contentType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	switch contentType {
	case "application/json":
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()

		rewritten, err := rewriteJSONModels(body, cfg)
		if err != nil {
			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
			resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
			return nil
		}

		resp.Body = io.NopCloser(bytes.NewReader(rewritten))
		resp.ContentLength = int64(len(rewritten))
		resp.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))
	case "text/event-stream":
		resp.Body = newModelRewriteReadCloser(resp.Body, cfg)
		resp.ContentLength = -1
		resp.Header.Del("Content-Length")
	}
	return nil
}

func rewriteJSONModels(body []byte, cfg Config) ([]byte, error) {
	var payload any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	rewriteModelFields(payload, cfg)
	return json.Marshal(payload)
}

func rewriteModelFields(value any, cfg Config) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if key == "model" {
				if model, ok := nested.(string); ok {
					typed[key] = cfg.NormalizeOutgoingModel(model)
					continue
				}
			}
			rewriteModelFields(nested, cfg)
		}
	case []any:
		for _, nested := range typed {
			rewriteModelFields(nested, cfg)
		}
	}
}
