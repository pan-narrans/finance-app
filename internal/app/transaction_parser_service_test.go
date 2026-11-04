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

func TestTransactionParserService_ParseText_ShouldPreserveSourceCase_WhenSourceIsUnknown(t *testing.T) {
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
	// Preserve capitalization of the typed source keyword.
	assert.Equal(t, "Income:alex", tx.Postings[1].Account)
	assert.Equal(t, "gift", tx.Description, "Description should not include unmapped source keyword")
}

func TestTransactionParserService_ParseText_ShouldReturnError_WhenFormatIsInvalid(t *testing.T) {
	// Arrange
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, _ := config.NewManager("config.json", "mappings.json", constructor)
	svc := NewTransactionParserService(manager)

	// Act
	_, err := svc.ParseText("invalid_input_without_amounts", "Test")

	// Assert
	assert.Error(t, err, "Should return error for format not containing any amount")
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

func TestTransactionParserService_ParseText_ShouldPreserveRolesForIncomeAccounts(t *testing.T) {
	data := domain.MappingData{
		Accounts: map[string]string{"CHECKING": "Assets:Checking"},
		AccountRoles: map[string]domain.AccountRoles{
			"SALARY":     {Source: "Income:Salary", Target: "Assets:PayrollHolding"},
			"CORRECTION": {Source: "Assets:PayrollHolding", Target: "Income:Salary"},
		},
	}
	settings := domain.Settings{
		DefaultAssetAccount: "Assets:Cash",
		DefaultCurrency:     "EUR",
	}
	constructor := func(data domain.MappingData, _ []string) ports.MappingProvider {
		return domain.NewMappingService(data, nil)
	}
	manager, err := config.NewManager("config.json", "mappings.json", constructor)
	require.NoError(t, err)
	defer manager.Close()
	manager.ReloadWithData(settings, data)
	svc := NewTransactionParserService(manager)

	for _, tc := range []struct {
		input, from, to string
		keys            [2]string
	}{
		{"salary 1000 checking", "Income:Salary", "Assets:Checking", [2]string{"checking", "salary"}},
		{"checking 1000 correction", "Assets:Checking", "Income:Salary", [2]string{"correction", "checking"}},
		{"1000 correction", "Assets:Cash", "Income:Salary", [2]string{"correction", ""}},
	} {
		t.Run(tc.input, func(t *testing.T) {
			tx, err := svc.ParseText(tc.input, domain.OriginTelegram)
			require.NoError(t, err)
			require.Len(t, tx.Postings, 2)
			assert.Equal(t, tc.to, tx.Postings[0].Account)
			assert.Equal(t, tc.from, tx.Postings[1].Account)
			require.NotNil(t, tx.Postings[0].Amount)
			assert.Equal(t, 1000.0, *tx.Postings[0].Amount)
			assert.Nil(t, tx.Postings[1].Amount)
			assert.Equal(t, tc.keys, tx.MappingKeys)
		})
	}
}

func TestTransactionParserService_HashID_ShouldBeConsistent(t *testing.T) {
	// Arrange
	svc := NewTransactionParserService(nil)

	// Act
	h1 := svc.hashID("some-stable-data")
	h2 := svc.hashID("some-stable-data")
	h3 := svc.hashID("different-data")

	// Assert
	assert.Equal(t, h1, h2, "Hashes should be identical for same input")
	assert.NotEqual(t, h1, h3, "Hashes should be different for different inputs")
	assert.Len(t, h1, 8, "Hash should be exactly 8 characters long")
}

func TestTransactionParserService_HashID_ShouldReturnEmpty_WhenInputIsEmpty(t *testing.T) {
	// Arrange
	svc := NewTransactionParserService(nil)

	// Act & Assert
	assert.Empty(t, svc.hashID(""))
}

func TestTransactionParserService_ParseAmount(t *testing.T) {
	svc := NewTransactionParserService(nil)

	tests := []struct {
		input    string
		expected float64
		wantErr  bool
	}{
		{"12.50", 12.50, false},
		{"12,50", 12.50, false},
		{"1,234.56", 1234.56, false},
		{"1.234,56", 1234.56, false},
		{"1234", 1234.0, false},
		{"1.000,00", 1000.0, false},
		{"1,000.00", 1000.0, false},
		{"1.000.000,50", 1000000.50, false},
		{"1,000,000.50", 1000000.50, false},
		{"  12.50  ", 12.50, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, err := svc.parseAmount(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, val)
			}
		})
	}
}
