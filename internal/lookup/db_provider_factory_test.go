package lookup

//
//import (
//	"context"
//	"fmt"
//	"os"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//	"go.uber.org/zap"
//)
//
//func TestDbType_IsValid(t *testing.T) {
//	assert.True(t, DbTypeCSV.IsValid())
//	assert.False(t, DbType("invalid").IsValid())
//}
//
//func TestDbType_String(t *testing.T) {
//	assert.Equal(t, "csv", DbTypeCSV.String())
//}
//
//func TestGetDbProvider_CSV(t *testing.T) {
//	logger := zap.NewNop()
//
//	// Create a temporary CSV file for testing
//	tempFile, err := os.CreateTemp("", "test-*.csv")
//	require.NoError(t, err)
//	defer func() {
//		_ = os.Remove(tempFile.Name())
//	}()
//
//	// Write test data to the temporary file
//	testData := "IP,CITY,COUNTRY\n1.2.3.4,New York,USA\n5.6.7.8,London,UK\n"
//	_, err = tempFile.WriteString(testData)
//	require.NoError(t, err)
//	require.NoError(t, tempFile.Close())
//
//	// Test CSV provider configuration
//	configJSON := `{
//		"dbtype": "csv",
//		"extra_details": {
//			"file_path": "` + tempFile.Name() + `"
//		}
//	}`
//
//	factory := NewDbProviderFactory(logger, nil)
//	provider, err := factory.CreateProvider(configJSON)
//	require.NoError(t, err)
//	assert.NotNil(t, provider)
//
//	// Test that the provider actually works
//	city, country, err := provider.Lookup(context.Background(), "1.2.3.4")
//	require.NoError(t, err)
//	assert.Equal(t, "New York", city)
//	assert.Equal(t, "USA", country)
//}
//
//func TestGetDbProvider_InvalidDbType(t *testing.T) {
//	logger := zap.NewNop()
//
//	// Test invalid database type
//	configJSON := `{
//		"dbtype": "invalid_type",
//		"extra_details": {}
//	}`
//
//	factory := NewDbProviderFactory(logger, nil)
//	provider, err := factory.CreateProvider(configJSON)
//	assert.Error(t, err)
//	assert.Nil(t, provider)
//	assert.Contains(t, err.Error(), "unsupported database type")
//}
//
//func TestGetDbProvider_MissingRequiredField(t *testing.T) {
//	logger := zap.NewNop()
//
//	// Test missing required field for CSV provider
//	configJSON := `{
//		"dbtype": "csv",
//		"extra_details": {}
//	}`
//
//	factory := NewDbProviderFactory(logger, nil)
//	provider, err := factory.CreateProvider(configJSON)
//	assert.Error(t, err)
//	assert.Nil(t, provider)
//	assert.Contains(t, err.Error(), "file_path is required for CSV provider")
//}
//
//func TestGetDbProvider_InvalidJSON(t *testing.T) {
//	logger := zap.NewNop()
//
//	// Test invalid JSON
//	configJSON := `{ invalid json }`
//
//	factory := NewDbProviderFactory(logger, nil)
//	provider, err := factory.CreateProvider(configJSON)
//	assert.Error(t, err)
//	assert.Nil(t, provider)
//	assert.Contains(t, err.Error(), "failed to parse database configuration JSON")
//}
//
//func TestNewFactory(t *testing.T) {
//	logger := zap.NewNop()
//	factory := NewDbProviderFactory(logger, nil)
//
//	assert.NotNil(t, factory)
//	assert.Equal(t, logger.Named("factory"), factory.logger)
//}
//
//func TestFactory_CreateProvider_InvalidJSON(t *testing.T) {
//	logger := zap.NewNop()
//	factory := NewDbProviderFactory(logger, nil)
//
//	configJSON := `{ invalid json }`
//
//	provider, err := factory.CreateProvider(configJSON)
//	assert.Error(t, err)
//	assert.Nil(t, provider)
//	assert.Contains(t, err.Error(), "failed to parse database configuration JSON")
//}
//
//func TestFactory_CreateProvider_InvalidDbType(t *testing.T) {
//	logger := zap.NewNop()
//	factory := NewDbProviderFactory(logger, nil)
//
//	configJSON := `{"dbtype": "invalid", "extra_details": {}}`
//
//	provider, err := factory.CreateProvider(configJSON)
//	assert.Error(t, err)
//	assert.Nil(t, provider)
//	assert.Contains(t, err.Error(), "unsupported database type: invalid")
//}
//
//func TestFactory_CreateProvider_MissingRequiredField(t *testing.T) {
//	logger := zap.NewNop()
//	factory := NewDbProviderFactory(logger, nil)
//
//	configJSON := `{"dbtype": "csv", "extra_details": {}}`
//
//	provider, err := factory.CreateProvider(configJSON)
//	assert.Error(t, err)
//	assert.Nil(t, provider)
//	assert.Contains(t, err.Error(), "file_path is required for CSV provider")
//}
//
//// MockFactory demonstrates how easy it is to create a mock factory for testing
//type MockFactory struct {
//	providers map[string]DbProvider
//	logger    *zap.Logger
//}
//
//func NewMockFactory(logger *zap.Logger) *MockFactory {
//	return &MockFactory{
//		providers: make(map[string]DbProvider),
//		logger:    logger,
//	}
//}
//
//func (m *MockFactory) CreateProvider(configJSON string) (DbProvider, error) {
//	// For testing, we can return predefined providers based on config
//	// This demonstrates the flexibility of the interface approach
//	if provider, exists := m.providers[configJSON]; exists {
//		return provider, nil
//	}
//	return nil, fmt.Errorf("no provider configured for: %s", configJSON)
//}
//
//func (m *MockFactory) SetProvider(configJSON string, provider DbProvider) {
//	m.providers[configJSON] = provider
//}
//
//func TestMockFactory(t *testing.T) {
//	logger := zap.NewNop()
//	mockFactory := NewMockFactory(logger)
//	ctx := context.Background()
//
//	// Create a mock provider
//	mockProvider := &MockProvider{
//		data: map[string]record{
//			"1.2.3.4": {city: "Test City", country: "Test Country"},
//		},
//	}
//
//	configJSON := `{"dbtype": "csv", "extra_details": {"file_path": "test.csv"}}`
//	mockFactory.SetProvider(configJSON, mockProvider)
//
//	// Test the mock factory
//	provider, err := mockFactory.CreateProvider(configJSON)
//	require.NoError(t, err)
//	assert.NotNil(t, provider)
//
//	city, country, err := provider.Lookup(ctx, "1.2.3.4")
//	require.NoError(t, err)
//	assert.Equal(t, "Test City", city)
//	assert.Equal(t, "Test Country", country)
//}
//
//// MockProvider for testing
//type MockProvider struct {
//	data map[string]record
//}
//
//func (m *MockProvider) Lookup(ctx context.Context, ip string) (string, string, error) {
//	if rec, exists := m.data[ip]; exists {
//		return rec.city, rec.country, nil
//	}
//	return "", "", fmt.Errorf("IP not found")
//}
