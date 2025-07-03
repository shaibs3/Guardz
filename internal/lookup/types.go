package lookup

// DbType represents the supported database types
type DbType string

const (
	DbTypeCSV      DbType = "csv"
	DbTypePostgres DbType = "postgres"
	// Add more database types here as you implement them
	// DbTypeMemory   DbType = "memory"
)

// String returns the string representation of the database type
func (dt DbType) String() string {
	return string(dt)
}

// IsValid checks if the database type is supported
func (dt DbType) IsValid() bool {
	switch dt {
	case DbTypeCSV, DbTypePostgres:
		return true
	default:
		return false
	}
}

type DbProviderConfig struct {
	DbType       DbType                 `json:"dbtype"`
	ExtraDetails map[string]interface{} `json:"extra_details"`
}
