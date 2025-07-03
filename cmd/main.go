package main

import (
	"log"

	"github.com/shaibs3/Guardz/internal/app"
	"github.com/shaibs3/Guardz/internal/config"
	"github.com/shaibs3/Guardz/internal/logger"
	"go.uber.org/zap"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Initialize logger first (for configuration loading)
	initialLogger, err := logger.NewLogger("production", "info")
	if err != nil {
		log.Fatal("failed to initialize logger:", err)
	}
	defer func() {
		_ = initialLogger.Sync()
	}()

	// Load configuration
	cfg := config.Load(initialLogger)

	// Create application logger with proper configuration
	appLogger, err := logger.NewLogger(cfg.Environment, cfg.LogLevel)
	if err != nil {
		initialLogger.Fatal("failed to create application logger", zap.Error(err))
	}
	defer func() {
		_ = appLogger.Sync()
	}()

	// Log build info
	appLogger.Info("Build info",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("date", date),
	)

	application, err := app.NewApp(cfg, appLogger)
	if err != nil {
		appLogger.Fatal("failed to create application", zap.Error(err))
	}

	if err := application.Run(); err != nil {
		appLogger.Fatal("application failed", zap.Error(err))
	}
}
