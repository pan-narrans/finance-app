package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockReportProvider is a manual mock implementation of ports.ReportProvider.
type MockReportProvider struct {
	mock.Mock
}

func (m *MockReportProvider) GetBalanceReport(period string) (string, error) {
	args := m.Called(period)
	return args.String(0), args.Error(1)
}

func TestReportService_GetMonthlyReport_ShouldReturnReport_WhenDataExists(t *testing.T) {
	// Arrange
	mockProvider := new(MockReportProvider)
	svc := NewReportService(mockProvider)
	expectedReport := "100 EUR Assets:Checking"

	mockProvider.On("GetBalanceReport", "this month").Return(expectedReport, nil)

	// Act
	report, err := svc.GetMonthlyReport()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedReport, report)
	mockProvider.AssertExpectations(t)
}

func TestReportService_GetMonthlyReport_ShouldReturnNoData_WhenReportIsEmpty(t *testing.T) {
	// Arrange
	mockProvider := new(MockReportProvider)
	svc := NewReportService(mockProvider)

	mockProvider.On("GetBalanceReport", "this month").Return("", nil)

	// Act
	report, err := svc.GetMonthlyReport()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "No data for this month.", report)
}

func TestReportService_GetMonthlyReport_ShouldReturnError_WhenProviderFails(t *testing.T) {
	// Arrange
	mockProvider := new(MockReportProvider)
	svc := NewReportService(mockProvider)

	mockProvider.On("GetBalanceReport", "this month").Return("", fmt.Errorf("CLI error"))

	// Act
	_, err := svc.GetMonthlyReport()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get monthly report")
}
