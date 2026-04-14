package fireproxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
)

type Server struct {
	cfg    Config
	logger *log.Logger
	proxy  *httputil.ReverseProxy
}

func NewServer(cfg Config, logger *log.Logger) (*Server, error) {
	proxy := httputil.NewSingleHostReverseProxy(cfg.UpstreamBaseURL)
	proxy.Director = nil
	baseQuery := cfg.UpstreamBaseURL.RawQuery

	proxy.Rewrite = func(pr *httputil.ProxyRequest) {
		pr.Out.URL.Scheme = cfg.UpstreamBaseURL.Scheme
		pr.Out.URL.Host = cfg.UpstreamBaseURL.Host
		pr.Out.URL.Path = joinURLPath(cfg.UpstreamBaseURL, pr.In.URL.Path)
		pr.Out.URL.RawPath = joinURLPath(cfg.UpstreamBaseURL, pr.In.URL.EscapedPath())
		pr.Out.Host = cfg.UpstreamBaseURL.Host
		if baseQuery == "" || pr.In.URL.RawQuery == "" {
			pr.Out.URL.RawQuery = baseQuery + pr.In.URL.RawQuery
		} else {
			pr.Out.URL.RawQuery = baseQuery + "&" + pr.In.URL.RawQuery
		}
		pr.SetXForwarded()
		if cfg.UpstreamAPIKey != "" {
			pr.Out.Header.Set("Authorization", "Bearer "+cfg.UpstreamAPIKey)
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Printf("proxy error %s %s: %v", r.Method, r.URL.Path, err)
		writeError(w, http.StatusBadGateway, "upstream_error", "upstream request failed")
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		return rewriteUpstreamResponse(resp, cfg)
	}

	return &Server{
		cfg:    cfg,
		logger: logger,
		proxy:  proxy,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.validateHost(r.Host); err != nil {
		writeError(w, http.StatusForbidden, "forbidden_host", err.Error())
		return
	}

	kind, resourceID, err := matchRoute(r.Method, r.URL.Path)
	if err != nil {
		writeError(w, http.StatusNotFound, "unsupported_path", err.Error())
		return
	}

	switch kind {
	case routeHealth:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	case routeListModels:
		if err := validateAuthorization(r.Header, s.cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_authorization", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, buildModelListResponse(s.cfg))
		return
	case routeListResponses:
		if err := validateListResponsesQuery(r.URL.Query()); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
			return
		}
	case routeGetResponse, routeDeleteResponse:
		if !responseIDPattern.MatchString(resourceID) {
			writeError(w, http.StatusBadRequest, "invalid_response_id", "response id format is invalid")
			return
		}
	case routeCreateResponse, routeCreateChatCompletion, routeCreateCompletion:
		if err := s.validateJSONRequest(r, rewriteModelInBody); err != nil {
			s.handleValidationError(w, err)
			return
		}
	}

	if err := validateAuthorization(r.Header, s.cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_authorization", err.Error())
		return
	}

	s.logger.Printf("forward %s %s", r.Method, r.URL.Path)
	s.proxy.ServeHTTP(w, r)
}

func (s *Server) validateHost(hostport string) error {
	if len(s.cfg.AllowedHosts) == 0 {
		return nil
	}

	host := strings.ToLower(strings.TrimSpace(hostport))
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	if _, ok := s.cfg.AllowedHosts[host]; ok {
		return nil
	}
	return fmt.Errorf("host %q is not allowed", host)
}
