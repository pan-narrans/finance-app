package config

import (
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
