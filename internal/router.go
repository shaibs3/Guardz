package internal

import (
	"encoding/json"
	"net/http"
)

// Router represents the HTTP router
type Router struct {
	mux *http.ServeMux
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return &Router{
		mux: http.NewServeMux(),
	}
}

// SetupRoutes configures all the routes for the router
func (r *Router) SetupRoutes() {
	r.mux.HandleFunc("POST /api/example", r.handleExample)
}

// handleExample handles POST requests to /api/example
func (r *Router) handleExample(w http.ResponseWriter, req *http.Request) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Parse request body (example)
	var requestBody map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create response
	response := map[string]interface{}{
		"message": "POST request received successfully",
		"data":    requestBody,
	}

	// Send response
	json.NewEncoder(w).Encode(response)
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
