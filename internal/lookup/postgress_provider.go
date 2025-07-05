package lookup

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/lib/pq"
	"github.com/shaibs3/Guardz/internal/db_model"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"time"
)

type PostgresProvider struct {
	db     *sql.DB
	logger *zap.Logger
	cb     *gobreaker.CircuitBreaker
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
	if _, err := dbConn.Exec(db_model.Schema); err != nil {
		pgLogger.Error("failed to create initial tables", zap.Error(err))
		return nil, fmt.Errorf("failed to create initial tables: %w", err)
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
		db:     dbConn,
		logger: pgLogger,
		cb:     cb,
	}, nil
}

// StoreURLsForPath stores a list of URLs for a given path (atomic, bulk insert, with circuit breaker and retry)
func (p *PostgresProvider) StoreURLsForPath(ctx context.Context, path string, urls []string) error {
	var opErr error
	err := retry.Do(
		func() error {
			_, err := p.cb.Execute(func() (interface{}, error) {
				tx, err := p.db.BeginTx(ctx, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to begin transaction: %w", err)
				}
				defer func() {
					if err != nil {
						if rerr := tx.Rollback(); rerr != nil {
							p.logger.Warn("tx.Rollback failed", zap.Error(rerr))
						}
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
					if rerr := tx.Rollback(); rerr != nil {
						p.logger.Warn("tx.Rollback failed", zap.Error(rerr))
					}
					return nil, fmt.Errorf("failed to get or create path: %w", err)
				}

				if len(urls) == 0 {
					if cerr := tx.Commit(); cerr != nil {
						p.logger.Warn("tx.Commit failed", zap.Error(cerr))
						return nil, fmt.Errorf("failed to commit transaction: %w", cerr)
					}
					return nil, nil
				}

				stmt, err := tx.Prepare(pq.CopyIn("urls", "path_id", "url"))
				if err != nil {
					if rerr := tx.Rollback(); rerr != nil {
						p.logger.Warn("tx.Rollback failed", zap.Error(rerr))
					}
					return nil, fmt.Errorf("failed to prepare bulk insert: %w", err)
				}
				defer func() {
					cerr := stmt.Close()
					if cerr != nil {
						p.logger.Warn("stmt.Close failed", zap.Error(cerr))
					}
				}()

				for _, url := range urls {
					_, err = stmt.Exec(pathID, url)
					if err != nil {
						if rerr := tx.Rollback(); rerr != nil {
							p.logger.Warn("tx.Rollback failed", zap.Error(rerr))
						}
						return nil, fmt.Errorf("failed to exec bulk insert: %w", err)
					}
				}
				_, err = stmt.Exec()
				if err != nil {
					if rerr := tx.Rollback(); rerr != nil {
						p.logger.Warn("tx.Rollback failed", zap.Error(rerr))
					}
					return nil, fmt.Errorf("failed to finalize bulk insert: %w", err)
				}
				if cerr := tx.Commit(); cerr != nil {
					p.logger.Warn("tx.Commit failed", zap.Error(cerr))
					return nil, fmt.Errorf("failed to commit transaction: %w", cerr)
				}
				return nil, nil
			})
			opErr = err
			return err
		},
		retry.Attempts(3),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			p.logger.Warn("retrying StoreURLsForPath", zap.Uint("attempt", n+1), zap.Error(err))
		}),
	)
	if err != nil {
		return err
	}
	return opErr
}

// GetURLsByPath returns all URL records for a given path (with circuit breaker and retry)
func (p *PostgresProvider) GetURLsByPath(ctx context.Context, path string) ([]db_model.URLRecord, error) {
	var result []db_model.URLRecord
	var opErr error
	err := retry.Do(
		func() error {
			res, err := p.cb.Execute(func() (interface{}, error) {
				recs, err := p.getURLsByPath(path)
				return recs, err
			})
			if err == nil {
				result = res.([]db_model.URLRecord)
			}
			opErr = err
			return err
		},
		retry.Attempts(3),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			p.logger.Warn("retrying GetURLsByPath", zap.Uint("attempt", n+1), zap.Error(err))
		}),
	)
	if err != nil {
		return nil, err
	}
	return result, opErr
}

func (p *PostgresProvider) getURLsByPath(path string) ([]db_model.URLRecord, error) {
	var records []db_model.URLRecord
	rows, err := p.db.Query(`
		SELECT u.id, u.path_id, u.url
		FROM urls u
		JOIN paths p ON u.path_id = p.id
		WHERE p.path = $1
		ORDER BY u.id ASC
	`, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := rows.Close()
		if cerr != nil {
			fmt.Print("Error closing rows: ", cerr)
		}
	}()
	for rows.Next() {
		var rec db_model.URLRecord
		err := rows.Scan(&rec.ID, &rec.PathID, &rec.URL)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}
