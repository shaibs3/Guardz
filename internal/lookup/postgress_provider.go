package lookup

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"time"
)

type PostgresProvider struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewPostgresProvider(config DbProviderConfig, logger *zap.Logger, meter metric.Meter) (*PostgresProvider, error) {
	if meter != nil {
		InitLookupMetrics(meter)
	}
	pgLogger := logger.Named("postgres")

	connStr, ok := config.ExtraDetails["conn_str"].(string)
	if !ok {
		return nil, fmt.Errorf("conn_str is required for Postgres provider")
	}
	pgLogger.Info("initializing Postgres provider", zap.String("conn_str", connStr))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		pgLogger.Error("failed to open Postgres connection", zap.Error(err))
		return nil, fmt.Errorf("failed to open Postgres connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		pgLogger.Error("failed to ping Postgres", zap.Error(err))
		return nil, fmt.Errorf("failed to ping Postgres: %w", err)
	}

	pgLogger.Info("Postgres provider initialized successfully")
	return &PostgresProvider{
		db:     db,
		logger: pgLogger,
	}, nil
}

func (p *PostgresProvider) Lookup(ctx context.Context, ip string) (string, string, error) {
	start := time.Now()
	p.logger.Debug("looking up IP", zap.String("ip", ip))
	var city, country string
	query := "SELECT city, country FROM ip_locations WHERE ip = $1"
	err := p.db.QueryRowContext(ctx, query, ip).Scan(&city, &country)
	if err != nil {
		IncLookupErrors(ctx)
		RecordLookupDuration(ctx, time.Since(start).Seconds())
		p.logger.Error("IP not found in database", zap.String("ip", ip), zap.Error(err))
		return "", "", fmt.Errorf("IP not found: %w", err)
	}

	RecordLookupDuration(ctx, time.Since(start).Seconds())

	p.logger.Debug("IP lookup successful",
		zap.String("ip", ip),
		zap.String("city", city),
		zap.String("country", country))

	return city, country, nil
}
