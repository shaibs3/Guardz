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
