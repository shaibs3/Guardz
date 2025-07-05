package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/shaibs3/Guardz/internal/lookup"
	"go.uber.org/zap"
)

// DynamicHandler handles dynamic path requests
type DynamicHandler struct {
	DB lookup.DbProvider
}

// NewDynamicHandler creates a new dynamic handler
func NewDynamicHandler(dbProvider lookup.DbProvider) *DynamicHandler {
	return &DynamicHandler{DB: dbProvider}
}

// RegisterRoutes registers the routes for this handler
func (h *DynamicHandler) RegisterRoutes(router *mux.Router, logger *zap.Logger) {
	router.HandleFunc("/{path:.*}", h.handleDynamicPath).Methods("GET")
	router.HandleFunc("/{path:.*}", h.handlePostPath).Methods("POST")
}

// handleDynamicPath handles GET requests to any arbitrary path
func (h *DynamicHandler) handleDynamicPath(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract the path from the request (strip leading slash)
	path := strings.TrimPrefix(req.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	records, err := h.DB.GetURLsByPath(req.Context(), path)
	if err != nil {
		http.Error(w, "Failed to fetch records", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"path":    path,
		"results": records,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handlePostPath handles POST requests to any arbitrary path
func (h *DynamicHandler) handlePostPath(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(req.URL.Path, "/")
	if path == "" {
		path = "/"
	}
	var body struct {
		URLs []string `json:"urls"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(body.URLs) == 0 {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		return
	}
	if err := h.DB.StoreURLsForPath(req.Context(), path, body.URLs); err != nil {
		http.Error(w, "Failed to store URLs", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "URLs stored successfully", "path": path, "count": len(body.URLs)})
}
