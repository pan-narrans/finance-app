package app

import (
	"testing"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockReportProvider is a manual mock implementation of ports.ReportProvider.
type MockReportProvider struct {
	mock.Mock
}

func (m *MockReportProvider) GetBalanceReport(period string, filter string) (string, error) {
	args := m.Called(period, filter)
	return args.String(0), args.Error(1)
}

// MockConfigUC is a manual mock implementation of ports.ConfigurationUseCase.
type MockConfigUC struct {
	mock.Mock
}

func (m *MockConfigUC) Get() *ports.AppConfig {
	args := m.Called()
	return args.Get(0).(*ports.AppConfig)
}

func (m *MockConfigUC) Reload() error { return nil }
func (m *MockConfigUC) SetRepository(repo ports.TransactionRepository) {}
func (m *MockConfigUC) Watch()         {}
func (m *MockConfigUC) Close() error   { return nil }
func (m *MockConfigUC) ReloadWithData(settings domain.Settings, mappings domain.MappingData) {}
func (m *MockConfigUC) SaveMappings(data domain.MappingData) error { return nil }
func (m *MockConfigUC) UpdateMapping(fn func(data *domain.MappingData)) error { return nil }
func (m *MockConfigUC) LearnMapping(tx domain.Transaction, tO, sO bool, oS string) error { return nil }

func TestReportService_GetMonthlyReport_ShouldReturnSections_WhenDataExists(t *testing.T) {
	// Arrange
	mockProvider := new(MockReportProvider)
	mockConfig := new(MockConfigUC)
	svc := NewReportService(mockProvider, mockConfig)
	
	settings := domain.DefaultSettings()
	settings.RootAccounts = []string{"Expenses", "Assets"}
	
	mockConfig.On("Get").Return(&ports.AppConfig{Settings: settings})
	mockProvider.On("GetBalanceReport", "this month", "Expenses").Return("100 EUR Expenses", nil)
	mockProvider.On("GetBalanceReport", "this month", "Assets").Return("500 EUR Assets", nil)

	// Act
	sections, err := svc.GetMonthlyReport("this month")

	// Assert
	assert.NoError(t, err)
	assert.Len(t, sections, 2)
	assert.Equal(t, "Expenses", sections[0].Title)
	assert.Equal(t, "100 EUR Expenses", sections[0].Content)
}

func TestReportService_GetMonthlyReport_ShouldUseLastMonth(t *testing.T) {
	// Arrange
	mockProvider := new(MockReportProvider)
	mockConfig := new(MockConfigUC)
	svc := NewReportService(mockProvider, mockConfig)
	
	settings := domain.DefaultSettings()
	settings.RootAccounts = []string{"Expenses"}
	
	mockConfig.On("Get").Return(&ports.AppConfig{Settings: settings})
	mockProvider.On("GetBalanceReport", "last month", "Expenses").Return("200 EUR Expenses", nil)

	// Act
	sections, err := svc.GetMonthlyReport("last month")

	// Assert
	assert.NoError(t, err)
	assert.Len(t, sections, 1)
	assert.Equal(t, "200 EUR Expenses", sections[0].Content)
}

func TestReportService_GetMonthlyReport_ShouldSkipEmptySections(t *testing.T) {
	// Arrange
	mockProvider := new(MockReportProvider)
	mockConfig := new(MockConfigUC)
	svc := NewReportService(mockProvider, mockConfig)
	
	settings := domain.DefaultSettings()
	settings.RootAccounts = []string{"Expenses", "Assets"}
	
	mockConfig.On("Get").Return(&ports.AppConfig{Settings: settings})
	mockProvider.On("GetBalanceReport", "this month", "Expenses").Return("", nil)
	mockProvider.On("GetBalanceReport", "this month", "Assets").Return("500 EUR Assets", nil)

	// Act
	sections, err := svc.GetMonthlyReport("this month")

	// Assert
	assert.NoError(t, err)
	assert.Len(t, sections, 1)
	assert.Equal(t, "Assets", sections[0].Title)
}
