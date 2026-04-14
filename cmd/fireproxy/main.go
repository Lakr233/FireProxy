package main

import (
	"log"
	"net/http"
	"os"

	"fireproxy/internal/fireproxy"
)

func main() {
	logger := log.New(os.Stdout, "fireproxy ", log.LstdFlags|log.LUTC)

	cfg, err := fireproxy.LoadConfigFromEnv()
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	server, err := fireproxy.NewServer(cfg, logger)
	if err != nil {
		logger.Fatalf("create server: %v", err)
	}

	logger.Printf("listening on %s, upstream=%s", cfg.ListenAddr, cfg.UpstreamBaseURL)

	if err := http.ListenAndServe(cfg.ListenAddr, server); err != nil {
		logger.Fatalf("listen: %v", err)
	}
}
