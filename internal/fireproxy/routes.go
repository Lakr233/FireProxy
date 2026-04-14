package fireproxy

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
)

var responseIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)

type routeKind string

const (
	routeUnknown              routeKind = ""
	routeHealth               routeKind = "health"
	routeListModels           routeKind = "list_models"
	routeCreateChatCompletion routeKind = "create_chat_completion"
	routeCreateResponse       routeKind = "create_response"
	routeListResponses        routeKind = "list_responses"
	routeGetResponse          routeKind = "get_response"
	routeDeleteResponse       routeKind = "delete_response"
	routeCreateCompletion     routeKind = "create_completion"
)

func matchRoute(method, requestPath string) (routeKind, string, error) {
	cleaned := cleanPath(requestPath)
	switch {
	case method == http.MethodGet && cleaned == "/healthz":
		return routeHealth, "", nil
	case method == http.MethodGet && cleaned == "/inference/v1/models":
		return routeListModels, "", nil
	case method == http.MethodPost && cleaned == "/inference/v1/chat/completions":
		return routeCreateChatCompletion, "", nil
	case method == http.MethodPost && cleaned == "/inference/v1/responses":
		return routeCreateResponse, "", nil
	case method == http.MethodGet && cleaned == "/inference/v1/responses":
		return routeListResponses, "", nil
	case method == http.MethodPost && cleaned == "/inference/v1/completions":
		return routeCreateCompletion, "", nil
	}

	if strings.HasPrefix(cleaned, "/inference/v1/responses/") {
		id := strings.TrimPrefix(cleaned, "/inference/v1/responses/")
		if id == "" || strings.Contains(id, "/") {
			return routeUnknown, "", fmt.Errorf("path %q is not supported", requestPath)
		}
		switch method {
		case http.MethodGet:
			return routeGetResponse, id, nil
		case http.MethodDelete:
			return routeDeleteResponse, id, nil
		}
	}

	return routeUnknown, "", fmt.Errorf("path %q with method %s is not supported", requestPath, method)
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	cleaned := path.Clean(p)
	if !strings.HasPrefix(cleaned, "/") {
		return "/" + cleaned
	}
	return cleaned
}

func joinURLPath(base *url.URL, incoming string) string {
	switch {
	case base.Path == "" || base.Path == "/":
		return incoming
	case incoming == "":
		return base.Path
	default:
		return strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(incoming, "/")
	}
}
