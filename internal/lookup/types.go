package lookup

import "github.com/shaibs3/Guardz/internal/lookup/shared"

// Re-export shared types for convenience
type DbType = shared.DbType
type DbProviderConfig = shared.DbProviderConfig

// Re-export constants
const (
	DbTypeCSV      = shared.DbTypeCSV
	DbTypePostgres = shared.DbTypePostgres
	DbTypeMemory   = shared.DbTypeMemory
	// Add more database types here as you implement them
)
