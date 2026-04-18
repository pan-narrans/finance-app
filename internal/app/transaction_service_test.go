package app

import (
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLedgerRepository is a manual mock implementation of ports.LedgerRepository.
type MockLedgerRepository struct {
	mock.Mock
}

func (mockLedgerRepository *MockLedgerRepository) Append(transaction domain.Transaction) error {
	args := mockLedgerRepository.Called(transaction)
	return args.Error(0)
}

func (mockLedgerRepository *MockLedgerRepository) FindByCode(code string) (*domain.Transaction, error) {
	args := mockLedgerRepository.Called(code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (mockLedgerRepository *MockLedgerRepository) Update(transaction domain.Transaction) error {
	args := mockLedgerRepository.Called(transaction)
	return args.Error(0)
}

func (mockLedgerRepository *MockLedgerRepository) Delete(code string) error {
	args := mockLedgerRepository.Called(code)
	return args.Error(0)
}

func TestTransactionService_Add_ShouldSaveTransaction_WhenInputIsValid(t *testing.T) {
	// Arrange
	mockLedgerRepository := new(MockLedgerRepository)
	transactionService := NewTransactionService(mockLedgerRepository)

	date := time.Now()
	val := 10.0
	transaction := domain.Transaction{
		Date:        date,
		Description: "Lunch",
		Postings: []domain.Posting{
			{Account: "Expenses:Food", Amount: &val, Currency: "USD"},
			{Account: "Assets:Checking", Amount: nil},
		},
	}

	mockLedgerRepository.On("Append", transaction).Return(nil)

	// Act
	err := transactionService.Add(transaction)

	// Assert
	assert.NoError(t, err)
	mockLedgerRepository.AssertExpectations(t)
}

func TestTransactionService_Add_ShouldReturnError_WhenDomainValidationFails(t *testing.T) {
	// Arrange
	mockLedgerRepository := new(MockLedgerRepository)
	transactionService := NewTransactionService(mockLedgerRepository)

	// Transaction missing date to trigger validation error
	transaction := domain.Transaction{
		Description: "Invalid",
	}

	// Act
	err := transactionService.Add(transaction)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction")
	mockLedgerRepository.AssertNotCalled(t, "Append", mock.Anything)
}

func TestTransactionService_Update_ShouldReplaceTransaction_WhenInputIsValid(t *testing.T) {
	// Arrange
	mockLedgerRepository := new(MockLedgerRepository)
	transactionService := NewTransactionService(mockLedgerRepository)

	date := time.Now()
	val := 10.0
	transaction := domain.Transaction{
		Date:        date,
		Code:        "UPDATE_CODE",
		Description: "Dinner",
		Postings: []domain.Posting{
			{Account: "Expenses:Food", Amount: &val, Currency: "USD"},
			{Account: "Assets:Checking", Amount: nil},
		},
	}

	mockLedgerRepository.On("Update", transaction).Return(nil)

	// Act
	err := transactionService.Update(transaction)

	// Assert
	assert.NoError(t, err)
	mockLedgerRepository.AssertExpectations(t)
}

func TestTransactionService_Delete_ShouldRemoveTransaction_WhenCodeIsValid(t *testing.T) {
	// Arrange
	mockLedgerRepository := new(MockLedgerRepository)
	transactionService := NewTransactionService(mockLedgerRepository)

	mockLedgerRepository.On("Delete", "CODE123").Return(nil)

	// Act
	err := transactionService.Delete("CODE123")

	// Assert
	assert.NoError(t, err)
	mockLedgerRepository.AssertExpectations(t)
}

func TestTransactionService_Delete_ShouldReturnError_WhenCodeIsEmpty(t *testing.T) {
	// Arrange
	mockLedgerRepository := new(MockLedgerRepository)
	transactionService := NewTransactionService(mockLedgerRepository)

	// Act
	err := transactionService.Delete("")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code must be provided")
}
