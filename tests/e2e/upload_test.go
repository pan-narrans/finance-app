package e2e

import (
	"os"
	"path/filepath"
	"strings"
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

	// Act 1: Upload document
	env.sendDocument(bankFilePath, []byte(csvContent))

	// Wait for prompt
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateAwaitingImportConfirm
	}, 5*time.Second, 100*time.Millisecond, "Should prompt for bank export confirmation")

	// Act 2: Click "Yes"
	env.sendCallback(telegram.CallbackImportYes)

	// Wait for selector
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateAwaitingBankSelection
	}, 5*time.Second, 100*time.Millisecond, "Should prompt for bank selection")

	// Act 3: Select Bank
	env.sendCallback(telegram.CallbackSelectBank, "imagin")

	// Wait for session to be created for the first pending transaction (Review Flow)
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateNone && s.Draft.Description == "RESTAURANT"
	}, 5*time.Second, 100*time.Millisecond, "Session should be created for review")

	env.sendCallback(telegram.CallbackConfirm)

	// Wait for ledger update
	var content []byte
	assert.Eventually(t, func() bool {
		var err error
		content, err = os.ReadFile(env.ledgerPath)
		return err == nil && len(content) > 0
	}, 5*time.Second, 100*time.Millisecond, "Ledger file should be populated")

	// Assert
	assert.Contains(t, string(content), "RESTAURANT")
	assert.Contains(t, string(content), "45.00 EUR")
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
	_, _ = env.importService.Import(bankFilePath, "imagin")
	
	// Wait for first import to persist
	assert.Eventually(t, func() bool {
		content, _ := os.ReadFile(env.ledgerPath)
		return len(content) > 0
	}, 5*time.Second, 100*time.Millisecond, "First import should persist")

	summary, err := env.importService.Import(bankFilePath, "imagin")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Updated, "Should update instead of adding duplicate")
}

func TestE2E_BankDocumentUpload_ShouldHandleCancellation(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	bankFileName := "imagin.csv"
	bankFilePath := filepath.Join(env.tmpDir, bankFileName)
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"RESTAURANT;15/04/2026;-45,00EUR;1000,00EUR\n"
	_ = os.WriteFile(bankFilePath, []byte(csvContent), 0644)

	// Act 1: Upload
	env.sendDocument(bankFilePath, []byte(csvContent))

	// Wait for prompt
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return ok
	}, 5*time.Second, 100*time.Millisecond)

	// Act 2: Click "No"
	env.sendCallback(telegram.CallbackImportNo)

	// Assert
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return !ok
	}, 5*time.Second, 100*time.Millisecond, "Session should be deleted after cancellation")

	content, _ := os.ReadFile(env.ledgerPath)
	assert.Empty(t, strings.TrimSpace(string(content)), "Ledger should be empty after cancellation")
}

func TestE2E_BankDocumentUpload_ShouldHandleAcceptAll(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	bankFileName := "imagin.csv"
	bankFilePath := filepath.Join(env.tmpDir, bankFileName)
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"RESTAURANT;15/04/2026;-45,00EUR;1000,00EUR\n" +
		"SUPERMARKET;16/04/2026;-20,00EUR;980,00EUR\n"
	_ = os.WriteFile(bankFilePath, []byte(csvContent), 0644)

	// Act 1: Upload
	env.sendDocument(bankFilePath, []byte(csvContent))
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateAwaitingImportConfirm
	}, 5*time.Second, 100*time.Millisecond)

	env.sendCallback(telegram.CallbackImportYes)
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateAwaitingBankSelection
	}, 5*time.Second, 100*time.Millisecond)

	env.sendCallback(telegram.CallbackSelectBank, "imagin")

	// Wait for review flow
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateNone
	}, 5*time.Second, 100*time.Millisecond, "Session should be created")


	// Act 2: Accept all
	env.sendCallback(telegram.CallbackAcceptAll)

	// Assert
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return !ok
	}, 5*time.Second, 100*time.Millisecond, "Session should be deleted after Accept All")

	// Wait for ledger update
	var content []byte
	assert.Eventually(t, func() bool {
		var err error
		content, err = os.ReadFile(env.ledgerPath)
		return err == nil && strings.Contains(string(content), "RESTAURANT") && strings.Contains(string(content), "SUPERMARKET")
	}, 5*time.Second, 100*time.Millisecond, "Ledger file should contain both transactions")
}

func TestE2E_BankDocumentUpload_ShouldHandleDiscard(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	bankFileName := "imagin.csv"
	bankFilePath := filepath.Join(env.tmpDir, bankFileName)
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"RESTAURANT;15/04/2026;-45,00EUR;1000,00EUR\n" +
		"SUPERMARKET;16/04/2026;-20,00EUR;980,00EUR\n"
	_ = os.WriteFile(bankFilePath, []byte(csvContent), 0644)

	// Act 1: Upload and trigger import
	env.sendDocument(bankFilePath, []byte(csvContent))
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateAwaitingImportConfirm
	}, 5*time.Second, 100*time.Millisecond)

	env.sendCallback(telegram.CallbackImportYes)
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateAwaitingBankSelection
	}, 5*time.Second, 100*time.Millisecond)

	env.sendCallback(telegram.CallbackSelectBank, "imagin")

	// Wait for session
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateNone
	}, 5*time.Second, 100*time.Millisecond, "Session should be created")


	// Act 2: Discard first
	env.sendCallback(telegram.CallbackDiscard)

	// Wait for next in queue
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.Draft.Description == "SUPERMARKET"
	}, 5*time.Second, 100*time.Millisecond, "Should advance to next transaction after discard")

	// Act 3: Confirm second
	env.sendCallback(telegram.CallbackConfirm)

	// Assert
	assert.Eventually(t, func() bool {
		content, err := os.ReadFile(env.ledgerPath)
		if err != nil {
			return false
		}
		ledgerText := string(content)
		return !strings.Contains(ledgerText, "RESTAURANT") && strings.Contains(ledgerText, "SUPERMARKET")
	}, 5*time.Second, 100*time.Millisecond, "Ledger should contain only the second transaction")
}

func TestE2E_BankDocumentUpload_ShouldBeOverwrittenByTextMessage(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	bankFileName := "imagin.csv"
	bankFilePath := filepath.Join(env.tmpDir, bankFileName)
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"RESTAURANT;15/04/2026;-45,00EUR;1000,00EUR\n"
	_ = os.WriteFile(bankFilePath, []byte(csvContent), 0644)

	// Act 1: Upload document
	env.sendDocument(bankFilePath, []byte(csvContent))
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateAwaitingImportConfirm
	}, 5*time.Second, 100*time.Millisecond, "Session should be created for import confirm")

	// Act 2: Send text message (Interruption)
	env.sendText("10 Pizza")
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.Draft.Description == "Pizza" && s.State == telegram.StateNone
	}, 5*time.Second, 100*time.Millisecond, "Import session should be overwritten by Pizza")

	env.sendCallback(telegram.CallbackConfirm)

	// Assert
	var content []byte
	assert.Eventually(t, func() bool {
		var err error
		content, err = os.ReadFile(env.ledgerPath)
		return err == nil && strings.Contains(string(content), "Pizza") && !strings.Contains(string(content), "RESTAURANT")
	}, 5*time.Second, 100*time.Millisecond, "Only the last manual transaction should be in ledger")
}

func TestE2E_BankDocumentUpload_ShouldFollowInteractiveFlow(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)
	
	bankFileName := "unknown_name.csv"
	bankFilePath := filepath.Join(env.tmpDir, bankFileName)
	csvContent := "Concepto;Fecha;Importe;Saldo\n" +
		"RESTAURANT;15/04/2026;-45,00EUR;1000,00EUR\n"
	_ = os.WriteFile(bankFilePath, []byte(csvContent), 0644)

	// Act 1: Upload document
	env.sendDocument(bankFilePath, []byte(csvContent))

	// Expect state to be AwaitingImportConfirm
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateAwaitingImportConfirm
	}, 5*time.Second, 100*time.Millisecond, "Should prompt for bank export confirmation")

	// Act 2: Click "Yes"
	env.sendCallback(telegram.CallbackImportYes)

	// Expect state to be AwaitingBankSelection
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.State == telegram.StateAwaitingBankSelection
	}, 5*time.Second, 100*time.Millisecond, "Should prompt for bank selection")

	// Act 3: Select "Imagin"
	env.sendCallback(telegram.CallbackSelectBank, "imagin")

	// Assert: Import summary should eventually appear and first transaction should be in session
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.Draft.Description == "RESTAURANT"
	}, 5*time.Second, 100*time.Millisecond, "Import should complete and enter review flow")
}
