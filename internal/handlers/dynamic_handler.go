package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/shaibs3/Guardz/internal/db_model"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

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
	router.HandleFunc("/{path:.*}", h.handleGetPath).Methods("GET")
	router.HandleFunc("/{path:.*}", h.handlePostPath).Methods("POST")
}

// handleGetPath handles GET requests to any arbitrary path
func (h *DynamicHandler) handleGetPath(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(req.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	urls, err := h.DB.GetURLsByPath(req.Context(), path)
	if err != nil {
		http.Error(w, "Failed to fetch records", http.StatusInternalServerError)
		return
	}

	// Create a channel to collect results
	type urlResult struct {
		index  int
		result map[string]interface{}
	}
	resultChan := make(chan urlResult, len(urls))

	// Create a WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	// Fetch URLs in parallel
	for i, urlRec := range urls {
		wg.Add(1)
		go func(index int, urlRec db_model.URLRecord) {
			defer wg.Done()

			result := map[string]interface{}{
				"url": urlRec.URL,
			}

			// Create a context with timeout for the HTTP request
			ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
			defer cancel()

			// Create HTTP request with context
			httpReq, err := http.NewRequestWithContext(ctx, "GET", urlRec.URL, nil)
			if err != nil {
				result["error"] = err.Error()
				resultChan <- urlResult{index: index, result: result}
				return
			}

			// Make the HTTP request
			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				result["error"] = err.Error()
				resultChan <- urlResult{index: index, result: result}
				return
			}
			defer resp.Body.Close()

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				result["error"] = err.Error()
				resultChan <- urlResult{index: index, result: result}
				return
			}

			contentType := resp.Header.Get("Content-Type")
			result["content_type"] = contentType
			result["status_code"] = resp.StatusCode

			// If not text, encode as base64
			if strings.HasPrefix(contentType, "text/") || strings.Contains(contentType, "json") || strings.Contains(contentType, "xml") {
				result["content"] = string(body)
			} else {
				result["content"] = base64.StdEncoding.EncodeToString(body)
			}

			resultChan <- urlResult{index: index, result: result}
		}(i, urlRec)
	}

	// Close the channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results in order
	results := make([]map[string]interface{}, len(urls))
	for result := range resultChan {
		results[result.index] = result.result
	}

	response := map[string]interface{}{
		"path":    path,
		"results": results,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
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
	err := json.NewEncoder(w).Encode(map[string]interface{}{"message": "URLs stored successfully", "path": path, "count": len(body.URLs)})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
