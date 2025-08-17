package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/grumpyguvner/gomail/cmd/webadmin/config"
	"github.com/grumpyguvner/gomail/cmd/webadmin/handlers"
	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
	"github.com/grumpyguvner/gomail/cmd/webadmin/middleware"
)

func main() {
	// Initialize logger
	logger, err := logging.NewLogger("info", "console")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create router
	router := mux.NewRouter()

	// Setup middleware
	router.Use(middleware.Logging(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS())

	// Get embedded static files
	staticFS, err := GetStaticFS()
	if err != nil {
		logger.Error("Failed to get static filesystem", "error", err)
		os.Exit(1)
	}

	// Setup handlers
	apiHandler := handlers.NewAPIHandler(cfg, logger)
	staticHandler := handlers.NewStaticHandler(cfg, logger, staticFS)
	healthHandler := handlers.NewHealthHandler(cfg, logger)

	// API routes with authentication
	api := router.PathPrefix("/api").Subrouter()
	api.Use(middleware.Auth(cfg.BearerToken))

	// Health endpoints
	api.HandleFunc("/health", healthHandler.SystemHealth).Methods("GET")
	api.HandleFunc("/domains/{domain}/health", healthHandler.DomainHealth).Methods("GET")
	api.HandleFunc("/domains/{domain}/health/refresh", healthHandler.RefreshDomainHealth).Methods("POST")

	// Domain management endpoints
	api.HandleFunc("/domains", apiHandler.ListDomains).Methods("GET")
	api.HandleFunc("/domains", apiHandler.CreateDomain).Methods("POST")
	api.HandleFunc("/domains/{domain}", apiHandler.GetDomain).Methods("GET")
	api.HandleFunc("/domains/{domain}", apiHandler.UpdateDomain).Methods("PUT")
	api.HandleFunc("/domains/{domain}", apiHandler.DeleteDomain).Methods("DELETE")

	// Email management endpoints
	api.HandleFunc("/emails", apiHandler.ListEmails).Methods("GET")
	api.HandleFunc("/emails/{id}", apiHandler.GetEmail).Methods("GET")
	api.HandleFunc("/emails/{id}", apiHandler.DeleteEmail).Methods("DELETE")
	api.HandleFunc("/emails/{id}/raw", apiHandler.GetEmailRaw).Methods("GET")

	// Routing configuration endpoints
	api.HandleFunc("/routing/rules", apiHandler.ListRoutingRules).Methods("GET")
	api.HandleFunc("/routing/rules", apiHandler.CreateRoutingRule).Methods("POST")
	api.HandleFunc("/routing/rules/{id}", apiHandler.UpdateRoutingRule).Methods("PUT")
	api.HandleFunc("/routing/rules/{id}", apiHandler.DeleteRoutingRule).Methods("DELETE")

	// Real-time events endpoint
	api.HandleFunc("/events", apiHandler.EventsSSE).Methods("GET")

	// Static file serving for SPA
	router.PathPrefix("/").Handler(staticHandler.ServeStatic())

	// Create HTTPS server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.IdleTimeout) * time.Second,
	}

	// Configure TLS
	if cfg.SSLCert != "" && cfg.SSLKey != "" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}
		server.TLSConfig = tlsConfig
	}

	// Start server
	go func() {
		logger.Info("Starting webadmin server", "port", cfg.Port, "ssl", cfg.SSLCert != "")

		var err error
		if cfg.SSLCert != "" && cfg.SSLKey != "" {
			err = server.ListenAndServeTLS(cfg.SSLCert, cfg.SSLKey)
		} else {
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down webadmin server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Webadmin server shutdown complete")
}
