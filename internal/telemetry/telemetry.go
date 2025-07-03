package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// Telemetry handles OpenTelemetry initialization and metrics
type Telemetry struct {
	Meter  metric.Meter
	logger *zap.Logger
}

// New initializes OpenTelemetry with Prometheus exporter
func NewTelemetry(logger *zap.Logger) (*Telemetry, error) {
	logger = logger.Named("telemetry")

	// Initialize Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	// Create meter provider
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)
	otel.SetMeterProvider(provider)

	logger.Info("OpenTelemetry metrics initialized with Prometheus exporter")

	// Initialize HTTP metrics
	meter := otel.GetMeterProvider().Meter("guardz")

	return &Telemetry{
		Meter:  meter,
		logger: logger,
	}, nil
}
