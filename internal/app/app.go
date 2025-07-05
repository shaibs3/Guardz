package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shaibs3/Guardz/internal/handlers"
	"github.com/shaibs3/Guardz/internal/router"
	"golang.org/x/time/rate"

	"github.com/shaibs3/Guardz/internal/config"
	"github.com/shaibs3/Guardz/internal/lookup"
	"github.com/shaibs3/Guardz/internal/telemetry"
	"go.uber.org/zap"
)

// App represents the main application
type App struct {
	config    *config.Config
	logger    *zap.Logger
	telemetry *telemetry.Telemetry
	server    *http.Server
}

func NewApp(cfg *config.Config, logger *zap.Logger) (*App, error) {
	// Initialize telemetry
	tel, err := telemetry.NewTelemetry(logger)
	if err != nil {
		return nil, err
	}

	// Use the factory to create the DB provider
	factory := lookup.NewDbProviderFactory(logger, tel)
	var configJSON string
	if cfg.IPDBConfig == "" {
		// Default to in-memory provider
		config := lookup.DbProviderConfig{
			DbType:       lookup.DbTypeMemory,
			ExtraDetails: map[string]interface{}{},
		}
		b, _ := json.Marshal(config)
		configJSON = string(b)
	} else {
		configJSON = cfg.IPDBConfig
	}
	dbProvider, err := factory.CreateProvider(configJSON)
	if err != nil {
		return nil, err
	}

	// Initialize router with handlers
	var limiter = rate.NewLimiter(rate.Limit(cfg.RPSLimit), cfg.RPSBurst)

	// Create handlers
	handlerList := []router.Handler{
		handlers.NewDynamicHandler(dbProvider),
	}

	appRouter := router.NewRouter(limiter, tel, logger, handlerList)
	server := appRouter.CreateServer(":" + cfg.Port)

	return &App{
		config:    cfg,
		logger:    logger,
		telemetry: tel,
		server:    server,
	}, nil
}

// Start starts the application server
func (app *App) start() error {
	app.logger.Info("starting server", zap.String("port", app.config.Port))

	go func() {
		if err := app.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Fatal("server failed to start", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully shuts down the application
func (app *App) stop() error {
	app.logger.Info("shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.server.Shutdown(shutdownCtx); err != nil {
		app.logger.Error("server forced to shutdown", zap.Error(err))
		return err
	}

	app.logger.Info("server exited gracefully")
	return nil
}

// Run starts the application and waits for shutdown signals
func (app *App) Run() error {
	// Start the server
	if err := app.start(); err != nil {
		return err
	}

	// Wait for interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop the application
	return app.stop()
}
