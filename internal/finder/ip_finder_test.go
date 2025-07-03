package finder

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider for testing
type MockProvider struct {
	data map[string]struct {
		city    string
		country string
	}
}

func (m *MockProvider) Lookup(ctx context.Context, ip string) (string, string, error) {
	if rec, exists := m.data[ip]; exists {
		return rec.city, rec.country, nil
	}
	return "", "", fmt.Errorf("IP not found")
}

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid IPv4",
			ip:      "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "valid IPv4 with zeros",
			ip:      "0.0.0.0",
			wantErr: false,
		},
		{
			name:    "valid IPv4 localhost",
			ip:      "127.0.0.1",
			wantErr: false,
		},
		{
			name:    "valid IPv6",
			ip:      "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "valid IPv6 localhost",
			ip:      "::1",
			wantErr: false,
		},
		{
			name:    "empty IP",
			ip:      "",
			wantErr: true,
			errMsg:  "IP address is required",
		},
		{
			name:    "invalid IP format",
			ip:      "invalid-ip",
			wantErr: true,
			errMsg:  "invalid IP address format: invalid-ip",
		},
		{
			name:    "IP with letters",
			ip:      "192.168.1.abc",
			wantErr: true,
			errMsg:  "invalid IP address format: 192.168.1.abc",
		},
		{
			name:    "IP with out of range numbers",
			ip:      "192.168.1.256",
			wantErr: true,
			errMsg:  "invalid IP address format: 192.168.1.256",
		},
		{
			name:    "incomplete IPv4",
			ip:      "192.168.1",
			wantErr: true,
			errMsg:  "invalid IP address format: 192.168.1",
		},
		{
			name:    "incomplete IPv6",
			ip:      "2001:db8:",
			wantErr: true,
			errMsg:  "invalid IP address format: 2001:db8:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIP(tt.ip)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIpFinder_FindIpHandler_ValidIP(t *testing.T) {
	// Create mock provider
	mockProvider := &MockProvider{
		data: map[string]struct {
			city    string
			country string
		}{
			"192.168.1.1": {city: "New York", country: "USA"},
		},
	}

	// Create IP finder
	ipFinder := NewIpFinder(mockProvider)

	// Create test request
	req := httptest.NewRequest("GET", "/v1/find-country?ip=192.168.1.1", nil)
	w := httptest.NewRecorder()

	// Call handler
	ipFinder.FindIpHandler(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "New York", response["city"])
	assert.Equal(t, "USA", response["country"])
	assert.Equal(t, "192.168.1.1", response["ip"])
}

func TestIpFinder_FindIpHandler_InvalidIP(t *testing.T) {
	mockProvider := &MockProvider{data: make(map[string]struct {
		city    string
		country string
	})}
	ipFinder := NewIpFinder(mockProvider)

	tests := []struct {
		name       string
		ip         string
		statusCode int
		errorMsg   string
	}{
		{
			name:       "empty IP",
			ip:         "",
			statusCode: http.StatusBadRequest,
			errorMsg:   "IP address is required",
		},
		{
			name:       "invalid IP format",
			ip:         "invalid-ip",
			statusCode: http.StatusBadRequest,
			errorMsg:   "invalid IP address format: invalid-ip",
		},
		{
			name:       "IP with letters",
			ip:         "192.168.1.abc",
			statusCode: http.StatusBadRequest,
			errorMsg:   "invalid IP address format: 192.168.1.abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/find-country?ip="+tt.ip, nil)
			w := httptest.NewRecorder()

			ipFinder.FindIpHandler(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
			assert.Contains(t, w.Body.String(), tt.errorMsg)
		})
	}
}

func TestIpFinder_FindIpHandler_IPNotFound(t *testing.T) {
	mockProvider := &MockProvider{data: make(map[string]struct {
		city    string
		country string
	})}
	ipFinder := NewIpFinder(mockProvider)

	req := httptest.NewRequest("GET", "/v1/find-country?ip=8.8.8.8", nil)
	w := httptest.NewRecorder()

	ipFinder.FindIpHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "IP not found")
}

func TestIpFinder_FindIpHandler_NoIPParameter(t *testing.T) {
	mockProvider := &MockProvider{data: make(map[string]struct {
		city    string
		country string
	})}
	ipFinder := NewIpFinder(mockProvider)

	req := httptest.NewRequest("GET", "/v1/find-country", nil)
	w := httptest.NewRecorder()

	ipFinder.FindIpHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "IP address is required")
}
