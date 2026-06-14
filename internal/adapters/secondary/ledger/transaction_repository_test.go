package ledger

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConfigUC struct {
	ports.ConfigurationUseCase
	alignment int
}

func (m *mockConfigUC) Get() *ports.AppConfig {
	return &ports.AppConfig{
		Settings: domain.Settings{LedgerAlignment: m.alignment},
	}
}

func TestFileRepository_Create_ShouldWriteFormattedTransactionToFile_WhenValidInputProvided(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_create_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)
	date := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)
	transaction := domain.Transaction{
		Date:        date,
		Description: "Lunch",
		Postings: []domain.Posting{
			{Account: "Expenses:Food", Amount: new(15.50), Currency: "EUR"},
			{Account: "Assets:Checking", Amount: nil},
		},
	}
	expectedContent := ";--------\n;- APRIL -\n;--------\n\n" + formatter.FormatTransaction(transaction, 52)
	expectedContent = strings.TrimSpace(expectedContent)

	// Act
	err = fileRepository.Create(transaction)

	// Assert
	assert.NoError(t, err)
	content, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, strings.TrimSpace(string(content)))
}

func TestFileRepository_FindByCode_ShouldReturnTransaction_WhenCodeExists(t *testing.T) {
	// Arrange
	tmpFile, _ := os.CreateTemp("", "test_find_*.ledger")
	defer os.Remove(tmpFile.Name())

	transaction := domain.Transaction{
		Date:        time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
		Code:        "FINDME",
		Description: "Target",
		Postings:    []domain.Posting{{Account: "A", Amount: new(10.0), Currency: "USD"}, {Account: "B", Amount: nil}},
	}
	formatter := NewLedgerFormatter()
	content := formatter.FormatTransaction(transaction, 52) + "\n"
	os.WriteFile(tmpFile.Name(), []byte(content), 0644)

	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Act
	found, err := fileRepository.FindByCode("FINDME")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "FINDME", found.Code)
}

func TestFileRepository_FindByCode_ShouldReturnNil_WhenCodeDoesNotExist(t *testing.T) {
	// Arrange
	tmpFile, _ := os.CreateTemp("", "test_find_none_*.ledger")
	defer os.Remove(tmpFile.Name())
	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Act
	found, err := fileRepository.FindByCode("NON_EXISTENT")

	// Assert
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestFileRepository_Update_ShouldReplaceExistingTransaction_WhenCodeMatches(t *testing.T) {
	// Arrange
	tmpFile, _ := os.CreateTemp("", "test_update_*.ledger")
	defer os.Remove(tmpFile.Name())

	transactionOld := domain.Transaction{
		Date:        time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
		Code:        "UPDATE_ME",
		Description: "Old",
		Postings:    []domain.Posting{{Account: "A", Amount: new(10.0), Currency: "USD"}, {Account: "B", Amount: nil}},
	}
	formatter := NewLedgerFormatter()
	content := formatter.FormatTransaction(transactionOld, 52) + "\n"
	os.WriteFile(tmpFile.Name(), []byte(content), 0644)

	transactionNew := domain.Transaction{
		Date:        time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
		Code:        "UPDATE_ME",
		Description: "New and Improved",
		Postings:    []domain.Posting{{Account: "A", Amount: new(20.0), Currency: "USD"}, {Account: "B", Amount: nil}},
	}
	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Act
	err := fileRepository.Update(transactionNew)

	// Assert
	assert.NoError(t, err)
	updatedContent, _ := os.ReadFile(tmpFile.Name())
	assert.Contains(t, string(updatedContent), "New and Improved")
	assert.NotContains(t, string(updatedContent), "Old")
}

func TestFileRepository_Update_ShouldReturnDomainError_WhenCodeIsNotFound(t *testing.T) {
	// Arrange
	tmpFile, _ := os.CreateTemp("", "test_update_fail_*.ledger")
	defer os.Remove(tmpFile.Name())
	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	transaction := domain.Transaction{Code: "GHOST_CODE"}

	// Act
	err := fileRepository.Update(transaction)

	// Assert
	assert.Error(t, err)
	// Assert
	var domainError *domain.DomainError
	ok := errors.As(err, &domainError)
	require.True(t, ok, "Error should be of type *domain.DomainError")
	assert.Equal(t, "Code", domainError.Errors[0].Field)
	assert.Contains(t, domainError.Errors[0].Message, "not found")
}

func TestFileRepository_Update_ShouldReturnError_WhenFileDoesNotExist(t *testing.T) {
	// Arrange
	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository("non_existent_folder/ledger.ledger", configUC, formatter)
	transaction := domain.Transaction{Code: "FAIL"}

	// Act
	err := fileRepository.Update(transaction)

	// Assert
	assert.Error(t, err)
}

func TestFileRepository_Delete_ShouldRemoveTransaction_WhenCodeMatches(t *testing.T) {
	// Arrange
	tmpFile, _ := os.CreateTemp("", "test_delete_*.ledger")
	defer os.Remove(tmpFile.Name())

	transaction := domain.Transaction{
		Date:        time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC),
		Code:        "DELETE_ME",
		Description: "Gone soon",
		Postings:    []domain.Posting{{Account: "A", Amount: new(10.0), Currency: "USD"}, {Account: "B", Amount: nil}},
	}
	formatter := NewLedgerFormatter()
	content := formatter.FormatTransaction(transaction, 52) + "\n"
	os.WriteFile(tmpFile.Name(), []byte(content), 0644)

	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Act
	err := fileRepository.Delete("DELETE_ME")

	// Assert
	assert.NoError(t, err)
	updatedContent, _ := os.ReadFile(tmpFile.Name())
	assert.Empty(t, string(updatedContent))
}

func TestFileRepository_Delete_ShouldReturnDomainError_WhenCodeIsNotFound(t *testing.T) {
	// Arrange
	tmpFile, _ := os.CreateTemp("", "test_delete_fail_*.ledger")
	defer os.Remove(tmpFile.Name())
	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Act
	err := fileRepository.Delete("GHOST_CODE")

	// Assert
	// Assert
	var domainError *domain.DomainError
	ok := errors.As(err, &domainError)
	require.True(t, ok, "Error should be of type *domain.DomainError")
	assert.Equal(t, "Code", domainError.Errors[0].Field)
	assert.Contains(t, domainError.Errors[0].Message, "not found")
}

func TestFileRepository_GetAccounts_ShouldReturnAccounts_WhenFileHasTransactions(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_accounts_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `
2026/01/01 Breakfast
    Expenses:Food:Morning    10.00 EUR
    Assets:Checking

2026/01/02 Games
    Expenses:Ocio:VideoGames  50.00 EUR
    Assets:Checking
`
	err = os.WriteFile(tmpFile.Name(), []byte(content), 0644)
	require.NoError(t, err)

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Act
	accounts, err := fileRepository.GetAccounts()

	// Assert
	assert.NoError(t, err)
	assert.Len(t, accounts, 3)
	assert.Contains(t, accounts, "Assets:Checking")
	assert.Contains(t, accounts, "Expenses:Food:Morning")
	assert.Contains(t, accounts, "Expenses:Ocio:VideoGames")
}

func TestFileRepository_GetBalanceReport_ShouldReturnReport_WhenFileHasTransactions(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_bal_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `
2026/01/01 Breakfast
    Expenses:Food    10.00 EUR
    Assets:Checking
`
	err = os.WriteFile(tmpFile.Name(), []byte(content), 0644)
	require.NoError(t, err)

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	fileRepository := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Act
	report, err := fileRepository.GetBalanceReport("", "")

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, report, "Expenses:Food")
	assert.Contains(t, report, "10.00 EUR")
}

func TestFileRepository_Create_ShouldSortTransactionsChronologically(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_sort_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	repo := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	txJan := domain.Transaction{Date: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Description: "Jan", Code: "JAN"}
	txFeb := domain.Transaction{Date: time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), Description: "Feb", Code: "FEB"}
	txMar := domain.Transaction{Date: time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC), Description: "Mar", Code: "MAR"}

	// Act: Add out of order
	_ = repo.Create(txFeb)
	_ = repo.Create(txMar)
	_ = repo.Create(txJan)

	// Assert
	content, _ := os.ReadFile(tmpFile.Name())
	text := string(content)

	// Check order of appearance
	janIdx := strings.Index(text, "Jan")
	febIdx := strings.Index(text, "Feb")
	marIdx := strings.Index(text, "Mar")

	assert.True(t, janIdx < febIdx, "January should be before February")
	assert.True(t, febIdx < marIdx, "February should be before March")
}

func TestFileRepository_Create_ShouldInsertMonthSeparators(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_sep_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	repo := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	txJan := domain.Transaction{Date: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Description: "Jan"}
	txFeb := domain.Transaction{Date: time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), Description: "Feb"}

	// Act
	_ = repo.Create(txJan)
	_ = repo.Create(txFeb)

	// Assert
	content, _ := os.ReadFile(tmpFile.Name())
	text := string(content)

	assert.Contains(t, text, ";--------\n;- JANUARY -\n;--------")
	assert.Contains(t, text, ";--------\n;- FEBRUARY -\n;--------")

	// Ensure Jan header is before Jan tx
	janHeaderIdx := strings.Index(text, "JANUARY")
	janTxIdx := strings.Index(text, "Jan")
	assert.True(t, janHeaderIdx < janTxIdx)

	// Ensure Feb header is between Jan and Feb txs
	febHeaderIdx := strings.Index(text, "FEBRUARY")
	febTxIdx := strings.Index(text, "Feb")
	assert.True(t, janTxIdx < febHeaderIdx)
	assert.True(t, febHeaderIdx < febTxIdx)
}

func TestFileRepository_Create_ShouldBeStableForSameDayTransactions(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_stable_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	repo := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tx1 := domain.Transaction{Date: date, Description: "First"}
	tx2 := domain.Transaction{Date: date, Description: "Second"}

	// Act
	_ = repo.Create(tx1)
	_ = repo.Create(tx2)

	// Assert
	content, _ := os.ReadFile(tmpFile.Name())
	text := string(content)

	firstIdx := strings.Index(text, "First")
	secondIdx := strings.Index(text, "Second")
	assert.True(t, firstIdx < secondIdx, "Insertion order should be preserved for same day")
}

func TestFileRepository_ShouldPreservePrologueAndEpilogue(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_prologue_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	prologue := "commodity EUR\naccount Assets:Cash\n\n"
	existingTx := "2026/01/01 Initial\n    Assets:Cash  100 EUR\n    Equity:Opening"
	epilogue := "\n\n; End of file"
	os.WriteFile(tmpFile.Name(), []byte(prologue+existingTx+epilogue), 0644)

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	repo := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	tx := domain.Transaction{Date: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Description: "Coffee"}

	// Act
	err = repo.Create(tx)

	// Assert
	assert.NoError(t, err)
	content, _ := os.ReadFile(tmpFile.Name())
	text := string(content)

	assert.Contains(t, text, "commodity EUR", "Prologue should be present")
	assert.Contains(t, text, "; End of file", "Epilogue should be present")
	assert.Contains(t, text, "Coffee", "New transaction should be present")
	assert.Contains(t, text, "Initial", "Existing transaction should be present")
}

func TestFileRepository_ShouldNotDuplicateMonthHeaders(t *testing.T) {
	// ... (existing test)
}

func TestFileRepository_ShouldPreserveInterleavedComments(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_comments_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	initial := `commodity EUR

; Global comment
account Assets:Cash

2026/01/01 * Initial
    Assets:Cash  100 EUR
    Equity:Opening

; Comment between transactions
; that usually gets lost

2026/01/02 * Coffee
    Expenses:Food   5 EUR
    Assets:Cash  (TX_CODE)

; Trailing comment
`
	os.WriteFile(tmpFile.Name(), []byte(initial), 0644)

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	repo := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Act: Update the Coffee transaction
	tx := domain.Transaction{
		Date:        time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Description: "Better Coffee",
		Code:        "TX_CODE",
		Postings: []domain.Posting{
			{Account: "Expenses:Food", Amount: new(float64)},
			{Account: "Assets:Cash", Amount: nil},
		},
	}
	*tx.Postings[0].Amount = 6.0

	err = repo.Update(tx)

	// Assert
	assert.NoError(t, err)
	content, _ := os.ReadFile(tmpFile.Name())
	text := string(content)

	assert.Contains(t, text, "; Global comment", "Global comment should be preserved")
	assert.Contains(t, text, "; Comment between transactions", "Interleaved comment should be preserved")
	assert.Contains(t, text, "; Trailing comment", "Trailing comment should be preserved")
	assert.Contains(t, text, "Better Coffee", "Update should be applied")
}

func TestFileRepository_ShouldSortPriceUpdatesChronologically(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_price_sort_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Start with a price update from the 5th
	initial := "P 2026/01/05 ROB 1.10 EUR\n\n"
	os.WriteFile(tmpFile.Name(), []byte(initial), 0644)

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	repo := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Add a transaction on the 1st
	tx1 := domain.Transaction{Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Description: "First"}

	// Add a transaction on the 10th
	tx10 := domain.Transaction{Date: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), Description: "Tenth"}

	// Act
	_ = repo.Create(tx10)
	_ = repo.Create(tx1)

	// Assert
	content, _ := os.ReadFile(tmpFile.Name())
	text := string(content)

	idx1 := strings.Index(text, "First")
	idx5 := strings.Index(text, "P 2026/01/05 ROB")
	idx10 := strings.Index(text, "Tenth")

	assert.True(t, idx1 != -1 && idx5 != -1 && idx10 != -1, "All entries must be present")
	assert.True(t, idx1 < idx5, "Transaction on 1st should be before Price on 5th")
	assert.True(t, idx5 < idx10, "Price on 5th should be before Transaction on 10th")
}

func TestFileRepository_ShouldDifferentiatePricesFromTransactions(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_diff_*.ledger")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Create a file with a transaction and an adjacent price update
	initial := `2026/01/01 Target (DELME)
    Expenses:Foo    10 EUR
    Assets:Cash

P 2026/01/02 GOLD 100 EUR
`
	os.WriteFile(tmpFile.Name(), []byte(initial), 0644)

	formatter := NewLedgerFormatter()
	configUC := &mockConfigUC{alignment: 52}
	repo := NewTransactionFileRepository(tmpFile.Name(), configUC, formatter)

	// Act
	// Delete the transaction. The price update MUST remain.
	err = repo.Delete("DELME")

	// Assert
	assert.NoError(t, err)
	content, _ := os.ReadFile(tmpFile.Name())
	text := string(content)

	assert.NotContains(t, text, "Target")
	assert.Contains(t, text, "P 2026/01/02 GOLD 100 EUR", "Price update must survive transaction deletion")
}
