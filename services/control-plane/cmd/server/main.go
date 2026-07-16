// Command server is the OSINT Lead Platform control-plane API.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/config"
	httpapi "github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/http"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/registry"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/runner"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	httpapi.SetCORSOrigin(cfg.CORSOrigin)

	var st store.Store
	if cfg.DatabaseURL != "" {
		ps, err := store.NewPostgresStore(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("connect to postgres: %v", err)
		}
		defer ps.Close()
		st = ps
		log.Println("store: postgres")
	} else {
		st = store.NewMemoryStore()
		log.Println("store: memory (set DATABASE_URL for postgres)")
	}

	r := runner.New(st, cfg.ModuleTimeout)
	reg := registry.New()
	srv := httpapi.NewServer(st, r, reg)

	addr := ":" + cfg.Port
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
	}

	go func() {
		log.Printf("control-plane listening on %s (CORS origin: %s)", addr, cfg.CORSOrigin)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}
}
