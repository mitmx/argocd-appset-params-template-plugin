package main

import (
	"log"
	"net/http"

	"github.com/mitmx/argocd-values-pipeline-plugin/internal/config"
	"github.com/mitmx/argocd-values-pipeline-plugin/internal/engine"
	"github.com/mitmx/argocd-values-pipeline-plugin/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	eng := engine.New(cfg.DefaultMaxTemplatePasses)
	handler := server.New(cfg, eng)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	log.Printf("values-pipeline-plugin listening on %s", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}
