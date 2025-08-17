package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/grumpyguvner/gomail/cmd/webadmin/config"
	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
	"github.com/grumpyguvner/gomail/cmd/webadmin/proxy"
)

type APIHandler struct {
	config    *config.Config
	logger    *logging.Logger
	gomailAPI *proxy.GoMailProxy
}

func NewAPIHandler(cfg *config.Config, logger *logging.Logger) *APIHandler {
	return &APIHandler{
		config:    cfg,
		logger:    logger,
		gomailAPI: proxy.NewGoMailProxy(cfg.GoMailAPIURL, cfg.BearerToken, logger),
	}
}

// Domain Management

func (h *APIHandler) ListDomains(w http.ResponseWriter, r *http.Request) {
	// Get domains from configuration
	domains := make([]map[string]interface{}, 0, len(h.config.Domains))
	for domain, cfg := range h.config.Domains {
		domains = append(domains, map[string]interface{}{
			"domain":         domain,
			"action":         cfg.Action,
			"forward_to":     cfg.ForwardTo,
			"bounce_message": cfg.BounceMessage,
			"health_checks":  cfg.HealthChecks,
		})
	}

	h.writeJSON(w, map[string]interface{}{
		"domains": domains,
		"total":   len(domains),
	})
}

func (h *APIHandler) CreateDomain(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	domain, ok := req["domain"].(string)
	if !ok || domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	// TODO: Implement domain creation
	h.writeJSON(w, map[string]interface{}{
		"message": "Domain creation not yet implemented",
		"domain":  domain,
	})
}

func (h *APIHandler) GetDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	cfg, exists := h.config.Domains[domain]
	if !exists {
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	h.writeJSON(w, map[string]interface{}{
		"domain":         domain,
		"action":         cfg.Action,
		"forward_to":     cfg.ForwardTo,
		"bounce_message": cfg.BounceMessage,
		"health_checks":  cfg.HealthChecks,
	})
}

func (h *APIHandler) UpdateDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Implement domain update
	h.writeJSON(w, map[string]interface{}{
		"message": "Domain update not yet implemented",
		"domain":  domain,
	})
}

func (h *APIHandler) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	// TODO: Implement domain deletion
	h.writeJSON(w, map[string]interface{}{
		"message": "Domain deletion not yet implemented",
		"domain":  domain,
	})
}

// Email Management

func (h *APIHandler) ListEmails(w http.ResponseWriter, r *http.Request) {
	// Proxy to GoMail API
	emails, err := h.gomailAPI.ListEmails(r.URL.Query())
	if err != nil {
		h.logger.Error("Failed to list emails", "error", err)
		http.Error(w, "Failed to retrieve emails", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, emails)
}

func (h *APIHandler) GetEmail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	emailID := vars["id"]

	// Proxy to GoMail API
	email, err := h.gomailAPI.GetEmail(emailID)
	if err != nil {
		h.logger.Error("Failed to get email", "error", err, "id", emailID)
		http.Error(w, "Failed to retrieve email", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, email)
}

func (h *APIHandler) DeleteEmail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	emailID := vars["id"]

	// Proxy to GoMail API
	err := h.gomailAPI.DeleteEmail(emailID)
	if err != nil {
		h.logger.Error("Failed to delete email", "error", err, "id", emailID)
		http.Error(w, "Failed to delete email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) GetEmailRaw(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	emailID := vars["id"]

	// Proxy to GoMail API
	rawEmail, err := h.gomailAPI.GetEmailRaw(emailID)
	if err != nil {
		h.logger.Error("Failed to get raw email", "error", err, "id", emailID)
		http.Error(w, "Failed to retrieve raw email", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"email-%s.eml\"", emailID))
	_, _ = w.Write(rawEmail)
}

// Routing Rules

func (h *APIHandler) ListRoutingRules(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement routing rules listing
	h.writeJSON(w, map[string]interface{}{
		"rules": []map[string]interface{}{},
		"total": 0,
	})
}

func (h *APIHandler) CreateRoutingRule(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Implement routing rule creation
	h.writeJSON(w, map[string]interface{}{
		"message": "Routing rule creation not yet implemented",
	})
}

func (h *APIHandler) UpdateRoutingRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Implement routing rule update
	h.writeJSON(w, map[string]interface{}{
		"message": "Routing rule update not yet implemented",
		"id":      ruleID,
	})
}

func (h *APIHandler) DeleteRoutingRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	// TODO: Implement routing rule deletion
	h.writeJSON(w, map[string]interface{}{
		"message": "Routing rule deletion not yet implemented",
		"id":      ruleID,
	})
}

// Server-Sent Events

func (h *APIHandler) EventsSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
	flusher.Flush()

	// Keep connection alive with heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, "data: {\"type\":\"heartbeat\",\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
			flusher.Flush()
		}
	}
}

// Helper methods

func (h *APIHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
