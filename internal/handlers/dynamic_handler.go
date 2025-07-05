package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/shaibs3/Guardz/internal/db_model"

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

// validateURL checks if a URL is safe to fetch
func validateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s (only http and https are allowed)", parsedURL.Scheme)
	}

	// Check for private/internal IP addresses (SSRF protection)
	host := parsedURL.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return fmt.Errorf("access to localhost is not allowed")
	}

	// Parse IP to check for private ranges
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("access to private IP %s is not allowed", ip)
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	privateBlocks := []string{
		"127.0.0.0/8",    // localhost
		"10.0.0.0/8",     // private
		"172.16.0.0/12",  // private
		"192.168.0.0/16", // private
		"169.254.0.0/16", // link-local
		"::1/128",        // localhost IPv6
		"fe80::/10",      // link-local IPv6
		"fc00::/7",       // unique local IPv6
	}

	for _, block := range privateBlocks {
		_, cidr, _ := net.ParseCIDR(block)
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
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

	// Limit concurrent requests to prevent resource exhaustion
	maxConcurrent := 10
	semaphore := make(chan struct{}, maxConcurrent)

	// Fetch URLs in parallel
	for i, urlRec := range urls {
		wg.Add(1)
		go func(index int, urlRec db_model.URLRecord) {
			defer wg.Done()

			// Acquire semaphore to limit concurrency
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := map[string]interface{}{
				"url": urlRec.URL,
			}

			// Validate URL before making request
			if err := validateURL(urlRec.URL); err != nil {
				result["error"] = err.Error()
				resultChan <- urlResult{index: index, result: result}
				return
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

			// Set a custom User-Agent
			httpReq.Header.Set("User-Agent", "Guardz-URL-Fetcher/1.0")

			// Create a custom HTTP client that handles redirects
			client := &http.Client{
				Timeout: 30 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					// Limit redirects to prevent infinite loops
					if len(via) >= 10 {
						return fmt.Errorf("too many redirects")
					}
					return nil
				},
			}

			// Make the HTTP request
			resp, err := client.Do(httpReq)
			if err != nil {
				result["error"] = err.Error()
				resultChan <- urlResult{index: index, result: result}
				return
			}

			// Read response body with size limit (1MB)
			limitedReader := io.LimitReader(resp.Body, 1<<20) // 1MB limit
			body, err := io.ReadAll(limitedReader)
			cerr := resp.Body.Close()
			if err != nil {
				result["error"] = err.Error()
				resultChan <- urlResult{index: index, result: result}
				return
			}
			if cerr != nil {
				result["error"] = cerr.Error()
				resultChan <- urlResult{index: index, result: result}
				return
			}

			// Check if response was truncated due to size limit
			if len(body) == 1<<20 {
				result["warning"] = "Response truncated due to size limit (1MB)"
			}

			// Track redirect information
			if len(resp.Request.URL.String()) != len(urlRec.URL) || resp.Request.URL.String() != urlRec.URL {
				result["original_url"] = urlRec.URL
				result["final_url"] = resp.Request.URL.String()
				result["redirected"] = true
			} else {
				result["redirected"] = false
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

	// Validate all URLs before storing
	var validURLs []string
	var invalidURLs []string
	for _, urlStr := range body.URLs {
		if err := validateURL(urlStr); err != nil {
			invalidURLs = append(invalidURLs, fmt.Sprintf("%s: %s", urlStr, err.Error()))
		} else {
			validURLs = append(validURLs, urlStr)
		}
	}

	// If all URLs are invalid, return error
	if len(validURLs) == 0 {
		http.Error(w, fmt.Sprintf("All URLs are invalid: %v", invalidURLs), http.StatusBadRequest)
		return
	}

	// Store only valid URLs
	if err := h.DB.StoreURLsForPath(req.Context(), path, validURLs); err != nil {
		http.Error(w, "Failed to store URLs", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "URLs stored successfully",
		"path":    path,
		"count":   len(validURLs),
	}

	// Include information about invalid URLs if any
	if len(invalidURLs) > 0 {
		response["invalid_urls"] = invalidURLs
		response["warning"] = fmt.Sprintf("Some URLs were rejected: %d valid, %d invalid", len(validURLs), len(invalidURLs))
	}

	w.WriteHeader(http.StatusCreated)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
