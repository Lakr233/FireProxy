package fireproxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type validationError struct {
	Code    string
	Status  int
	Message string
}

func (e validationError) Error() string {
	return e.Message
}

func (s *Server) validateJSONRequest(r *http.Request, validator func(Config, map[string]any) error) error {
	if query := r.URL.RawQuery; query != "" {
		return validationError{Code: "invalid_query", Status: http.StatusBadRequest, Message: "query string is not supported for this endpoint"}
	}
	if err := validateContentType(r.Header); err != nil {
		return validationError{Code: "invalid_content_type", Status: http.StatusBadRequest, Message: err.Error()}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, s.cfg.BodyLimitBytes+1))
	if err != nil {
		return validationError{Code: "invalid_body", Status: http.StatusBadRequest, Message: "request body could not be read"}
	}
	defer r.Body.Close()

	if int64(len(body)) > s.cfg.BodyLimitBytes {
		return validationError{Code: "body_too_large", Status: http.StatusRequestEntityTooLarge, Message: "request body exceeds limit"}
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return validationError{Code: "invalid_body", Status: http.StatusBadRequest, Message: "request body is required"}
	}

	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return validationError{Code: "invalid_json", Status: http.StatusBadRequest, Message: "request body must be valid JSON"}
	}
	if decoder.More() {
		return validationError{Code: "invalid_json", Status: http.StatusBadRequest, Message: "request body must contain one JSON object"}
	}
	if err := validator(s.cfg, payload); err != nil {
		return err
	}

	rewrittenBody, err := json.Marshal(payload)
	if err != nil {
		return validationError{Code: "invalid_body", Status: http.StatusBadRequest, Message: "request body could not be normalized"}
	}

	r.Body = io.NopCloser(bytes.NewReader(rewrittenBody))
	r.ContentLength = int64(len(rewrittenBody))
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(rewrittenBody)), nil
	}
	return nil
}

func (s *Server) handleValidationError(w http.ResponseWriter, err error) {
	var vErr validationError
	if errors.As(err, &vErr) {
		writeError(w, vErr.Status, vErr.Code, vErr.Message)
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
}

func validateAuthorization(header http.Header, cfg Config) error {
	value := strings.TrimSpace(header.Get("Authorization"))
	if value == "" {
		return fmt.Errorf("authorization header is required")
	}
	if !strings.HasPrefix(value, "Bearer ") {
		return fmt.Errorf("authorization header must use Bearer token")
	}

	token := strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
	if token == "" {
		return fmt.Errorf("bearer token is required")
	}
	if len(cfg.APIKeys) > 0 {
		if _, ok := cfg.APIKeys[strings.ToLower(token)]; ok {
			return nil
		}
		return fmt.Errorf("bearer token is not allowed")
	}
	if cfg.UpstreamAPIKey != "" {
		return nil
	}
	if strings.HasPrefix(token, "fw-") || strings.HasPrefix(token, "fw_") {
		return nil
	}
	return fmt.Errorf("bearer token must start with fw- or fw_")
}

func validateContentType(header http.Header) error {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(header.Get("Content-Type"), ";")[0]))
	if mediaType != "application/json" {
		return fmt.Errorf("content-type must be application/json")
	}
	return nil
}

func validateListResponsesQuery(values url.Values) error {
	allowed := map[string]struct{}{"after": {}, "before": {}, "limit": {}}
	for key, value := range values {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("query parameter %q is not allowed", key)
		}
		if len(value) != 1 {
			return fmt.Errorf("query parameter %q must appear once", key)
		}
	}
	if limit := values.Get("limit"); limit != "" {
		parsed, err := strconv.Atoi(limit)
		if err != nil || parsed < 1 || parsed > 100 {
			return fmt.Errorf("limit must be an integer between 1 and 100")
		}
	}
	return nil
}

func rewriteModelInBody(cfg Config, payload map[string]any) error {
	model, err := requireNonEmptyString(payload, "model")
	if err != nil {
		return err
	}
	payload["model"] = cfg.NormalizeIncomingModel(model)
	return validateModelAllowlist(cfg, model)
}

func requireNonEmptyString(payload map[string]any, field string) (string, error) {
	value, ok := payload[field]
	if !ok {
		return "", validationError{Code: "missing_field", Status: http.StatusBadRequest, Message: fmt.Sprintf("field %q is required", field)}
	}
	str, ok := value.(string)
	if !ok || strings.TrimSpace(str) == "" {
		return "", validationError{Code: "invalid_field", Status: http.StatusBadRequest, Message: fmt.Sprintf("field %q must be a non-empty string", field)}
	}
	return strings.TrimSpace(str), nil
}

func validateModelAllowlist(cfg Config, model string) error {
	model = cfg.NormalizeIncomingModel(model)
	if len(cfg.AllowedModels) == 0 {
		return nil
	}
	if _, ok := cfg.AllowedModels[strings.ToLower(model)]; ok {
		return nil
	}
	return validationError{Code: "unsupported_model", Status: http.StatusBadRequest, Message: fmt.Sprintf("model %q is not allowed", model)}
}
