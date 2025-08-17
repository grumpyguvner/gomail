package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/grumpyguvner/gomail/cmd/webadmin/config"
	"github.com/grumpyguvner/gomail/cmd/webadmin/health"
	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type HealthHandler struct {
	config        *config.Config
	logger        *logging.Logger
	healthChecker *health.Checker
}

func NewHealthHandler(cfg *config.Config, logger *logging.Logger) *HealthHandler {
	return &HealthHandler{
		config:        cfg,
		logger:        logger,
		healthChecker: health.NewChecker(logger),
	}
}

func (h *HealthHandler) SystemHealth(w http.ResponseWriter, r *http.Request) {
	// Basic system health check
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",                         // TODO: Get from build info
		"uptime":    time.Since(time.Now()).String(), // TODO: Track actual uptime
		"checks": map[string]interface{}{
			"gomail_api": h.checkGoMailAPI(),
			"disk_space": h.checkDiskSpace(),
			"memory":     h.checkMemory(),
		},
	}

	h.writeJSON(w, health)
}

func (h *HealthHandler) DomainHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	// Check if domain is configured
	_, exists := h.config.Domains[domain]
	if !exists {
		http.Error(w, "Domain not configured", http.StatusNotFound)
		return
	}

	// Perform health check
	health, err := h.healthChecker.CheckDomain(domain)
	if err != nil {
		h.logger.Error("Failed to check domain health", "error", err, "domain", domain)
		http.Error(w, "Failed to check domain health", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, health)
}

func (h *HealthHandler) RefreshDomainHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	// Check if domain is configured
	_, exists := h.config.Domains[domain]
	if !exists {
		http.Error(w, "Domain not configured", http.StatusNotFound)
		return
	}

	// Force refresh health check
	health, err := h.healthChecker.RefreshDomain(domain)
	if err != nil {
		h.logger.Error("Failed to refresh domain health", "error", err, "domain", domain)
		http.Error(w, "Failed to refresh domain health", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, health)
}

// Helper methods for system health checks

func (h *HealthHandler) checkGoMailAPI() map[string]interface{} {
	// TODO: Implement actual GoMail API health check
	return map[string]interface{}{
		"status":      "healthy",
		"response_ms": 45,
	}
}

func (h *HealthHandler) checkDiskSpace() map[string]interface{} {
	// TODO: Implement actual disk space check
	return map[string]interface{}{
		"status":    "healthy",
		"usage_pct": 23,
		"free_gb":   45.2,
	}
}

func (h *HealthHandler) checkMemory() map[string]interface{} {
	// TODO: Implement actual memory check
	return map[string]interface{}{
		"status":    "healthy",
		"usage_pct": 67,
		"free_mb":   512,
	}
}

func (h *HealthHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
