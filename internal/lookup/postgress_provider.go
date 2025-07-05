package lookup

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/shaibs3/Guardz/internal/db"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"io"
	"net/http"
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

// StoreURLsForPath stores a list of URLs for a given path
func (p *PostgresProvider) StoreURLsForPath(ctx context.Context, path string, urls []string) error {
	pathID, err := db.GetOrCreatePath(p.db, path)
	if err != nil {
		return fmt.Errorf("failed to get or create path: %w", err)
	}
	for _, url := range urls {
		// Fetch the URL content
		resp, err := fetchURL(ctx, url)
		if err != nil {
			return fmt.Errorf("failed to fetch url %s: %w", url, err)
		}
		rec := db.URLRecord{
			PathID:     pathID,
			URL:        url,
			Content:    resp.Content,
			StatusCode: resp.StatusCode,
			FetchedAt:  resp.FetchedAt,
			Error:      resp.Error,
		}
		if err := db.InsertURLRecord(p.db, rec); err != nil {
			return fmt.Errorf("failed to insert url record: %w", err)
		}
	}
	return nil
}

// GetURLsByPath returns all URL records for a given path
func (p *PostgresProvider) GetURLsByPath(ctx context.Context, path string) ([]db.URLRecord, error) {
	return db.GetURLsByPath(p.db, path)
}

// fetchURL fetches the content of a URL and returns a struct for storage
func fetchURL(ctx context.Context, url string) (struct {
	Content    string
	StatusCode int
	FetchedAt  time.Time
	Error      *string
}, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		errStr := err.Error()
		return struct {
			Content    string
			StatusCode int
			FetchedAt  time.Time
			Error      *string
		}{
			Content:    "",
			StatusCode: 0,
			FetchedAt:  time.Now(),
			Error:      &errStr,
		}, nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errStr := err.Error()
		return struct {
			Content    string
			StatusCode int
			FetchedAt  time.Time
			Error      *string
		}{
			Content:    "",
			StatusCode: resp.StatusCode,
			FetchedAt:  time.Now(),
			Error:      &errStr,
		}, nil
	}
	return struct {
		Content    string
		StatusCode int
		FetchedAt  time.Time
		Error      *string
	}{
		Content:    string(body),
		StatusCode: resp.StatusCode,
		FetchedAt:  time.Now(),
		Error:      nil,
	}, nil
}
