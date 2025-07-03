package service_health

import (
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
	"time"
)

// LivenessHandler checks if the service is alive
func LivenessHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := HealthResponse{
			Status:    "alive",
			Timestamp: time.Now(),
			Service:   "guardz",
		}

		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			logger.Error("failed to encode liveness response", zap.Error(err))
			return
		}

		logger.Info("liveness check completed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr))
	}
}
