package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shaibs3/Guardz/internal/config"
	"github.com/shaibs3/Guardz/internal/finder"
	"github.com/shaibs3/Guardz/internal/handlers"
	"github.com/shaibs3/Guardz/internal/logger"
	"github.com/shaibs3/Guardz/internal/lookup"
	"github.com/shaibs3/Guardz/internal/router"
	"github.com/shaibs3/Guardz/internal/telemetry"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
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

	// Initialize telemetry
	telemetryInstance, err := telemetry.NewTelemetry(appLogger)
	if err != nil {
		appLogger.Fatal("failed to initialize telemetry", zap.Error(err))
	}

	// Initialize database provider factory
	dbProviderFactory := lookup.NewDbProviderFactory(appLogger, telemetryInstance)

	// Create database provider using the factory
	dbProvider, err := dbProviderFactory.CreateProvider(cfg.IPDBConfig)
	if err != nil {
		appLogger.Fatal("failed to initialize database provider", zap.Error(err))
	}

	// Initialize IP finder
	ipFinder := finder.NewIpFinder(dbProvider)

	// Create rate limiter
	rateLimiter := rate.NewLimiter(rate.Every(time.Second), cfg.RPSBurst)

	// Create handlers
	handlerList := []router.Handler{
		handlers.NewDynamicHandler(),
		handlers.NewIPHandler(ipFinder),
	}

	// Create router with handlers
	routerInstance := router.NewRouter(rateLimiter, telemetryInstance, appLogger, handlerList)

	// Create server
	port := fmt.Sprintf(":%s", cfg.Port)
	server := routerInstance.CreateServer(port)

	// Start server
	appLogger.Info("starting server", zap.String("port", port))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		appLogger.Fatal("server failed", zap.Error(err))
	}
}
