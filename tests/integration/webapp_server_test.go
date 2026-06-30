package integration

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRefresher implements telegram.MessageRefresher
type MockRefresher struct {
	mock.Mock
}

func (m *MockRefresher) RefreshDraftMessage(userID int64) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockRefresher) StartImportReview(userID int64, pending []domain.Transaction) error {
	args := m.Called(userID, pending)
	return args.Error(0)
}

// MockConfigUseCase implements ports.ConfigurationUseCase
type MockConfigUseCase struct {
	mock.Mock
}

func (m *MockConfigUseCase) Get() *ports.AppConfig {
	args := m.Called()
	return args.Get(0).(*ports.AppConfig)
}

func (m *MockConfigUseCase) SaveMappings(data domain.MappingData) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockConfigUseCase) UpdateMapping(fn func(data *domain.MappingData)) error {
	args := m.Called(fn)
	return args.Error(0)
}

func (m *MockConfigUseCase) LearnMapping(transaction domain.Transaction, t, s bool, os string) error {
	args := m.Called(transaction, t, s, os)
	return args.Error(0)
}

// MockTransactionUseCase implements ports.TransactionUseCase
type MockTransactionUseCase struct {
	mock.Mock
}

func (m *MockTransactionUseCase) Add(transaction domain.Transaction) error {
	args := m.Called(transaction)
	return args.Error(0)
}

func (m *MockTransactionUseCase) Update(transaction domain.Transaction) error {
	args := m.Called(transaction)
	return args.Error(0)
}

func (m *MockTransactionUseCase) Delete(code string) error {
	args := m.Called(code)
	return args.Error(0)
}

func (m *MockTransactionUseCase) GetByCode(code string) (*domain.Transaction, error) {
	args := m.Called(code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (m *MockTransactionUseCase) List(limit int) ([]domain.Transaction, error) {
	args := m.Called(limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Transaction), args.Error(1)
}

// MockReportUseCase implements ports.ReportUseCase
type MockReportUseCase struct {
	mock.Mock
}

func (m *MockReportUseCase) GetMonthlyReport(period string) ([]ports.ReportSection, error) {
	args := m.Called(period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.ReportSection), args.Error(1)
}

// MockImportUseCase implements ports.ImportUseCase
type MockImportUseCase struct {
	mock.Mock
}

func (m *MockImportUseCase) Import(filePath string) (*ports.ImportSummary, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.ImportSummary), args.Error(1)
}

func generateValidInitData(botToken string, userID int64) string {
	userJSON := fmt.Sprintf(`{"id":%d,"first_name":"Test","username":"testuser"}`, userID)
	params := url.Values{}
	params.Set("query_id", "AAG6_...")
	params.Set("user", userJSON)
	params.Set("auth_date", "1718290000")

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var dataCheckString strings.Builder
	for i, k := range keys {
		if i > 0 {
			dataCheckString.WriteString("\n")
		}
		dataCheckString.WriteString(fmt.Sprintf("%s=%s", k, params.Get(k)))
	}

	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)

	mac = hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString.String()))
	hash := hex.EncodeToString(mac.Sum(nil))

	params.Set("hash", hash)
	return params.Encode()
}

func TestWebAppServer_GetAccounts_ShouldReturnAccountsFromConfig(t *testing.T) {
	// Arrange
	mockConfig := new(MockConfigUseCase)
	sessionManager := telegram.NewSessionManager()
	mockRefresher := new(MockRefresher)
	botToken := "fake-token"

	server := telegram.NewWebAppServer(0, botToken, mockConfig, new(MockTransactionUseCase), new(MockReportUseCase), new(MockImportUseCase), sessionManager, mockRefresher)

	mappingService := domain.NewMappingService(domain.MappingData{}, []string{"Assets:Checking", "Expenses:Food"})
	appConfig := &ports.AppConfig{
		Settings: domain.Settings{
			RootAccounts:    []string{"Assets", "Expenses"},
			TelegramUserIDs: []int64{12345},
		},
		Mappings: mappingService,
	}
	mockConfig.On("Get").Return(appConfig)

	ts := httptest.NewServer(server.Router())
	defer ts.Close()

	initData := generateValidInitData(botToken, 12345)
	req, err := http.NewRequest("GET", ts.URL+"/api/accounts", nil)
	require.NoError(t, err)
	req.Header.Set("X-TMA-Init-Data", initData)

	// Act
	resp, err := http.DefaultClient.Do(req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Accounts []string `json:"accounts"`
		Roots    []string `json:"roots"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result.Accounts, "Assets:Checking")
	assert.Contains(t, result.Roots, "Assets")
}

func TestWebAppServer_SelectAccount_ShouldUpdateSessionAndRefreshMessage(t *testing.T) {
	// Arrange
	mockConfig := new(MockConfigUseCase)
	sessionManager := telegram.NewSessionManager()
	mockRefresher := new(MockRefresher)
	botToken := "fake-token"
	userID := int64(12345)

	server := telegram.NewWebAppServer(0, botToken, mockConfig, new(MockTransactionUseCase), new(MockReportUseCase), new(MockImportUseCase), sessionManager, mockRefresher)

	mockConfig.On("Get").Return(&ports.AppConfig{
		Settings: domain.Settings{
			TelegramUserIDs: []int64{userID},
		},
	})

	sessionManager.Set(userID, &telegram.UserSession{
		Draft: domain.Transaction{
			Postings: []domain.Posting{{}, {}},
		},
	})

	initData := generateValidInitData(botToken, userID)
	payload := map[string]string{
		"account": "Assets:Checking:OpenBank",
		"type":    "target",
	}
	body, _ := json.Marshal(payload)

	mockRefresher.On("RefreshDraftMessage", userID).Return(nil)

	ts := httptest.NewServer(server.Router())
	defer ts.Close()

	req, err := http.NewRequest("POST", ts.URL+"/api/select", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TMA-Init-Data", initData)

	// Act
	resp, err := http.DefaultClient.Do(req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	sess, ok := sessionManager.Get(userID)
	require.True(t, ok)
	assert.Equal(t, "Assets:Checking:Openbank", sess.Draft.Postings[0].Account)
	assert.True(t, sess.TargetOverridden)
	mockRefresher.AssertExpectations(t)
}
