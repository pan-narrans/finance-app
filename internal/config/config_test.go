package config

import (
	"github.com/a-perez/finance-app/internal/domain"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ShouldLoadValues_WhenFileExists(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")
	content := `{"default_currency": "USD", "ledger_alignment": 60}`
	_ = os.WriteFile(path, []byte(content), 0644)

	// Act
	cfg, err := LoadConfig(path)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "USD", cfg.DefaultCurrency)
	assert.Equal(t, 60, cfg.LedgerAlignment)
}

func TestLoadConfig_ShouldReturnDefaults_WhenFileNotFound(t *testing.T) {
	// Act
	cfg, err := LoadConfig("non_existent.json")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "EUR", cfg.DefaultCurrency) // Default value
}

func TestLoadMappings_ShouldLoadData_WhenFileExists(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "mappings.json")
	content := `{"accounts": {"AMZN": "Expenses:Amazon"}, "prefixes": ["Pay:"]}`
	_ = os.WriteFile(path, []byte(content), 0644)

	// Act
	data, err := LoadMappings(path)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Expenses:Amazon", data.Accounts["AMZN"])
	assert.Contains(t, data.Prefixes, "Pay:")
}

func TestLoadMappings_ShouldReturnEmpty_WhenFileNotFound(t *testing.T) {
	// Act
	data, err := LoadMappings("ghost.json")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, data.Accounts)
	assert.Empty(t, data.Accounts)
}

func TestLoadConfig_ShouldApplyEnvironmentUserIDsOnEachLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("TELEGRAM_USER_IDS", "42, 43")

	settings, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, []int64{42, 43}, settings.TelegramUserIDs)

	require.NoError(t, os.WriteFile(path, []byte(`{"telegram_user_ids":[99]}`), 0600))
	settings, err = LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, []int64{42, 43}, settings.TelegramUserIDs)

	t.Setenv("TELEGRAM_USER_IDS", "")
	settings, err = LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, []int64{99}, settings.TelegramUserIDs)
}

func TestAccountMappingRoundTrip_ShouldPreserveOtherRoleOnOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mappings.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
   "accounts": {
     "cash": "Assets:OpenBank",
     "employer": {"source":"Income:MyEmployer","target":"Assets:MyBank"}
   }
 }`), 0600))

	data, err := LoadMappings(path)
	require.NoError(t, err)
	tx := domain.Transaction{
		Description: "Coffee",
		Postings:    []domain.Posting{{Account: "Expenses:Coffee"}, {Account: "Assets:NewBank"}},
	}
	assert.True(t, data.Learn(tx, false, true, "CASH"))
	assert.False(t, data.Learn(tx, false, true, "cash"))
	require.NoError(t, WriteMappings(path, data))

	data, err = LoadMappings(path)
	require.NoError(t, err)
	service := domain.NewMappingService(data, nil)
	source, found := service.ResolveSource("CaSh")
	assert.True(t, found)
	assert.Equal(t, "Assets:NewBank", source)
	target, found := service.ResolveAccount("paid in CASH")
	assert.True(t, found)
	assert.Equal(t, "Assets:OpenBank", target)
	source, found = service.ResolveSourceDescription("transfer from EMPLOYER")
	assert.True(t, found)
	assert.Equal(t, "Income:MyEmployer", source)
}
