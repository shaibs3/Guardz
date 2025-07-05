package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// DynamicHandler handles dynamic path requests
type DynamicHandler struct{}

// NewDynamicHandler creates a new dynamic handler
func NewDynamicHandler() *DynamicHandler {
	return &DynamicHandler{}
}

// RegisterRoutes registers the routes for this handler
func (h *DynamicHandler) RegisterRoutes(router *mux.Router, logger *zap.Logger) {
	router.HandleFunc("/{path:.*}", h.handleDynamicPath).Methods("GET")
}

// handleDynamicPath handles GET requests to any arbitrary path
func (h *DynamicHandler) handleDynamicPath(w http.ResponseWriter, req *http.Request) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Get the path from the request
	path := req.URL.Path

	// Create response
	response := map[string]interface{}{
		"message": "GET request received successfully",
		"method":  "GET",
		"path":    path,
		"url":     req.URL.String(),
		"host":    req.Host,
	}

	// Send response
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
