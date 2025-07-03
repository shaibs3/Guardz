package router

import (
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type HTTPMetrics struct {
	RequestDuration     metric.Float64Histogram
	RequestCount        metric.Int64Counter
	ErrorRequests       metric.Int64Counter
	ResponseStatus      metric.Int64Counter
	ActiveRequests      metric.Int64UpDownCounter
	RateLimitedRequests metric.Int64Counter
}

func NewHTTPMetrics(meter metric.Meter, logger *zap.Logger) *HTTPMetrics {
	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		logger.Error("failed to create request duration metric", zap.Error(err))
	}

	requestCount, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.Error("failed to create request count metric", zap.Error(err))
	}

	errorRequests, err := meter.Int64Counter(
		"http_error_requests_total",
		metric.WithDescription("Total number of HTTP error requests (4xx, 5xx)"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.Error("failed to create error requests metric", zap.Error(err))
	}

	responseStatus, err := meter.Int64Counter(
		"http_response_status_total",
		metric.WithDescription("Total number of HTTP responses by status code"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.Error("failed to create response status metric", zap.Error(err))
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"http_requests_in_flight",
		metric.WithDescription("Number of HTTP requests currently in flight"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.Error("failed to create active requests metric", zap.Error(err))
	}

	rateLimitedRequests, err := meter.Int64Counter(
		"http_rate_limited_requests_total",
		metric.WithDescription("Total number of HTTP requests that were rate limited"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.Error("failed to create rate limited requests metric", zap.Error(err))
	}

	return &HTTPMetrics{
		RequestDuration:     requestDuration,
		RequestCount:        requestCount,
		ErrorRequests:       errorRequests,
		ResponseStatus:      responseStatus,
		ActiveRequests:      activeRequests,
		RateLimitedRequests: rateLimitedRequests,
	}
}
