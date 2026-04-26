package app

import (
	"errors"
	"testing"

	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTransactionUseCase struct {
	mock.Mock
}

func (mockUseCase *MockTransactionUseCase) Add(transaction domain.Transaction) error {
	return mockUseCase.Called(transaction).Error(0)
}
func (mockUseCase *MockTransactionUseCase) Update(transaction domain.Transaction) error {
	return mockUseCase.Called(transaction).Error(0)
}
func (mockUseCase *MockTransactionUseCase) Delete(code string) error {
	return mockUseCase.Called(code).Error(0)
}
func (mockUseCase *MockTransactionUseCase) GetByCode(code string) (*domain.Transaction, error) {
	args := mockUseCase.Called(code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

type MockBankParser struct {
	mock.Mock
}

func (mockParser *MockBankParser) Parse(filePath string) ([]domain.Transaction, error) {
	args := mockParser.Called(filePath)
	return args.Get(0).([]domain.Transaction), args.Error(1)
}

func TestImportService_Import_ShouldHandlePartialSuccess(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	service := NewImportService(mockUseCase)

	transactions := []domain.Transaction{
		{Description: "Success", Code: "CODE1"},
		{Description: "Fail", Code: "CODE2"},
	}

	mockParser.On("Parse", "file.xls").Return(transactions, nil)

	// First transaction succeeds (Add)
	mockUseCase.On("GetByCode", "CODE1").Return(nil, nil).Once()
	mockUseCase.On("Add", transactions[0]).Return(nil).Once()

	// Second transaction fails on Add
	mockUseCase.On("GetByCode", "CODE2").Return(nil, nil).Once()
	mockUseCase.On("Add", transactions[1]).Return(errors.New("db error")).Once()

	// Act
	summary, err := service.Import(mockParser, "file.xls")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 1, summary.Added)
	assert.Equal(t, 1, summary.Failed)
	assert.Len(t, summary.Errors, 1)

	mockUseCase.AssertExpectations(t)
}

func TestImportService_Import_ShouldHandleUpdates(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	service := NewImportService(mockUseCase)

	transactions := []domain.Transaction{
		{Description: "Existing", Code: "EXISTING_CODE"},
	}
	existing := &domain.Transaction{Code: "EXISTING_CODE"}

	mockParser.On("Parse", "file.xls").Return(transactions, nil)
	mockUseCase.On("GetByCode", "EXISTING_CODE").Return(existing, nil)
	mockUseCase.On("Update", transactions[0]).Return(nil)

	// Act
	summary, err := service.Import(mockParser, "file.xls")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, summary.Updated)
	assert.Equal(t, 0, summary.Added)
	mockUseCase.AssertExpectations(t)
}

func TestImportService_Import_ShouldReturnError_WhenParserFails(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	service := NewImportService(mockUseCase)

	mockParser.On("Parse", "invalid.xls").Return([]domain.Transaction{}, errors.New("parse error"))

	// Act
	summary, err := service.Import(mockParser, "invalid.xls")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, summary)
	assert.Contains(t, err.Error(), "parse error")
}

func TestImportService_Import_ShouldHandleLookupErrorAsRowFailure(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	service := NewImportService(mockUseCase)

	transactions := []domain.Transaction{{Code: "BAD_LOOKUP"}}
	mockParser.On("Parse", "file.xls").Return(transactions, nil)
	mockUseCase.On("GetByCode", "BAD_LOOKUP").Return(nil, errors.New("io error"))

	// Act
	summary, err := service.Import(mockParser, "file.xls")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, summary.Failed)
	assert.Contains(t, summary.Errors[0].Error(), "lookup failed")
}
