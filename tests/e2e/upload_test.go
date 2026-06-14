package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_BankDocumentUpload_ShouldUpdateLedger_WhenHappyPath(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)
	
	// Create a dummy imagin bank file
	bankFileName := "imagin.csv"
	bankFilePath := filepath.Join(env.tmpDir, bankFileName)
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"RESTAURANT;15/04/2026;-45,00EUR;1000,00EUR\n"
	_ = os.WriteFile(bankFilePath, []byte(csvContent), 0644)

	// Act
	env.sendDocument(bankFilePath, []byte(csvContent))

	// Wait for session to be created for the first pending transaction
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return ok
	}, 2*time.Second, 100*time.Millisecond, "Session should be created for review")

	env.sendCallback(telegram.CallbackConfirm)

	// Wait for ledger update
	var content []byte
	assert.Eventually(t, func() bool {
		var err error
		content, err = os.ReadFile(env.ledgerPath)
		return err == nil && len(content) > 0
	}, 2*time.Second, 100*time.Millisecond, "Ledger file should be populated")

	// Assert
	assert.Contains(t, string(content), "RESTAURANT")
	assert.Contains(t, string(content), "-45.00 EUR")
}

func TestE2E_BankDocumentUpload_ShouldHandleDuplicates(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)
	
	// Add mapping to make it "Known"
	_ = os.WriteFile(filepath.Join(env.tmpDir, "mappings.json"), []byte(`{"accounts": {"RESTAURANT": "Expenses:Food"}}`), 0644)
	_ = env.configManager.Reload()

	bankFilePath := filepath.Join(env.tmpDir, "imagin.csv")
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"RESTAURANT;15/04/2026;-45,00EUR;1000,00EUR\n"
	_ = os.WriteFile(bankFilePath, []byte(csvContent), 0644)

	// Act - Import twice
	_, _ = env.importService.Import(bankFilePath)
	
	// Wait for first import to persist (it's synchronous but let's be safe)
	assert.Eventually(t, func() bool {
		content, _ := os.ReadFile(env.ledgerPath)
		return len(content) > 0
	}, 2*time.Second, 100*time.Millisecond, "First import should persist")

	summary, err := env.importService.Import(bankFilePath)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Updated, "Should update instead of adding duplicate")
}
