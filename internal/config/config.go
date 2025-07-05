package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Config holds all application configuration
type Config struct {
	Port        string
	RPSLimit    int
	RPSBurst    int
	IPDBConfig  string
	Environment string
	LogLevel    string
}

// Load loads configuration from environment variables
func Load(logger *zap.Logger) *Config {
	// Load .env if present (optional)
	if err := godotenv.Load(); err != nil {
		logger.Debug("no .env file found, using environment variables")
	}

	config := &Config{
		Port:        getEnv("PORT", "8080"),
		RPSLimit:    getEnvAsInt("RPS_LIMIT", 10),
		RPSBurst:    getEnvAsInt("RPS_BURST", 10),
		IPDBConfig:  os.Getenv("DB_CONFIG"),
		Environment: getEnv("ENVIRONMENT", "production"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	logger.Info("configuration loaded",
		zap.String("port", config.Port),
		zap.Int("rps_limit", config.RPSLimit),
		zap.Int("rps_burst", config.RPSBurst),
		zap.String("environment", config.Environment),
		zap.String("log_level", config.LogLevel),
	)

	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer with a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
