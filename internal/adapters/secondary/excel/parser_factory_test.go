package excel

import (
	"testing"

	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserFactory_GetParser_ShouldReturnOpenBankParser_WhenFilenameMatches(t *testing.T) {
	// Arrange
	mappingService := domain.NewMappingService(config.MappingData{}, config.Config{})
	factory := NewParserFactory(mappingService)

	// Act
	parser, err := factory.GetParser("/path/to/openbank_export.xls")

	// Assert
	require.NoError(t, err)
	assert.IsType(t, &OpenBankParser{}, parser)
}

func TestParserFactory_GetParser_ShouldReturnImaginBankParser_WhenFilenameMatches(t *testing.T) {
	// Arrange
	mappingService := domain.NewMappingService(config.MappingData{}, config.Config{})
	factory := NewParserFactory(mappingService)

	// Act
	parser, err := factory.GetParser("2026_imaginbank.csv")

	// Assert
	require.NoError(t, err)
	assert.IsType(t, &ImaginBankParser{}, parser)
}

func TestParserFactory_GetParser_ShouldReturnError_WhenNoMatchFound(t *testing.T) {
	// Arrange
	factory := NewParserFactory(nil)

	// Act
	parser, err := factory.GetParser("unknown_bank.pdf")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, parser)
	assert.Contains(t, err.Error(), "no parser found")
}

func TestParserFactory_GetParser_ShouldBeCaseInsensitive(t *testing.T) {
	// Arrange
	factory := NewParserFactory(domain.NewMappingService(config.MappingData{}, config.Config{}))

	// Act
	parser, err := factory.GetParser("OPENBANK.XLS")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, parser)
}
