package ledger

import (
	"errors"
	"os"
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
	expectedContent := formatter.FormatTransaction(transaction, 52) + "\n"

	// Act
	err = fileRepository.Create(transaction)

	// Assert
	assert.NoError(t, err)
	content, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, string(content))
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
