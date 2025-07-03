package lookup

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type CSVProvider struct {
	data   map[string]record
	mu     sync.RWMutex
	logger *zap.Logger
}

type record struct {
	city    string
	country string
}

func NewCSVProvider(config DbProviderConfig, logger *zap.Logger, meter metric.Meter) (*CSVProvider, error) {
	if meter != nil {
		InitLookupMetrics(meter)
	}
	// Create a named logger for the CSV provider
	csvLogger := logger.Named("csv")

	path, ok := config.ExtraDetails["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path is required for CSV provider")
	}
	csvLogger.Info("initializing CSV provider", zap.String("path", path))

	file, err := os.Open(path) // #nosec G304
	if err != nil {
		csvLogger.Error("failed to open CSV file", zap.Error(err), zap.String("path", path))
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		csvLogger.Error("failed to read CSV file", zap.Error(err), zap.String("path", path))
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	data := make(map[string]record)
	validRows := 0
	skippedRows := 0

	for i, row := range rows {
		if len(row) < 3 {
			csvLogger.Debug("skipping invalid row", zap.Int("row_number", i+1), zap.Int("columns", len(row)))
			skippedRows++
			continue // skip invalid rows
		}
		ip := row[0]
		city := row[1]
		country := row[2]
		data[ip] = record{city: city, country: country}
		validRows++
	}

	csvLogger.Info("CSV provider initialized successfully",
		zap.String("path", path),
		zap.Int("total_rows", len(rows)),
		zap.Int("valid_rows", validRows),
		zap.Int("skipped_rows", skippedRows),
		zap.Int("unique_ips", len(data)))

	return &CSVProvider{
		data:   data,
		logger: csvLogger,
	}, nil
}

func (p *CSVProvider) Lookup(ctx context.Context, ip string) (string, string, error) {
	start := time.Now()
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.logger.Debug("looking up IP", zap.String("ip", ip))

	rec, ok := p.data[ip]
	if !ok {
		IncLookupErrors(context.Background())
		RecordLookupDuration(ctx, time.Since(start).Seconds())
		p.logger.Debug("IP not found in database", zap.String("ip", ip))
		return "", "", fmt.Errorf("IP not found")
	}

	RecordLookupDuration(ctx, time.Since(start).Seconds())

	p.logger.Debug("IP lookup successful",
		zap.String("ip", ip),
		zap.String("city", rec.city),
		zap.String("country", rec.country))

	return rec.city, rec.country, nil
}
