package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/shaibs3/Guardz/internal/lookup"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestHandler() *DynamicHandler {
	return NewDynamicHandler(lookup.NewInMemoryProvider())
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
			w.Write([]byte("Final destination reached"))
		case "/single-redirect":
			// Single redirect
			http.Redirect(w, r, "/final", http.StatusMovedPermanently)
		case "/no-redirect":
			// No redirect
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("No redirect"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

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
			w.Write([]byte("Should not reach here"))
		}
	}))
	defer mockServer.Close()

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
