package lookup

import (
	"context"
	"go.opentelemetry.io/otel/metric"
	"sync"
)

var (
	lookupDuration metric.Float64Histogram
	lookupErrors   metric.Int64Counter
	metricsInit    sync.Once
)

func InitLookupMetrics(meter metric.Meter) {
	metricsInit.Do(func() {
		lookupDuration, _ = meter.Float64Histogram(
			"ip_lookup_duration_seconds",
			metric.WithDescription("Duration of IP lookup in seconds"),
		)
		lookupErrors, _ = meter.Int64Counter(
			"ip_lookup_errors_total",
			metric.WithDescription("Total number of IP lookup errors"),
		)
	})
}

func RecordLookupDuration(ctx context.Context, seconds float64) {
	if lookupDuration != nil {
		lookupDuration.Record(ctx, seconds)
	}
}

func IncLookupErrors(ctx context.Context) {
	if lookupErrors != nil {
		lookupErrors.Add(ctx, 1)
	}
}
