package app

import (
	"testing"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionParserService_ParseText_ShouldReturnTransaction_WhenValidInputProvided(t *testing.T) {
	// Arrange
	data := domain.MappingData{
		Accounts: map[string]string{
			"CASH":   "Assets:Cash",
			"COFFEE": "Expenses:Food:Coffee",
		},
	}
	settings := domain.Settings{
		DefaultCurrency: "EUR",
	}
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	// Inject test data
	manager.ReloadWithData(settings, data)

	svc := NewTransactionParserService(manager)

	// Act
	tx, err := svc.ParseText("cash 3.50 morning coffee", domain.OriginTelegram)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "morning coffee", tx.Description)
	require.Len(t, tx.Postings, 2)
	assert.Equal(t, "Expenses:Food:Coffee", tx.Postings[0].Account)
	assert.Equal(t, 3.50, *tx.Postings[0].Amount)
	assert.Equal(t, "EUR", tx.Postings[0].Currency)
	assert.Equal(t, "Assets:Cash", tx.Postings[1].Account)
	assert.Nil(t, tx.Postings[1].Amount)
	assert.Equal(t, domain.OriginTelegram, tx.Metadata.Origin)
}

func TestTransactionParserService_ParseText_ShouldHandleMinimalInput_WhenSourceIsMissing(t *testing.T) {
	// Arrange
	settings := domain.Settings{
		DefaultAssetAccount: "Assets:Checking:Main",
		DefaultCurrency:     "USD",
	}
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	manager.ReloadWithData(settings, domain.MappingData{})

	svc := NewTransactionParserService(manager)

	// Act
	tx, err := svc.ParseText("10 lunch", "Bot")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 10.0, *tx.Postings[0].Amount)
	assert.Equal(t, "Assets:Checking:Main", tx.Postings[1].Account)
}

func TestTransactionParserService_ParseText_ShouldHandleCommaAsDecimalSeparator(t *testing.T) {
	// Arrange
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	svc := NewTransactionParserService(manager)

	// Act
	tx, err := svc.ParseText("12,50 dinner", "Test")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 12.50, *tx.Postings[0].Amount)
}

func TestTransactionParserService_ParseText_ShouldFallbackToIncomeSource_WhenSourceIsUnknown(t *testing.T) {
	// Arrange
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	svc := NewTransactionParserService(manager)

	// Act
	tx, err := svc.ParseText("alex 50 gift", "Test")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Income:Alex", tx.Postings[1].Account)
}

func TestTransactionParserService_ParseText_ShouldReturnError_WhenFormatIsInvalid(t *testing.T) {
	// Arrange
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	svc := NewTransactionParserService(manager)

	// Act & Assert
	_, err := svc.ParseText("just-description", "Test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "format not recognized")
}

func TestTransactionParserService_HashID_ShouldBeConsistent(t *testing.T) {
	// Arrange
	svc := NewTransactionParserService(nil)

	// Act
	id1 := svc.hashID("test-data")
	id2 := svc.hashID("test-data")

	// Assert
	assert.Equal(t, id1, id2)
	assert.Len(t, id1, 8)
}

func TestTransactionParserService_HashID_ShouldReturnEmpty_WhenInputIsEmpty(t *testing.T) {
	// Arrange
	svc := NewTransactionParserService(nil)

	// Act & Assert
	assert.Empty(t, svc.hashID(""))
}
