package router

import (
	"golang.org/x/time/rate"
	"net/http"
	"strconv"
	"time"

	"github.com/shaibs3/Guardz/internal/telemetry"

	"github.com/shaibs3/Guardz/internal/finder"
	"github.com/shaibs3/Guardz/internal/service_health"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// Router handles all routing logic and middleware setup
type Router struct {
	router        *mux.Router
	rateLimiter   *rate.Limiter
	logger        *zap.Logger
	routerMetrics *HTTPMetrics
}

// NewRouter creates a new router instance
func NewRouter(rateLimiter *rate.Limiter, telemetry *telemetry.Telemetry, logger *zap.Logger) *Router {
	httpMetrics := NewHTTPMetrics(telemetry.Meter, logger.Named("metrics"))

	r := &Router{
		router:        mux.NewRouter(),
		rateLimiter:   rateLimiter,
		logger:        logger.Named("router"),
		routerMetrics: httpMetrics,
	}
	return r
}

// CreateServer creates and configures a complete HTTP server with all routes and middleware
func (router *Router) CreateServer(port string, ipFinder *finder.IpFinder) *http.Server {
	router.logger.Info("creating HTTP server", zap.String("port", port))

	// Setup routes
	router.setupRoutes(ipFinder)

	// Setup middleware
	handler := router.setupMiddleware()

	// Create server
	srv := &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	router.logger.Info("server configuration",
		zap.String("addr", srv.Addr),
		zap.Duration("read_timeout", srv.ReadTimeout),
		zap.Duration("write_timeout", srv.WriteTimeout),
		zap.Duration("idle_timeout", srv.IdleTimeout))

	return srv
}

// setupRoutes configures all application routes (private method)
func (router *Router) setupRoutes(ipFinder *finder.IpFinder) {
	router.logger.Info("setting up application routes")

	// Health check endpoints
	router.router.HandleFunc("/health/live", service_health.LivenessHandler(router.logger)).Methods("GET", "HEAD")
	router.router.HandleFunc("/health/ready", service_health.ReadinessHandler(router.logger)).Methods("GET", "HEAD")

	// Metrics endpoint
	router.router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// API endpoints
	router.router.HandleFunc("/v1/find-country", ipFinder.FindIpHandler).Methods("GET")

	router.logger.Info("routes configured successfully")
}

// setupMiddleware configures rate limiting and metrics middleware (private method)
func (router *Router) setupMiddleware() http.Handler {
	router.logger.Info("setting up middleware")

	// Apply middlewares in order: metrics -> rate limiting -> router
	metricsHandler := router.metricsMiddleware(router.logger.Named("metrics"))(router.router)
	rateLimitedRouter := router.rateLimitMiddleware(metricsHandler)

	router.logger.Info("middleware configured successfully")
	return rateLimitedRouter
}

// MetricsMiddleware creates middleware for comprehensive HTTP metrics
func (router *Router) metricsMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Increment active requests
			if router.routerMetrics.ActiveRequests != nil {
				router.routerMetrics.ActiveRequests.Add(r.Context(), 1)
				defer router.routerMetrics.ActiveRequests.Add(r.Context(), -1)
			}

			// Create response writer wrapper to capture status code
			wrappedWriter := &ResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call next handler
			next.ServeHTTP(wrappedWriter, r)

			// Record metrics
			duration := time.Since(start)

			attrs := []attribute.KeyValue{
				attribute.String("method", r.Method),
				attribute.String("path", r.URL.Path),
				attribute.Int("status_code", wrappedWriter.statusCode),
			}

			// Record request duration
			if router.routerMetrics.RequestDuration != nil {
				router.routerMetrics.RequestDuration.Record(r.Context(), duration.Seconds(), metric.WithAttributes(attrs...))
			}

			// Record request count
			if router.routerMetrics.RequestCount != nil {
				router.routerMetrics.RequestCount.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			}

			// Record error requests (4xx, 5xx status codes)
			if router.routerMetrics.ErrorRequests != nil && (wrappedWriter.statusCode >= 400) {
				errorAttrs := []attribute.KeyValue{
					attribute.String("method", r.Method),
					attribute.String("path", r.URL.Path),
					attribute.String("status_code", strconv.Itoa(wrappedWriter.statusCode)),
				}
				router.routerMetrics.ErrorRequests.Add(r.Context(), 1, metric.WithAttributes(errorAttrs...))
			}

			// Record response status
			if router.routerMetrics.ResponseStatus != nil {
				statusAttrs := []attribute.KeyValue{
					attribute.String("status_code", strconv.Itoa(wrappedWriter.statusCode)),
				}
				router.routerMetrics.ResponseStatus.Add(r.Context(), 1, metric.WithAttributes(statusAttrs...))
			}

			logger.Info("request completed",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status_code", wrappedWriter.statusCode),
				zap.Duration("duration", duration),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

func (router *Router) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health check and metrics endpoints
		// in normal app i would have created diffrent http servers listening on different ports for app logic, metrics and health endpoints
		if r.URL.Path == "/metrics" || r.URL.Path == "/health/live" || r.URL.Path == "/health/ready" {
			next.ServeHTTP(w, r)
			return
		}

		if !router.rateLimiter.Allow() {
			if router.routerMetrics != nil && router.routerMetrics.RateLimitedRequests != nil {
				router.routerMetrics.RateLimitedRequests.Add(r.Context(), 1)
			}
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
