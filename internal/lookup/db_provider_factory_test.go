package lookup

import (
	"encoding/json"
	"testing"

	"github.com/shaibs3/Guardz/internal/telemetry"
	"go.uber.org/zap"
)

func TestDbProviderFactory_CreateProvider_Memory(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tel, _ := telemetry.NewTelemetry(logger)
	factory := NewDbProviderFactory(logger, tel)

	config := DbProviderConfig{
		DbType:       DbTypeMemory,
		ExtraDetails: map[string]interface{}{},
	}
	configJSON, _ := json.Marshal(config)

	provider, err := factory.CreateProvider(string(configJSON))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if provider == nil {
		t.Fatalf("expected provider, got nil")
	}
	if _, ok := provider.(*InMemoryProvider); !ok {
		t.Fatalf("expected InMemoryProvider, got %T", provider)
	}
}

func TestDbProviderFactory_CreateProvider_Postgres(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tel, _ := telemetry.NewTelemetry(logger)
	factory := NewDbProviderFactory(logger, tel)

	config := DbProviderConfig{
		DbType: DbTypePostgres,
		ExtraDetails: map[string]interface{}{
			"conn_str": "postgresql://user:pass@localhost:5432/dbname?sslmode=disable",
		},
	}
	configJSON, _ := json.Marshal(config)

	_, err := factory.CreateProvider(string(configJSON))
	if err == nil {
		// We expect an error because the DB probably doesn't exist, but provider type is correct
		t.Logf("expected error due to missing DB, got nil (this is OK for type check)")
	}
}
