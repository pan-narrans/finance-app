package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram"
	"github.com/a-perez/finance-app/internal/adapters/secondary/excel"
	"github.com/a-perez/finance-app/internal/adapters/secondary/ledger"
	"github.com/a-perez/finance-app/internal/app"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/require"
	"gopkg.in/telebot.v3"
)

type e2eEnv struct {
	adapter       *telegram.TelegramAdapter
	ledgerPath    string
	configManager *config.Manager
	reportService ports.ReportUseCase
	importService ports.ImportUseCase
	userID        int64
	chatID        int64
	tmpDir        string
}

func mappingServiceConstructor(data domain.MappingData, discoveredAccounts []string) ports.MappingProvider {
	return domain.NewMappingService(data, discoveredAccounts)
}

func setupE2EEnv(t *testing.T) *e2eEnv {
	if os.Getenv("RUN_E2E") != "true" {
		t.Skip("Skipping E2E test. Set RUN_E2E=true to run.")
	}

	// Use a hardcoded fake token for E2E to ensure we never hit real API
	botToken := "123456:mock-token"

	var msgCounter int64

	// Mock Telegram API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			fmt.Fprintln(w, `{"ok":true,"result":{"id":12345,"is_bot":true,"first_name":"Test Bot","username":"test_bot"}}`)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/sendMessage") || strings.HasSuffix(r.URL.Path, "/editMessageText") || strings.HasSuffix(r.URL.Path, "/editMessageReplyMarkup") {
			newID := atomic.AddInt64(&msgCounter, 1)
			fmt.Fprintf(w, `{"ok":true,"result":{"message_id":%d,"date":1623672000,"chat":{"id":12345,"type":"private"},"text":"mock response"}}`+"\n", newID)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/answerCallbackQuery") || strings.HasSuffix(r.URL.Path, "/setMyCommands") {
			fmt.Fprintln(w, `{"ok":true,"result":true}`)
			return
		}

		// Default fallback: 404 to trigger E2E fallbacks for downloads
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, `{"ok":false,"error_code":404,"description":"Not Found"}`)
	}))

	t.Cleanup(server.Close)

	// User and Chat ID
	userID := int64(12345)
	chatID := int64(12345)

	tmpDir := t.TempDir()
	ledgerPath := filepath.Join(tmpDir, "test.ledger")
	configPath := filepath.Join(tmpDir, "config.json")
	mappingsPath := filepath.Join(tmpDir, "mappings.json")

	// Base config
	configJSON := `{
		"default_currency": "EUR",
		"imagin_bank_account": "Assets:Checking:ImaginBank",
		"default_income_account": "Income:Unknown",
		"default_expense_account": "Expenses:Unknown",
		"default_asset_account": "Assets:Cash",
		"root_accounts": ["Assets", "Expenses"]
	}`

	_ = os.WriteFile(configPath, []byte(configJSON), 0644)
	_ = os.WriteFile(mappingsPath, []byte(`{"mappings": []}`), 0644)

	configManager, err := config.NewManager(configPath, mappingsPath, mappingServiceConstructor)
	require.NoError(t, err)

	repo := ledger.NewTransactionFileRepository(ledgerPath, configManager, ledger.NewLedgerFormatter())
	txService := app.NewTransactionService(repo)
	parserService := app.NewTransactionParserService(configManager)
	reportService := app.NewReportService(repo, configManager)
	parserFactory := excel.NewParserFactory(configManager)
	importService := app.NewImportService(txService, parserFactory)

	// In E2E, we use the mock server
	settings := telebot.Settings{
		URL:    server.URL,
		Token:  botToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	tgConfig := telegram.TelegramConfig{
		Settings:      settings,
		AllowedIDs:    []int64{userID},
		BotToken:      botToken,
		WebAppBaseURL: "https://test.webapp",
		HTTPPort:      0,
	}

	adapter, err := telegram.NewTelegramAdapter(
		tgConfig,
		txService,
		parserService,
		importService,
		reportService,
		configManager,
		ledger.NewLedgerFormatter(),
	)
	require.NoError(t, err)

	adapter.RegisterHandlers()

	// Note: We don't call adapter.Start() because we don't want the real poller to compete with our manual updates
	// and we don't want to block. We will use adapter.Bot().ProcessUpdate()

	return &e2eEnv{
		adapter:       adapter,
		ledgerPath:    ledgerPath,
		configManager: configManager,
		reportService: reportService,
		importService: importService,
		userID:        userID,
		chatID:        chatID,
		tmpDir:        tmpDir,
	}
}

func (e *e2eEnv) sendText(text string) {
	e.adapter.Bot().ProcessUpdate(telebot.Update{
		Message: &telebot.Message{
			ID:     1,
			Text:   text,
			Sender: &telebot.User{ID: e.userID},
			Chat:   &telebot.Chat{ID: e.chatID, Type: telebot.ChatPrivate},
		},
	})
}

func (e *e2eEnv) sendCommand(command string) {
	e.sendText("/" + command)
}

func (e *e2eEnv) sendCallback(unique string, data ...string) {
	callbackData := unique
	if len(data) > 0 {
		callbackData += "\f" + strings.Join(data, "\f")
	}

	// Retrieve last message ID to satisfy stale message protection
	msgID := 2
	if s, ok := e.adapter.SessionManager().Get(e.userID); ok && s.LastMessageID != 0 {
		msgID = s.LastMessageID
	}

	e.adapter.Bot().ProcessUpdate(telebot.Update{
		Callback: &telebot.Callback{
			ID:     "cb",
			Unique: unique,
			Data:   "\f" + callbackData,
			Sender: &telebot.User{ID: e.userID},
			Message: &telebot.Message{
				ID:   msgID,
				Chat: &telebot.Chat{ID: e.chatID, Type: telebot.ChatPrivate},
			},
		},
	})
}

func (e *e2eEnv) sendDocument(fileName string, content []byte) {
	e.adapter.Bot().ProcessUpdate(telebot.Update{
		Message: &telebot.Message{
			ID: 3,
			Document: &telebot.Document{
				File:     telebot.File{FileID: "file-id", FileLocal: fileName},
				FileName: filepath.Base(fileName),
			},
			Sender: &telebot.User{ID: e.userID},
			Chat:   &telebot.Chat{ID: e.chatID, Type: telebot.ChatPrivate},
		},
	})
}
