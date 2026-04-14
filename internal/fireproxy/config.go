package fireproxy

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	defaultListenAddr      = ":8080"
	defaultUpstreamBase    = "https://api.fireworks.ai"
	defaultBodyLimitBytes  = 2 << 20
	defaultPublicModelID   = "kimi-k2p5-turbo"
	defaultUpstreamModelID = "accounts/fireworks/routers/kimi-k2p5-turbo"
)

type Config struct {
	ListenAddr           string
	UpstreamBaseURL      *url.URL
	AllowedHosts         map[string]struct{}
	AllowedModels        map[string]struct{}
	APIKeys              map[string]struct{}
	PublicModelID        string
	UpstreamModelID      string
	UpstreamModelAliases map[string]struct{}
	UpstreamAPIKey       string
	BodyLimitBytes       int64
}

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		ListenAddr:      readEnv("LISTEN_ADDR", defaultListenAddr),
		AllowedHosts:    splitSet(os.Getenv("ALLOWED_HOSTS")),
		PublicModelID:   readEnv("PUBLIC_MODEL_ID", defaultPublicModelID),
		UpstreamModelID: readEnv("UPSTREAM_MODEL_ID", defaultUpstreamModelID),
		UpstreamAPIKey:  strings.TrimSpace(os.Getenv("UPSTREAM_API_KEY")),
		APIKeys:         splitSet(os.Getenv("API_KEYS")),
		BodyLimitBytes:  defaultBodyLimitBytes,
	}

	cfg.AllowedModels = splitSet(os.Getenv("ALLOWED_MODELS"))
	if len(cfg.AllowedModels) == 0 {
		cfg.AllowedModels = map[string]struct{}{
			strings.ToLower(cfg.PublicModelID):   {},
			strings.ToLower(cfg.UpstreamModelID): {},
		}
	}
	cfg.UpstreamModelAliases = splitSet(os.Getenv("UPSTREAM_MODEL_ALIASES"))
	if len(cfg.UpstreamModelAliases) == 0 {
		cfg.UpstreamModelAliases = map[string]struct{}{
			strings.ToLower(cfg.UpstreamModelID):  {},
			"accounts/fireworks/models/kimi-k2p5": {},
		}
	}

	upstream := readEnv("UPSTREAM_BASE_URL", defaultUpstreamBase)
	upstreamURL, err := url.Parse(upstream)
	if err != nil {
		return Config{}, fmt.Errorf("parse UPSTREAM_BASE_URL: %w", err)
	}
	if upstreamURL.Scheme == "" || upstreamURL.Host == "" {
		return Config{}, fmt.Errorf("UPSTREAM_BASE_URL must include scheme and host")
	}
	cfg.UpstreamBaseURL = upstreamURL

	if raw := strings.TrimSpace(os.Getenv("REQUEST_BODY_LIMIT_BYTES")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("parse REQUEST_BODY_LIMIT_BYTES: %w", err)
		}
		if value <= 0 {
			return Config{}, fmt.Errorf("REQUEST_BODY_LIMIT_BYTES must be positive")
		}
		cfg.BodyLimitBytes = value
	}

	return cfg, nil
}

func (c Config) NormalizeIncomingModel(model string) string {
	if strings.EqualFold(strings.TrimSpace(model), c.PublicModelID) {
		return c.UpstreamModelID
	}
	return strings.TrimSpace(model)
}

func (c Config) NormalizeOutgoingModel(model string) string {
	if _, ok := c.UpstreamModelAliases[strings.ToLower(strings.TrimSpace(model))]; ok {
		return c.PublicModelID
	}
	return strings.TrimSpace(model)
}

func readEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitSet(raw string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		value := strings.ToLower(strings.TrimSpace(part))
		if value == "" {
			continue
		}
		result[value] = struct{}{}
	}
	return result
}
