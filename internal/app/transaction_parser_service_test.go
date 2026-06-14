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

func TestTransactionParserService_ParseText_ShouldUseDefaultAsset_WhenSourceIsUnknown(t *testing.T) {
	// Arrange
	settings := domain.Settings{
		DefaultAssetAccount: "Assets:Cash",
	}
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	manager.ReloadWithData(settings, domain.MappingData{})
	svc := NewTransactionParserService(manager)

	// Act
	tx, err := svc.ParseText("alex 50 gift", "Test")

	// Assert
	require.NoError(t, err)
	// We no longer fallback to Income:Alex automatically if it's not mapped, 
	// to avoid "Hey 10 coffee" becoming "Income:Hey".
	assert.Equal(t, "Assets:Cash", tx.Postings[1].Account)
}


func TestTransactionParserService_ParseText_ShouldReturnError_WhenFormatIsInvalid(t *testing.T) {
	// ... (existing test)
}

func TestTransactionParserService_ParseText_ShouldIgnoreConversationalNoise_AndUseDefaultAsset(t *testing.T) {
	// Arrange
	settings := domain.Settings{
		DefaultAssetAccount: "Assets:Checking:Main",
	}
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	manager.ReloadWithData(settings, domain.MappingData{})

	svc := NewTransactionParserService(manager)

	// Act: "Hey" is not a mapped source keyword
	tx, err := svc.ParseText("Hey 10 coffee", domain.OriginTelegram)

	// Assert
	require.NoError(t, err)
	// Currently it incorrectly falls back to Income:Hey, we want Assets:Checking:Main
	assert.Equal(t, "Assets:Checking:Main", tx.Postings[1].Account)
}


func TestTransactionParserService_ParseText_ShouldTreatPositiveAmountAsExpenseByDefault(t *testing.T) {
	// Arrange
	settings := domain.Settings{
		DefaultAssetAccount:   "Assets:Cash",
		DefaultExpenseAccount: "Expenses:Unknown",
		DefaultCurrency:       "EUR",
	}
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	manager.ReloadWithData(settings, domain.MappingData{})

	svc := NewTransactionParserService(manager)

	// Act: "10 coffee" - positive amount, unknown source
	tx, err := svc.ParseText("10 coffee", domain.OriginTelegram)

	// Assert
	require.NoError(t, err)
	require.Len(t, tx.Postings, 2)
	
	// Convention: Target (Debit) first. For expense, target is Expenses.
	assert.Equal(t, "Expenses:Unknown", tx.Postings[0].Account)
	assert.Equal(t, 10.0, *tx.Postings[0].Amount)
	
	// Source (Credit) second.
	assert.Equal(t, "Assets:Cash", tx.Postings[1].Account)
	assert.Nil(t, tx.Postings[1].Amount)
}

func TestTransactionParserService_ParseText_ShouldFormatIncomeCorrectly(t *testing.T) {
	// Arrange
	data := domain.MappingData{
		Accounts: map[string]string{"SALARY": "Income:Salary"},
	}
	settings := domain.Settings{
		DefaultAssetAccount: "Assets:Cash",
		DefaultCurrency:     "EUR",
	}
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	manager.ReloadWithData(settings, data)

	svc := NewTransactionParserService(manager)

	// Act: "1000 salary" - salary is an Income account
	tx, err := svc.ParseText("1000 salary", domain.OriginTelegram)

	// Assert
	require.NoError(t, err)
	require.Len(t, tx.Postings, 2)

	// Convention: Target (Debit) first. For income, target is Assets.
	assert.Equal(t, "Assets:Cash", tx.Postings[0].Account)
	assert.Equal(t, 1000.0, *tx.Postings[0].Amount)

	// Source (Credit) second.
	assert.Equal(t, "Income:Salary", tx.Postings[1].Account)
	assert.Nil(t, tx.Postings[1].Amount)
}

func TestTransactionParserService_HashID_ShouldBeConsistent(t *testing.T) {
	// ... (existing test)
}


func TestTransactionParserService_HashID_ShouldReturnEmpty_WhenInputIsEmpty(t *testing.T) {
	// Arrange
	svc := NewTransactionParserService(nil)

	// Act & Assert
	assert.Empty(t, svc.hashID(""))
}
