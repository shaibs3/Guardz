package lookup

import (
	"encoding/json"
	"fmt"

	"github.com/shaibs3/Guardz/internal/lookup/postgres"
	"github.com/shaibs3/Guardz/internal/lookup/shared"

	"github.com/shaibs3/Guardz/internal/telemetry"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// ProviderFactory defines the interface for creating database providers
type ProviderFactory interface {
	CreateProvider(configJSON string) (DbProvider, error)
}

// Factory implements ProviderFactory for creating database providers
type DbProviderFactory struct {
	logger    *zap.Logger
	telemetry *telemetry.Telemetry
}

func NewDbProviderFactory(logger *zap.Logger, tel *telemetry.Telemetry) *DbProviderFactory {
	return &DbProviderFactory{
		logger:    logger.Named("factory"),
		telemetry: tel,
	}
}

func (f *DbProviderFactory) CreateProvider(configJSON string) (DbProvider, error) {
	var config shared.DbProviderConfig
	f.logger.Info("parsing configuration", zap.String("configJSON", configJSON))

	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to parse database configuration JSON: %w", err)
	}

	f.logger.Info("creating database provider",
		zap.String("db_type", config.DbType.String()),
		zap.Any("extra_details", config.ExtraDetails))

	// Validate database type
	if !config.DbType.IsValid() {
		return nil, fmt.Errorf("unsupported database type: %s", config.DbType)
	}

	var telemetryMeter metric.Meter

	if f.telemetry != nil {
		telemetryMeter = f.telemetry.Meter
	} else {
		telemetryMeter = nil
	}
	switch config.DbType {
	case shared.DbTypePostgres:
		return postgres.NewPostgresProvider(config, f.logger, telemetryMeter)
	case shared.DbTypeMemory:
		f.logger.Info("Using InMemoryProvider for DB")
		return NewInMemoryProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.DbType)
	}
}
