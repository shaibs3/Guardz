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

	dbConn, err := sql.Open("postgres", connStr)
	if err != nil {
		pgLogger.Error("failed to open Postgres connection", zap.Error(err))
		return nil, fmt.Errorf("failed to open Postgres connection: %w", err)
	}

	if err := dbConn.Ping(); err != nil {
		pgLogger.Error("failed to ping Postgres", zap.Error(err))
		return nil, fmt.Errorf("failed to ping Postgres: %w", err)
	}

	// Automatically create tables if they do not exist
	if _, err := dbConn.Exec(db.Schema); err != nil {
		pgLogger.Error("failed to create initial tables", zap.Error(err))
		return nil, fmt.Errorf("failed to create initial tables: %w", err)
	}

	pgLogger.Info("Postgres provider initialized successfully")
	return &PostgresProvider{
		db:     dbConn,
		logger: pgLogger,
	}, nil
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
