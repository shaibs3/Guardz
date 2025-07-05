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
	t.Skip("Skipping Postgres provider test; not needed for unit tests.")
	// The rest of the test is intentionally skipped.
}
