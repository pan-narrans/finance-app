package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/a-perez/finance-app/internal/adapters/secondary/excel"
	"github.com/a-perez/finance-app/internal/adapters/secondary/ledger"
	"github.com/a-perez/finance-app/internal/app"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mappingServiceConstructor(data domain.MappingData, discoveredAccounts []string) ports.MappingProvider {
	return domain.NewMappingService(data, discoveredAccounts)
}

func TestImportService_Integration_ShouldImportTransactions_WhenValidBankFileProvided(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()

	// 1. Setup Config Files
	configPath := filepath.Join(tempDir, "config.json")
	mappingsPath := filepath.Join(tempDir, "mappings.json")
	configJSON := `{
		"settings": {
			"default_currency": "EUR", 
			"imagin_bank_account": "Assets:Checking:ImaginBank",
			"default_income_account": "Income:Unknown",
			"default_expense_account": "Expenses:Unknown"
		}
	}`
	_ = os.WriteFile(configPath, []byte(configJSON), 0644)
	
	// Add mapping to avoid "Unknown" and ensure it gets Added
	mappingsJSON := `{
		"accounts": {
			"SUPERMARKET": "Expenses:Food",
			"SALARY": "Income:Job"
		}
	}`
	_ = os.WriteFile(mappingsPath, []byte(mappingsJSON), 0644)

	configManager, err := config.NewManager(configPath, mappingsPath, mappingServiceConstructor)
	require.NoError(t, err)

	// 2. Setup Ledger Repository
	ledgerPath := filepath.Join(tempDir, "test.ledger")
	repo := ledger.NewTransactionFileRepository(ledgerPath, configManager, ledger.NewLedgerFormatter())

	// 3. Setup Services
	txService := app.NewTransactionService(repo)
	parserFactory := excel.NewParserFactory(configManager)
	importService := app.NewImportService(txService, parserFactory)

	// 4. Create dummy Bank File (ImaginBank CSV)
	bankFilePath := filepath.Join(tempDir, "2026_imagin.csv")
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"SUPERMARKET;15/04/2026;-25,50EUR;1000,00EUR\n" +
		"SALARY;14/04/2026;1500,00EUR;2500,00EUR\n"
	_ = os.WriteFile(bankFilePath, []byte(csvContent), 0644)

	// Act
	summary, err := importService.Import(bankFilePath)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 2, summary.Added)
	assert.Empty(t, summary.Errors)

	// Verify ledger content
	ledgerContent, err := os.ReadFile(ledgerPath)
	require.NoError(t, err)
	assert.Contains(t, string(ledgerContent), "SUPERMARKET")
	assert.Contains(t, string(ledgerContent), "SALARY")
}

func TestImportService_Integration_ShouldHandleUnknownAccounts_WhenMappingsAreMissing(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, "config.json")
	mappingsPath := filepath.Join(tempDir, "mappings.json")
	_ = os.WriteFile(configPath, []byte(`{"settings": {"default_currency": "EUR", "imagin_bank_account": "Assets:Checking:ImaginBank", "default_expense_account": "Expenses:Unknown"}}`), 0644)
	_ = os.WriteFile(mappingsPath, []byte(`{"mappings": []}`), 0644)

	configManager, err := config.NewManager(configPath, mappingsPath, mappingServiceConstructor)
	require.NoError(t, err)

	ledgerPath := filepath.Join(tempDir, "test.ledger")
	repo := ledger.NewTransactionFileRepository(ledgerPath, configManager, ledger.NewLedgerFormatter())
	txService := app.NewTransactionService(repo)
	parserFactory := excel.NewParserFactory(configManager)
	importService := app.NewImportService(txService, parserFactory)

	// ImaginBank CSV with unknown account (no mapping for 'UNKNOWN_SHOP')
	bankFilePath := filepath.Join(tempDir, "imagin.csv")
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"UNKNOWN_SHOP;15/04/2026;-50,00EUR;950,00EUR\n"
	_ = os.WriteFile(bankFilePath, []byte(csvContent), 0644)

	// Act
	summary, err := importService.Import(bankFilePath)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Total)
	assert.Equal(t, 1, len(summary.Pending))
	assert.Equal(t, 0, summary.Added)
	assert.Equal(t, "UNKNOWN_SHOP", summary.Pending[0].Description)
}
