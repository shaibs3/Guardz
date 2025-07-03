package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(environment, logLevel string) (*zap.Logger, error) {
	var config zap.Config

	switch environment {
	case "development":
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	case "production":
		config = zap.NewProductionConfig()
	default:
		config = zap.NewProductionConfig()
	}

	// Set log level
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(level)

	// Build logger
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}
