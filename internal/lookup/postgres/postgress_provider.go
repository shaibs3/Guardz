package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/shaibs3/Guardz/internal/db_model"
	"github.com/shaibs3/Guardz/internal/lookup/shared"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresProvider struct {
	gormDB *gorm.DB
	logger *zap.Logger
	cb     *gobreaker.CircuitBreaker
}

func NewPostgresProvider(config shared.DbProviderConfig, logger *zap.Logger, meter metric.Meter) (*PostgresProvider, error) {
	pgLogger := logger.Named("postgres")

	connStr, ok := config.ExtraDetails["conn_str"].(string)
	if !ok {
		return nil, fmt.Errorf("conn_str is required for Postgres provider")
	}
	pgLogger.Info("initializing Postgres provider", zap.String("conn_str", connStr))

	// Initialize GORM
	gormDB, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open GORM connection: %w", err)
	}
	if err := gormDB.AutoMigrate(&GormPath{}, &GormURL{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "PostgresDB",
		MaxRequests: 5,
		Interval:    60 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
	})

	pgLogger.Info("Postgres provider initialized successfully")
	return &PostgresProvider{
		gormDB: gormDB,
		logger: pgLogger,
		cb:     cb,
	}, nil
}

// Example: StoreURLsForPath using GORM (for demonstration)
func (p *PostgresProvider) StoreURLsForPath(ctx context.Context, path string, urls []string) error {
	return p.gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pth GormPath
		if err := tx.Where("path = ?", path).FirstOrCreate(&pth, GormPath{Path: path}).Error; err != nil {
			return err
		}
		// Remove old URLs for idempotency
		if err := tx.Where("path_id = ?", pth.ID).Delete(&GormURL{}).Error; err != nil {
			return err
		}
		urlObjs := make([]GormURL, len(urls))
		for i, u := range urls {
			urlObjs[i] = GormURL{PathID: pth.ID, URL: u}
		}
		return tx.Create(&urlObjs).Error
	})
}

func (p *PostgresProvider) GetURLsByPath(ctx context.Context, path string) ([]db_model.URLRecord, error) {
	var pth GormPath
	if err := p.gormDB.WithContext(ctx).Where("path = ?", path).First(&pth).Error; err != nil {
		return nil, nil // Not found is not an error
	}
	var urls []GormURL
	if err := p.gormDB.WithContext(ctx).Where("path_id = ?", pth.ID).Find(&urls).Error; err != nil {
		return nil, err
	}

	// Convert GormURL to db_model.URLRecord
	records := make([]db_model.URLRecord, len(urls))
	for i, url := range urls {
		records[i] = db_model.URLRecord{
			ID:     int64(url.ID),
			PathID: int64(url.PathID),
			URL:    url.URL,
		}
	}
	return records, nil
}
