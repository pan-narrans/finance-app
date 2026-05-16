package app

import (
	"testing"

	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextParserService_ParseText_ShouldReturnTransaction_WhenValidInputProvided(t *testing.T) {
	// Arrange
	data := config.MappingData{
		Sources:  map[string]string{"cash": "Assets:Cash"},
		Accounts: map[string]string{"coffee": "Expenses:Food:Coffee"},
	}
	cfg := config.Config{
		DefaultCurrency: "EUR",
	}
	svc := NewTextParserService(domain.NewMappingService(data, cfg), cfg)

	// Act
	tx, err := svc.ParseText("cash 3.50 morning coffee", "Telegram")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "morning coffee", tx.Description)
	require.Len(t, tx.Postings, 2)
	assert.Equal(t, "Expenses:Food:Coffee", tx.Postings[0].Account)
	assert.Equal(t, 3.50, *tx.Postings[0].Amount)
	assert.Equal(t, "EUR", tx.Postings[0].Currency)
	assert.Equal(t, "Assets:Cash", tx.Postings[1].Account)
	assert.Nil(t, tx.Postings[1].Amount)
	assert.Equal(t, "Telegram", tx.Metadata.Origin)
	assert.NotEmpty(t, tx.Metadata.ID)
	assert.NotEmpty(t, tx.Code)
}

func TestTextParserService_ParseText_ShouldHandleMinimalInput_WhenSourceIsMissing(t *testing.T) {
	// Arrange
	cfg := config.Config{
		DefaultBotAccount: "Assets:Checking:Main",
		DefaultCurrency:   "USD",
	}
	svc := NewTextParserService(domain.NewMappingService(config.MappingData{}, cfg), cfg)

	// Act
	tx, err := svc.ParseText("10 lunch", "Bot")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 10.0, *tx.Postings[0].Amount)
	assert.Equal(t, "Assets:Checking:Main", tx.Postings[1].Account)
}

func TestTextParserService_ParseText_ShouldHandleCommaAsDecimalSeparator(t *testing.T) {
	// Arrange
	svc := NewTextParserService(domain.NewMappingService(config.MappingData{}, config.Config{}), config.Config{})

	// Act
	tx, err := svc.ParseText("12,50 dinner", "Test")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 12.50, *tx.Postings[0].Amount)
}

func TestTextParserService_ParseText_ShouldFallbackToIncomeSource_WhenSourceIsUnknown(t *testing.T) {
	// Arrange
	svc := NewTextParserService(domain.NewMappingService(config.MappingData{}, config.Config{}), config.Config{})

	// Act
	tx, err := svc.ParseText("alex 50 gift", "Test")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Income:Alex", tx.Postings[1].Account)
}

func TestTextParserService_ParseText_ShouldReturnError_WhenFormatIsInvalid(t *testing.T) {
	// Arrange
	svc := NewTextParserService(domain.NewMappingService(config.MappingData{}, config.Config{}), config.Config{})

	// Act & Assert
	_, err := svc.ParseText("just-description", "Test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "format not recognized")

	_, err = svc.ParseText("invalid-amount description", "Test")
	assert.Error(t, err)
}

func TestTextParserService_HashID_ShouldBeConsistent(t *testing.T) {
	// Arrange
	svc := NewTextParserService(nil, config.Config{})

	// Act
	id1 := svc.hashID("test-data")
	id2 := svc.hashID("test-data")

	// Assert
	assert.Equal(t, id1, id2)
	assert.Len(t, id1, 8)
}

func TestTextParserService_HashID_ShouldReturnEmpty_WhenInputIsEmpty(t *testing.T) {
	// Arrange
	svc := NewTextParserService(nil, config.Config{})

	// Act & Assert
	assert.Empty(t, svc.hashID(""))
}
