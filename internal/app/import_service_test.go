package app

import (
	"errors"
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/app/ports"
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
func (mockUseCase *MockTransactionUseCase) AddWithMappings(
	transaction domain.Transaction,
	targetOverride bool,
	sourceOverride bool,
	originalSource string,
) error {
	return mockUseCase.Called(transaction, targetOverride, sourceOverride, originalSource).Error(0)
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

type MockFileParserProvider struct {
	mock.Mock
}

func (m *MockFileParserProvider) GetParser(filePath string) (ports.BankParser, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.BankParser), args.Error(1)
}

func TestImportService_Import_ShouldHandlePartialSuccess(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	mockProvider := new(MockFileParserProvider)
	service := NewImportService(mockUseCase, mockProvider)

	transactions := []domain.Transaction{
		{Description: "Success", Code: "CODE1"},
		{Description: "Fail", Code: "CODE2"},
	}

	mockProvider.On("GetParser", "file.xls").Return(mockParser, nil)
	mockParser.On("Parse", "file.xls").Return(transactions, nil)

	// First transaction succeeds (Add)
	mockUseCase.On("GetByCode", "CODE1").Return(nil, nil).Once()
	mockUseCase.On("Add", transactions[0]).Return(nil).Once()

	// Second transaction fails on Add
	mockUseCase.On("GetByCode", "CODE2").Return(nil, nil).Once()
	mockUseCase.On("Add", transactions[1]).Return(errors.New("db error")).Once()

	// Act
	summary, err := service.Import("file.xls")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 1, summary.Added)
	assert.Equal(t, 1, summary.Failed)
	assert.Len(t, summary.Errors, 1)

	mockUseCase.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

func TestImportService_Import_ShouldHandleUpdates(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	mockProvider := new(MockFileParserProvider)
	service := NewImportService(mockUseCase, mockProvider)

	transactions := []domain.Transaction{
		{Description: "Existing", Code: "EXISTING_CODE"},
	}
	existing := &domain.Transaction{Code: "EXISTING_CODE"}

	mockProvider.On("GetParser", "file.xls").Return(mockParser, nil)
	mockParser.On("Parse", "file.xls").Return(transactions, nil)
	mockUseCase.On("GetByCode", "EXISTING_CODE").Return(existing, nil)
	mockUseCase.On("Update", transactions[0]).Return(nil)

	// Act
	summary, err := service.Import("file.xls")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, summary.Updated)
	assert.Equal(t, 0, summary.Added)
	mockUseCase.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

func TestImportService_Import_ShouldReturnError_WhenParserFails(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	mockProvider := new(MockFileParserProvider)
	service := NewImportService(mockUseCase, mockProvider)

	mockProvider.On("GetParser", "invalid.xls").Return(mockParser, nil)
	mockParser.On("Parse", "invalid.xls").Return([]domain.Transaction{}, errors.New("parse error"))

	// Act
	summary, err := service.Import("invalid.xls")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, summary)
	assert.Contains(t, err.Error(), "parse error")
}

func TestImportService_Import_ShouldHandleLookupErrorAsRowFailure(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	mockProvider := new(MockFileParserProvider)
	service := NewImportService(mockUseCase, mockProvider)

	transactions := []domain.Transaction{{Code: "BAD_LOOKUP"}}
	mockProvider.On("GetParser", "file.xls").Return(mockParser, nil)
	mockParser.On("Parse", "file.xls").Return(transactions, nil)
	mockUseCase.On("GetByCode", "BAD_LOOKUP").Return(nil, errors.New("io error"))

	// Act
	summary, err := service.Import("file.xls")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, summary.Failed)
	assert.Contains(t, summary.Errors[0].Error(), "lookup failed")
}

func TestImportService_Import_ShouldSortTransactionsChronologically(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	mockProvider := new(MockFileParserProvider)
	service := NewImportService(mockUseCase, mockProvider)

	date1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	date3 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	// Unsorted input (newest first)
	transactions := []domain.Transaction{
		{Date: date3, Description: "Newest", Code: "CODE3"},
		{Date: date2, Description: "Middle", Code: "CODE2"},
		{Date: date1, Description: "Oldest", Code: "CODE1"},
	}

	mockProvider.On("GetParser", "file.xls").Return(mockParser, nil)
	mockParser.On("Parse", "file.xls").Return(transactions, nil)

	// Expect calls in chronological order: CODE1, CODE2, CODE3
	mockUseCase.On("GetByCode", "CODE1").Return(nil, nil).Once()
	mockUseCase.On("Add", mock.MatchedBy(func(tx domain.Transaction) bool { return tx.Code == "CODE1" })).Return(nil).Once()

	mockUseCase.On("GetByCode", "CODE2").Return(nil, nil).Once()
	mockUseCase.On("Add", mock.MatchedBy(func(tx domain.Transaction) bool { return tx.Code == "CODE2" })).Return(nil).Once()

	mockUseCase.On("GetByCode", "CODE3").Return(nil, nil).Once()
	mockUseCase.On("Add", mock.MatchedBy(func(tx domain.Transaction) bool { return tx.Code == "CODE3" })).Return(nil).Once()

	// Act
	_, err := service.Import("file.xls")

	// Assert
	assert.NoError(t, err)
	mockUseCase.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

func TestImportService_Import_ShouldIdentifyUnknownTransactions(t *testing.T) {
	// Arrange
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	mockProvider := new(MockFileParserProvider)
	service := NewImportService(mockUseCase, mockProvider)

	transactions := []domain.Transaction{
		{
			Description: "Known",
			Code:        "KNOWN",
			Postings: []domain.Posting{
				{Account: "Assets:Bank", Amount: new(float64), Currency: "EUR"},
				{Account: "Expenses:Food"},
			},
		},
		{
			Description: "Unknown",
			Code:        "UNKNOWN",
			Postings: []domain.Posting{
				{Account: "Assets:Bank", Amount: new(float64), Currency: "EUR"},
				{Account: "Expenses:Unknown"},
			},
		},
	}

	mockProvider.On("GetParser", "file.xls").Return(mockParser, nil)
	mockParser.On("Parse", "file.xls").Return(transactions, nil)

	// Only "Known" should be processed
	mockUseCase.On("GetByCode", "KNOWN").Return(nil, nil).Once()
	mockUseCase.On("Add", mock.MatchedBy(func(tx domain.Transaction) bool { return tx.Code == "KNOWN" })).Return(nil).Once()

	// Act
	summary, err := service.Import("file.xls")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 1, summary.Added)
	assert.Equal(t, 1, len(summary.Pending))
	assert.Equal(t, "UNKNOWN", summary.Pending[0].Code)

	mockUseCase.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}
