package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/shaibs3/Guardz/internal/lookup"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestHandler() *DynamicHandler {
	return NewDynamicHandler(lookup.NewInMemoryProvider())
}

// allowlistTestServer adds the test server's host to the allowlist for SSRF validation
func allowlistTestServer(t *testing.T, serverURL string) func() {
	host := strings.Split(strings.TrimPrefix(serverURL, "http://"), ":")[0]
	os.Setenv("GUARDZ_TEST_ALLOWLIST", host)
	return func() {
		os.Unsetenv("GUARDZ_TEST_ALLOWLIST")
	}
}

func TestDynamicHandler_POST_and_GET(t *testing.T) {
	h := setupTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r, zap.NewNop())

	// Test POST
	postBody := map[string]interface{}{
		"urls": []string{"https://jsonplaceholder.typicode.com/todos/1", "https://example.com"},
	}
	bodyBytes, _ := json.Marshal(postBody)
	req := httptest.NewRequest(http.MethodPost, "/testpath", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "expected status 201")

	// Test GET
	getReq := httptest.NewRequest(http.MethodGet, "/testpath", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code, "expected status 200")
	var resp map[string]interface{}
	err := json.Unmarshal(getW.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to decode response")
	require.Equal(t, "testpath", resp["path"], "expected path 'testpath'")
	results, ok := resp["results"].([]interface{})
	require.True(t, ok, "expected results to be a slice")
	require.Len(t, results, 2, "expected 2 results")
}

func TestDynamicHandler_RedirectHandling(t *testing.T) {
	// Create a mock server that simulates redirects
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect1":
			// First redirect: /redirect1 -> /redirect2
			http.Redirect(w, r, "/redirect2", http.StatusMovedPermanently)
		case "/redirect2":
			// Second redirect: /redirect2 -> /final
			http.Redirect(w, r, "/final", http.StatusFound)
		case "/final":
			// Final destination
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Final destination reached"))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
		case "/single-redirect":
			// Single redirect
			http.Redirect(w, r, "/final", http.StatusMovedPermanently)
		case "/no-redirect":
			// No redirect
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("No redirect"))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// Allowlist the test server's host
	cleanup := allowlistTestServer(t, mockServer.URL)
	defer cleanup()

	h := setupTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r, zap.NewNop())

	// Test URLs with different redirect scenarios
	testURLs := []string{
		mockServer.URL + "/redirect1",       // Multiple redirects
		mockServer.URL + "/single-redirect", // Single redirect
		mockServer.URL + "/no-redirect",     // No redirect
	}

	// Store URLs
	postBody := map[string]interface{}{
		"urls": testURLs,
	}
	bodyBytes, _ := json.Marshal(postBody)
	req := httptest.NewRequest(http.MethodPost, "/redirect-test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "expected status 201")

	// Fetch URLs and check redirect handling
	getReq := httptest.NewRequest(http.MethodGet, "/redirect-test", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code, "expected status 200")

	var resp map[string]interface{}
	err := json.Unmarshal(getW.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to decode response")

	results, ok := resp["results"].([]interface{})
	require.True(t, ok, "expected results to be a slice")
	require.Len(t, results, 3, "expected 3 results")

	// Check first result (multiple redirects)
	result1 := results[0].(map[string]interface{})
	require.Equal(t, mockServer.URL+"/redirect1", result1["url"], "original URL should match")
	require.Equal(t, mockServer.URL+"/final", result1["final_url"], "final URL should be the destination")
	require.Equal(t, true, result1["redirected"], "should indicate redirect occurred")
	require.Equal(t, float64(200), result1["status_code"], "final status should be 200")
	require.Equal(t, "Final destination reached", result1["content"], "should have final content")

	// Check second result (single redirect)
	result2 := results[1].(map[string]interface{})
	require.Equal(t, mockServer.URL+"/single-redirect", result2["url"], "original URL should match")
	require.Equal(t, mockServer.URL+"/final", result2["final_url"], "final URL should be the destination")
	require.Equal(t, true, result2["redirected"], "should indicate redirect occurred")
	require.Equal(t, float64(200), result2["status_code"], "final status should be 200")
	require.Equal(t, "Final destination reached", result2["content"], "should have final content")

	// Check third result (no redirect)
	result3 := results[2].(map[string]interface{})
	require.Equal(t, mockServer.URL+"/no-redirect", result3["url"], "original URL should match")
	require.Equal(t, false, result3["redirected"], "should indicate no redirect occurred")
	require.Equal(t, float64(200), result3["status_code"], "status should be 200")
	require.Equal(t, "No redirect", result3["content"], "should have original content")
}

func TestDynamicHandler_RedirectLoopProtection(t *testing.T) {
	// Create a mock server that simulates a redirect loop
	redirectCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount <= 15 { // More than our 10 redirect limit
			http.Redirect(w, r, "/loop", http.StatusMovedPermanently)
		} else {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Should not reach here"))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
		}
	}))
	defer mockServer.Close()

	// Allowlist the test server's host
	cleanup := allowlistTestServer(t, mockServer.URL)
	defer cleanup()

	h := setupTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r, zap.NewNop())

	// Store URL
	postBody := map[string]interface{}{
		"urls": []string{mockServer.URL + "/loop"},
	}
	bodyBytes, _ := json.Marshal(postBody)
	req := httptest.NewRequest(http.MethodPost, "/loop-test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "expected status 201")

	// Fetch URL and check that redirect loop is detected
	getReq := httptest.NewRequest(http.MethodGet, "/loop-test", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code, "expected status 200")

	var resp map[string]interface{}
	err := json.Unmarshal(getW.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to decode response")

	results, ok := resp["results"].([]interface{})
	require.True(t, ok, "expected results to be a slice")
	require.Len(t, results, 1, "expected 1 result")

	result := results[0].(map[string]interface{})
	require.Contains(t, result["error"], "too many redirects", "should detect redirect loop")
}

func TestDynamicHandler_MultipleContentTypes(t *testing.T) {
	// Create a mock server that returns different content types
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"name": "test", "value": 123, "active": true}`))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
		case "/image":
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			// Create a minimal PNG file (1x1 transparent pixel)
			pngData := []byte{
				0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
				0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
				0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 image
				0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // bit depth, color type, etc.
				0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
				0x54, 0x08, 0x99, 0x01, 0x01, 0x00, 0x00, 0xFF, // compressed data
				0xFF, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, 0xE2, // more data
				0x21, 0xBC, 0x33, 0x00, 0x00, 0x00, 0x00, 0x49, // IEND chunk
				0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
			}
			_, err := w.Write(pngData)
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("This is plain text content with some special characters: áéíóú ñ ç"))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
		case "/html":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Hello World</h1></body></html>`))
			if err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// Allowlist the test server's host
	cleanup := allowlistTestServer(t, mockServer.URL)
	defer cleanup()

	h := setupTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r, zap.NewNop())

	// Test URLs with different content types
	testURLs := []string{
		mockServer.URL + "/json",  // JSON content
		mockServer.URL + "/image", // PNG image
		mockServer.URL + "/text",  // Plain text
		mockServer.URL + "/html",  // HTML content
	}

	// Store URLs
	postBody := map[string]interface{}{
		"urls": testURLs,
	}
	bodyBytes, _ := json.Marshal(postBody)
	req := httptest.NewRequest(http.MethodPost, "/content-test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "expected status 201")

	// Fetch URLs and check content type handling
	getReq := httptest.NewRequest(http.MethodGet, "/content-test", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code, "expected status 200")

	var resp map[string]interface{}
	err := json.Unmarshal(getW.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to decode response")

	results, ok := resp["results"].([]interface{})
	require.True(t, ok, "expected results to be a slice")
	require.Len(t, results, 4, "expected 4 results")

	// Check JSON content
	result1 := results[0].(map[string]interface{})
	require.Equal(t, mockServer.URL+"/json", result1["url"], "JSON URL should match")
	require.Equal(t, "application/json", result1["content_type"], "should have JSON content type")
	require.Equal(t, float64(200), result1["status_code"], "should have 200 status")
	require.Equal(t, `{"name": "test", "value": 123, "active": true}`, result1["content"], "should have JSON content as text")

	// Check PNG image content
	result2 := results[1].(map[string]interface{})
	require.Equal(t, mockServer.URL+"/image", result2["url"], "Image URL should match")
	require.Equal(t, "image/png", result2["content_type"], "should have PNG content type")
	require.Equal(t, float64(200), result2["status_code"], "should have 200 status")
	// PNG content should be base64 encoded
	content2 := result2["content"].(string)
	require.True(t, len(content2) > 0, "should have base64 encoded content")
	// Verify it's valid base64 (contains only base64 characters)
	require.Regexp(t, `^[A-Za-z0-9+/]*={0,2}$`, content2, "should be valid base64")

	// Check plain text content
	result3 := results[2].(map[string]interface{})
	require.Equal(t, mockServer.URL+"/text", result3["url"], "Text URL should match")
	require.Equal(t, "text/plain", result3["content_type"], "should have plain text content type")
	require.Equal(t, float64(200), result3["status_code"], "should have 200 status")
	require.Equal(t, "This is plain text content with some special characters: áéíóú ñ ç", result3["content"], "should have text content")

	// Check HTML content
	result4 := results[3].(map[string]interface{})
	require.Equal(t, mockServer.URL+"/html", result4["url"], "HTML URL should match")
	require.Equal(t, "text/html", result4["content_type"], "should have HTML content type")
	require.Equal(t, float64(200), result4["status_code"], "should have 200 status")
	require.Equal(t, `<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Hello World</h1></body></html>`, result4["content"], "should have HTML content as text")
}

func TestDynamicHandler_RealURLsContentTypes(t *testing.T) {
	// Skip this test if running in CI or if network is not available
	if testing.Short() {
		t.Skip("Skipping real URL test in short mode")
	}

	// Allowlist the test server's host
	host := "httpbin.org"
	os.Setenv("GUARDZ_TEST_ALLOWLIST", host)
	defer os.Unsetenv("GUARDZ_TEST_ALLOWLIST")

	h := setupTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r, zap.NewNop())

	// Real URLs with different content types
	testURLs := []string{
		"https://httpbin.org/json",       // JSON content
		"https://httpbin.org/image/png",  // PNG image
		"https://httpbin.org/robots.txt", // Plain text
	}

	// Store URLs
	postBody := map[string]interface{}{
		"urls": testURLs,
	}
	bodyBytes, _ := json.Marshal(postBody)
	req := httptest.NewRequest(http.MethodPost, "/real-content-test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "expected status 201")

	// Fetch URLs and check content type handling
	getReq := httptest.NewRequest(http.MethodGet, "/real-content-test", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code, "expected status 200")

	var resp map[string]interface{}
	err := json.Unmarshal(getW.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to decode response")

	results, ok := resp["results"].([]interface{})
	require.True(t, ok, "expected results to be a slice")
	require.Len(t, results, 3, "expected 3 results")

	// Check JSON content
	result1 := results[0].(map[string]interface{})
	require.Equal(t, "https://httpbin.org/json", result1["url"], "JSON URL should match")
	require.Equal(t, "application/json", result1["content_type"], "should have JSON content type")
	require.Equal(t, float64(200), result1["status_code"], "should have 200 status")
	content1 := result1["content"].(string)
	require.Contains(t, content1, "slideshow", "should contain expected JSON content")

	// Check PNG image content
	result2 := results[1].(map[string]interface{})
	require.Equal(t, "https://httpbin.org/image/png", result2["url"], "Image URL should match")
	require.Equal(t, "image/png", result2["content_type"], "should have PNG content type")
	require.Equal(t, float64(200), result2["status_code"], "should have 200 status")
	content2 := result2["content"].(string)
	require.True(t, len(content2) > 0, "should have base64 encoded content")
	require.Regexp(t, `^[A-Za-z0-9+/]*={0,2}$`, content2, "should be valid base64")

	// Check plain text content
	result3 := results[2].(map[string]interface{})
	require.Equal(t, "https://httpbin.org/robots.txt", result3["url"], "Text URL should match")
	require.Equal(t, "text/plain", result3["content_type"], "should have plain text content type")
	require.Equal(t, float64(200), result3["status_code"], "should have 200 status")
	content3 := result3["content"].(string)
	require.Contains(t, content3, "User-agent", "should contain expected text content")
}

func TestDynamicHandler_SecurityValidation(t *testing.T) {
	h := setupTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r, zap.NewNop())

	// Test various security scenarios
	testCases := []struct {
		name        string
		urls        []string
		expectedErr bool
		statusCode  int
	}{
		{
			name:        "SSRF - localhost",
			urls:        []string{"http://localhost:8080/api"},
			expectedErr: true,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "SSRF - 127.0.0.1",
			urls:        []string{"http://127.0.0.1:8080/api"},
			expectedErr: true,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "SSRF - private IP",
			urls:        []string{"http://192.168.1.1:8080/api"},
			expectedErr: true,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "SSRF - IPv6 localhost",
			urls:        []string{"http://[::1]:8080/api"},
			expectedErr: true,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "Invalid scheme - file",
			urls:        []string{"file:///etc/passwd"},
			expectedErr: true,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "Invalid scheme - ftp",
			urls:        []string{"ftp://example.com/file"},
			expectedErr: true,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "Invalid scheme - data",
			urls:        []string{"data:text/plain;base64,SGVsbG8="},
			expectedErr: true,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "Malformed URL",
			urls:        []string{"not-a-url"},
			expectedErr: true,
			statusCode:  http.StatusBadRequest,
		},
		{
			name:        "Valid URLs mixed with invalid",
			urls:        []string{"https://httpbin.org/json", "http://localhost:8080/api", "https://example.com"},
			expectedErr: false,
			statusCode:  http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			postBody := map[string]interface{}{
				"urls": tc.urls,
			}
			bodyBytes, _ := json.Marshal(postBody)
			req := httptest.NewRequest(http.MethodPost, "/security-test", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, tc.statusCode, w.Code, "expected status %d", tc.statusCode)

			if tc.expectedErr {
				// For 400 errors, the response might be plain text, not JSON
				if w.Code == http.StatusBadRequest {
					// Check if it's a JSON response
					contentType := w.Header().Get("Content-Type")
					if strings.Contains(contentType, "application/json") {
						var resp map[string]interface{}
						err := json.Unmarshal(w.Body.Bytes(), &resp)
						require.NoError(t, err, "failed to decode error response")
						require.Contains(t, resp, "invalid_urls", "should contain invalid URLs list")
					} else {
						// Plain text error response
						body := w.Body.String()
						require.Contains(t, body, "invalid", "should contain error message")
					}
				}
			} else {
				// Should accept valid URLs and reject invalid ones
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err, "failed to decode response")
				require.Equal(t, "URLs stored successfully", resp["message"])
				require.Contains(t, resp, "warning", "should warn about rejected URLs")
			}
		})
	}
}

func TestDynamicHandler_ResponseSizeLimit(t *testing.T) {
	// Create a mock server that returns large responses
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		// Generate a response larger than 1MB
		largeData := make([]byte, 2<<20) // 2MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		_, err := w.Write(largeData)
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	}))
	defer mockServer.Close()

	// Allowlist the test server's host
	cleanup := allowlistTestServer(t, mockServer.URL)
	defer cleanup()

	h := setupTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r, zap.NewNop())

	// Store URL
	postBody := map[string]interface{}{
		"urls": []string{mockServer.URL},
	}
	bodyBytes, _ := json.Marshal(postBody)
	req := httptest.NewRequest(http.MethodPost, "/size-test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "expected status 201")

	// Fetch URL and check size limit
	getReq := httptest.NewRequest(http.MethodGet, "/size-test", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code, "expected status 200")

	var resp map[string]interface{}
	err := json.Unmarshal(getW.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to decode response")

	results, ok := resp["results"].([]interface{})
	require.True(t, ok, "expected results to be a slice")
	require.Len(t, results, 1, "expected 1 result")

	result := results[0].(map[string]interface{})
	require.Equal(t, mockServer.URL, result["url"], "URL should match")
	require.Equal(t, float64(200), result["status_code"], "should have 200 status")

	// Check that response was truncated
	require.Contains(t, result, "warning", "should have warning about truncation")
	require.Contains(t, result["warning"], "truncated", "should mention truncation")

	// Check that content is exactly 1MB (plain or base64 encoded)
	content := result["content"].(string)
	if enc, ok := result["content_encoding"]; ok && enc == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(content)
		require.NoError(t, err, "should decode base64 content")
		fmt.Printf("[DEBUG TEST] Received base64 content length: %d\n", len(decoded))
		require.Equal(t, 1<<20, len(decoded), "decoded content should be exactly 1MB (truncated from 2MB)")
	} else {
		fmt.Printf("[DEBUG TEST] Received content length: %d\n", len(content))
		require.Equal(t, 1<<20, len(content), "content should be exactly 1MB (truncated from 2MB)")
	}
}

func TestDynamicHandler_ConcurrentRequestLimit(t *testing.T) {
	// Create a mock server that delays responses
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("response"))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	}))
	defer mockServer.Close()

	// Allowlist the test server's host
	cleanup := allowlistTestServer(t, mockServer.URL)
	defer cleanup()

	h := setupTestHandler()
	r := mux.NewRouter()
	h.RegisterRoutes(r, zap.NewNop())

	// Create many URLs to test concurrency limit
	urls := make([]string, 20)
	for i := range urls {
		urls[i] = mockServer.URL
	}

	// Store URLs
	postBody := map[string]interface{}{
		"urls": urls,
	}
	bodyBytes, _ := json.Marshal(postBody)
	req := httptest.NewRequest(http.MethodPost, "/concurrency-test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "expected status 201")

	// Fetch URLs and measure time
	start := time.Now()
	getReq := httptest.NewRequest(http.MethodGet, "/concurrency-test", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	duration := time.Since(start)

	require.Equal(t, http.StatusOK, getW.Code, "expected status 200")

	// With 20 URLs and max 10 concurrent, should take at least 200ms (2 batches of 100ms each)
	// But less than 2 seconds (all sequential would be 2 seconds)
	require.True(t, duration >= 200*time.Millisecond, "should take at least 200ms due to concurrency limit")
	require.True(t, duration < 2*time.Second, "should not take 2 seconds (all sequential)")

	var resp map[string]interface{}
	err := json.Unmarshal(getW.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to decode response")

	results, ok := resp["results"].([]interface{})
	require.True(t, ok, "expected results to be a slice")
	require.Len(t, results, 20, "expected 20 results")

	// All results should be successful
	for i, result := range results {
		resultMap := result.(map[string]interface{})
		require.Equal(t, float64(200), resultMap["status_code"], "result %d should have 200 status", i)
		require.Equal(t, "response", resultMap["content"], "result %d should have expected content", i)
	}
}
