package lookup

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/shaibs3/Guardz/internal/db"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"strings"
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

// StoreURLsForPath stores a list of URLs for a given path (atomic, bulk insert)
func (p *PostgresProvider) StoreURLsForPath(ctx context.Context, path string, urls []string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Get or create path inside the transaction
	var pathID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO paths (path) VALUES ($1)
		ON CONFLICT (path) DO UPDATE SET path=EXCLUDED.path
		RETURNING id
	`, path).Scan(&pathID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get or create path: %w", err)
	}

	if len(urls) == 0 {
		tx.Commit()
		return nil
	}

	valueStrings := make([]string, 0, len(urls))
	valueArgs := make([]interface{}, 0, len(urls))
	for i, url := range urls {
		valueStrings = append(valueStrings, fmt.Sprintf("($1, $%d)", i+2))
		valueArgs = append(valueArgs, url)
	}
	args := append([]interface{}{pathID}, valueArgs...)
	stmt := fmt.Sprintf("INSERT INTO urls (path_id, url) VALUES %s", strings.Join(valueStrings, ","))
	_, err = tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to bulk insert urls: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// GetURLsByPath returns all URL records for a given path
func (p *PostgresProvider) GetURLsByPath(ctx context.Context, path string) ([]db.URLRecord, error) {
	return db.GetURLsByPath(p.db, path)
}
