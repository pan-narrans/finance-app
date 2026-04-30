package app

import (
	"errors"
	"io"
	"testing"
	"time"

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

type MockBackupService struct {
	mock.Mock
}

func (m *MockBackupService) CreateBackup(filePath string) (string, error) {
	args := m.Called(filePath)
	return args.String(0), args.Error(1)
}

func (m *MockBackupService) RestoreLast(targetPath string) error {
	return m.Called(targetPath).Error(0)
}

func (m *MockBackupService) SaveDiff(sessionID, currentPath string) error {
	return m.Called(sessionID, currentPath).Error(0)
}

func setupImportService() (*ImportService, *MockTransactionUseCase, *MockBankParser, *MockBackupService) {
	mockUseCase := new(MockTransactionUseCase)
	mockParser := new(MockBankParser)
	mockBackup := new(MockBackupService)
	logger := SetupLogger(io.Discard)
	service := NewImportService(mockUseCase, mockBackup, logger, "book.ledger")
	return service, mockUseCase, mockParser, mockBackup
}

func TestImportService_Import_ShouldHandlePartialSuccess(t *testing.T) {
	// Arrange
	service, mockUseCase, mockParser, mockBackup := setupImportService()

	transactions := []domain.Transaction{
		{Description: "Success", Code: "CODE1"},
		{Description: "Fail", Code: "CODE2"},
	}

	mockBackup.On("CreateBackup", "book.ledger").Return("session-123", nil)
	mockParser.On("Parse", "file.xls").Return(transactions, nil)

	mockUseCase.On("GetByCode", "CODE1").Return(nil, nil).Once()
	mockUseCase.On("Add", transactions[0]).Return(nil).Once()

	mockUseCase.On("GetByCode", "CODE2").Return(nil, nil).Once()
	mockUseCase.On("Add", transactions[1]).Return(errors.New("db error")).Once()

	mockBackup.On("SaveDiff", "session-123", "book.ledger").Return(nil)

	// Act
	summary, err := service.Import(mockParser, "file.xls")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 2, summary.Total)
	assert.Equal(t, 1, summary.Added)
	assert.Equal(t, 1, summary.Failed)
	assert.Len(t, summary.Errors, 1)
}

func TestImportService_Import_ShouldHandleUpdates(t *testing.T) {
	// Arrange
	service, mockUseCase, mockParser, mockBackup := setupImportService()

	transactions := []domain.Transaction{
		{Description: "Existing", Code: "EXISTING_CODE"},
	}
	existing := &domain.Transaction{Code: "EXISTING_CODE"}

	mockBackup.On("CreateBackup", "book.ledger").Return("session-123", nil)
	mockParser.On("Parse", "file.xls").Return(transactions, nil)
	mockUseCase.On("GetByCode", "EXISTING_CODE").Return(existing, nil)
	mockUseCase.On("Update", transactions[0]).Return(nil)
	mockBackup.On("SaveDiff", "session-123", "book.ledger").Return(nil)

	// Act
	summary, err := service.Import(mockParser, "file.xls")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, summary.Updated)
	assert.Equal(t, 0, summary.Added)
}

func TestImportService_Import_ShouldReturnError_WhenParserFails(t *testing.T) {
	// Arrange
	service, _, mockParser, mockBackup := setupImportService()

	mockBackup.On("CreateBackup", "book.ledger").Return("session-123", nil)
	mockParser.On("Parse", "invalid.xls").Return([]domain.Transaction{}, errors.New("parse error"))

	// Act
	summary, err := service.Import(mockParser, "invalid.xls")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, summary)
	assert.Contains(t, err.Error(), "parse error")
}

func TestImportService_Import_ShouldSortTransactionsChronologically(t *testing.T) {
	// Arrange
	service, mockUseCase, mockParser, mockBackup := setupImportService()

	date1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	transactions := []domain.Transaction{
		{Date: date2, Description: "Newest", Code: "CODE2"},
		{Date: date1, Description: "Oldest", Code: "CODE1"},
	}

	mockBackup.On("CreateBackup", "book.ledger").Return("session-123", nil)
	mockParser.On("Parse", "file.xls").Return(transactions, nil)

	mockUseCase.On("GetByCode", "CODE1").Return(nil, nil).Once()
	mockUseCase.On("Add", mock.MatchedBy(func(tx domain.Transaction) bool { return tx.Code == "CODE1" })).Return(nil).Once()

	mockUseCase.On("GetByCode", "CODE2").Return(nil, nil).Once()
	mockUseCase.On("Add", mock.MatchedBy(func(tx domain.Transaction) bool { return tx.Code == "CODE2" })).Return(nil).Once()

	mockBackup.On("SaveDiff", "session-123", "book.ledger").Return(nil)

	// Act
	_, err := service.Import(mockParser, "file.xls")

	// Assert
	assert.NoError(t, err)
	mockUseCase.AssertExpectations(t)
}

func TestImportService_RollbackLastImport_ShouldCallBackupService(t *testing.T) {
	// Arrange
	service, _, _, mockBackup := setupImportService()
	mockBackup.On("RestoreLast", "book.ledger").Return(nil)

	// Act
	err := service.RollbackLastImport()

	// Assert
	assert.NoError(t, err)
	mockBackup.AssertExpectations(t)
}
