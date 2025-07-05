package handlers

import (
	"github.com/gorilla/mux"
	"github.com/shaibs3/Guardz/internal/finder"
	"go.uber.org/zap"
)

// IPHandler handles IP-related requests
type IPHandler struct {
	ipFinder *finder.IpFinder
}

// NewIPHandler creates a new IP handler
func NewIPHandler(ipFinder *finder.IpFinder) *IPHandler {
	return &IPHandler{
		ipFinder: ipFinder,
	}
}

// RegisterRoutes registers the routes for this handler
func (h *IPHandler) RegisterRoutes(router *mux.Router, logger *zap.Logger) {
	router.HandleFunc("/v1/find-country", h.ipFinder.FindIpHandler).Methods("GET")
}
