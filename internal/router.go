package internal

import (
	"encoding/json"
	"golang.org/x/time/rate"
	"net/http"
)

// Router represents the HTTP router
type Router struct {
	mux     *http.ServeMux
	limiter *rate.Limiter
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return &Router{
		mux:     http.NewServeMux(),
		limiter: rate.NewLimiter(5, 2), // 5 requests per second, burst of 2
	}
}

// SetupRoutes configures all the routes for the router
func (r *Router) SetupRoutes() {
	// Handle all GET requests to any path
	r.mux.HandleFunc("GET /{path...}", r.handleDynamicPath)
}

// handleDynamicPath handles GET requests to any arbitrary path
func (r *Router) handleDynamicPath(w http.ResponseWriter, req *http.Request) {
	// Check rate limit
	if !r.limiter.Allow() {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

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
		return
	}
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
