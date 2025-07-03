package finder

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/shaibs3/Guardz/internal/lookup"
)

type IpFinder struct {
	provider lookup.DbProvider
}

func NewIpFinder(provider lookup.DbProvider) *IpFinder {
	return &IpFinder{provider: provider}
}

// ValidateIP checks if the provided string is a valid IP address
func ValidateIP(ip string) error {
	if ip == "" {
		return fmt.Errorf("IP address is required")
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}

	// Check if it's a valid IPv4 or IPv6 address
	if parsedIP.To4() == nil && parsedIP.To16() == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	return nil
}

func (ipF *IpFinder) FindIpHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ip := r.URL.Query().Get("ip")

	// Validate IP address
	if err := ValidateIP(ip); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	city, country, err := ipF.provider.Lookup(context.Background(), ip)
	if err != nil {
		http.Error(w, `{"error":"IP not found"}`, http.StatusNotFound)
		return
	}

	resp := map[string]string{
		"city":    city,
		"country": country,
		"ip":      ip,
	}
	jsonResp, _ := json.Marshal(resp)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jsonResp)
}
