package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTransactionRepository is a manual mock implementation of ports.TransactionRepository.
type MockTransactionRepository struct {
	mock.Mock
}

func (mockTransactionRepository *MockTransactionRepository) Create(transaction domain.Transaction) error {
	args := mockTransactionRepository.Called(transaction)
	return args.Error(0)
}

func (mockTransactionRepository *MockTransactionRepository) FindByCode(code string) (*domain.Transaction, error) {
	args := mockTransactionRepository.Called(code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (mockTransactionRepository *MockTransactionRepository) Update(transaction domain.Transaction) error {
	args := mockTransactionRepository.Called(transaction)
	return args.Error(0)
}

func (mockTransactionRepository *MockTransactionRepository) Delete(code string) error {
	args := mockTransactionRepository.Called(code)
	return args.Error(0)
}

func (mockTransactionRepository *MockTransactionRepository) GetAccounts() ([]string, error) {
	args := mockTransactionRepository.Called()
	return args.Get(0).([]string), args.Error(1)
}

func TestTransactionService_Add_ShouldSaveTransaction_WhenInputIsValid(t *testing.T) {
	// Arrange
	mockTransactionRepository := new(MockTransactionRepository)
	transactionService := NewTransactionService(mockTransactionRepository)

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

	mockTransactionRepository.On("Create", transaction).Return(nil)

	// Act
	err := transactionService.Add(transaction)

	// Assert
	assert.NoError(t, err)
	mockTransactionRepository.AssertExpectations(t)
}

func TestTransactionService_Add_ShouldReturnError_WhenDomainValidationFails(t *testing.T) {
	// Arrange
	mockTransactionRepository := new(MockTransactionRepository)
	transactionService := NewTransactionService(mockTransactionRepository)

	// Transaction missing date to trigger validation error
	transaction := domain.Transaction{
		Description: "Invalid",
	}

	// Act
	err := transactionService.Add(transaction)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction")
	mockTransactionRepository.AssertNotCalled(t, "Create", mock.Anything)
}

func TestTransactionService_Update_ShouldReplaceTransaction_WhenInputIsValid(t *testing.T) {
	// Arrange
	mockTransactionRepository := new(MockTransactionRepository)
	transactionService := NewTransactionService(mockTransactionRepository)

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

	mockTransactionRepository.On("Update", transaction).Return(nil)

	// Act
	err := transactionService.Update(transaction)

	// Assert
	assert.NoError(t, err)
	mockTransactionRepository.AssertExpectations(t)
}

func TestTransactionService_Delete_ShouldRemoveTransaction_WhenCodeIsValid(t *testing.T) {
	// Arrange
	mockTransactionRepository := new(MockTransactionRepository)
	transactionService := NewTransactionService(mockTransactionRepository)

	mockTransactionRepository.On("Delete", "CODE123").Return(nil)

	// Act
	err := transactionService.Delete("CODE123")

	// Assert
	assert.NoError(t, err)
	mockTransactionRepository.AssertExpectations(t)
}

func TestTransactionService_Delete_ShouldReturnError_WhenCodeIsEmpty(t *testing.T) {
	// Arrange
	mockTransactionRepository := new(MockTransactionRepository)
	transactionService := NewTransactionService(mockTransactionRepository)

	// Act
	err := transactionService.Delete("")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code must be provided")
}

func TestTransactionService_Add_ShouldReturnError_WhenPersistenceFails(t *testing.T) {
	// Arrange
	mockTransactionRepository := new(MockTransactionRepository)
	transactionService := NewTransactionService(mockTransactionRepository)

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

	mockTransactionRepository.On("Create", transaction).Return(fmt.Errorf("disk full"))

	// Act
	err := transactionService.Add(transaction)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save transaction")
	mockTransactionRepository.AssertExpectations(t)
}
