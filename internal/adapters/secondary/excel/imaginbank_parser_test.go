package excel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImaginBankParser_Parse_ShouldReturnTransactions_WhenValidCsvProvided(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	csvPath := filepath.Join(tempDir, "imagin.csv")
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"AYTO.DE SOTO REAL;15/04/2026;-33,00EUR;482,76EUR\n" +
		"BIZUM RECIBIDO;14/04/2026;3,50EUR;515,76EUR\n"
	_ = os.WriteFile(csvPath, []byte(csvContent), 0644)

	mappingProvider := domain.NewMappingService(domain.MappingData{}, nil)
	settings := domain.Settings{
		DefaultCurrency:       "EUR",
		ImaginBankAccount:     "Assets:Checking:ImaginBank",
		DefaultIncomeAccount:  "Income:Unknown",
		DefaultExpenseAccount: "Expenses:Unknown",
	}

	parser := NewImaginBankParser(mappingProvider, settings)

	// Act
	transactions, err := parser.Parse(csvPath)

	// Assert
	require.NoError(t, err)
	require.Len(t, transactions, 2)

	assert.Equal(t, "2026-04-15", transactions[0].Date.Format("2006-01-02"))
	assert.Equal(t, "AYTO.DE SOTO REAL", transactions[0].Description)
	// Target (Expense) first
	assert.Equal(t, "Expenses:Unknown", transactions[0].Postings[0].Account)
	assert.Equal(t, 33.00, *transactions[0].Postings[0].Amount)
	// Source (Assets) second
	assert.Equal(t, "Assets:Checking:ImaginBank", transactions[0].Postings[1].Account)
	assert.Equal(t, "Imaginbank", transactions[0].Metadata.Origin)
	assert.NotEmpty(t, transactions[0].Metadata.ID)

	assert.Equal(t, "2026-04-14", transactions[1].Date.Format("2006-01-02"))
	// Target (Assets) first
	assert.Equal(t, "Assets:Checking:ImaginBank", transactions[1].Postings[0].Account)
	assert.Equal(t, 3.50, *transactions[1].Postings[0].Amount)
	// Source (Income) second
	assert.Equal(t, "Income:Unknown", transactions[1].Postings[1].Account)
}

func TestImaginBankParser_Parse_ShouldHandleEmptyFile(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	csvPath := filepath.Join(tempDir, "empty.csv")
	_ = os.WriteFile(csvPath, []byte("Concepto;Fecha;Importe;Saldo\n"), 0644)

	mappingProvider := domain.NewMappingService(domain.MappingData{}, nil)
	settings := domain.Settings{
		DefaultCurrency:       "EUR",
		ImaginBankAccount:     "Assets:Checking:ImaginBank",
		DefaultIncomeAccount:  "Income:Unknown",
		DefaultExpenseAccount: "Expenses:Unknown",
	}

	parser := NewImaginBankParser(mappingProvider, settings)

	// Act
	transactions, err := parser.Parse(csvPath)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, transactions)
}

func TestImaginBankParser_Parse_ShouldReturnError_WhenFileNotFound(t *testing.T) {
	// Arrange
	mappingProvider := domain.NewMappingService(domain.MappingData{}, nil)
	settings := domain.Settings{
		DefaultCurrency:       "EUR",
		ImaginBankAccount:     "Assets:Checking:ImaginBank",
		DefaultIncomeAccount:  "Income:Unknown",
		DefaultExpenseAccount: "Expenses:Unknown",
	}

	parser := NewImaginBankParser(mappingProvider, settings)

	// Act
	transactions, err := parser.Parse("non-existent.csv")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, transactions)
}
