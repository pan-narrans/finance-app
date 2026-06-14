package e2e

import (
	"os"
	"path/filepath"
	"strings"
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

	botToken := os.Getenv("TELEGRAM_TOKEN")
	require.NotEmpty(t, botToken, "TELEGRAM_TOKEN must be set for E2E tests")

	// We might need a real User ID and Chat ID for some assertions if we want to check real Telegram state
	// but for now we'll use them to simulate the "from" user.
	userID := int64(12345) // Default or from env
	chatID := int64(12345)

	tmpDir := t.TempDir()
	ledgerPath := filepath.Join(tmpDir, "test.ledger")
	configPath := filepath.Join(tmpDir, "config.json")
	mappingsPath := filepath.Join(tmpDir, "mappings.json")

	// Base config
	configJSON := `{
		"settings": {
			"default_currency": "EUR",
			"imagin_bank_account": "Assets:Checking:ImaginBank",
			"default_income_account": "Income:Unknown",
			"default_expense_account": "Expenses:Unknown",
			"default_asset_account": "Assets:Cash",
			"root_accounts": ["Assets", "Expenses"]
		}
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

	// In E2E, we use the real Telegram API
	settings := telebot.Settings{
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

	e.adapter.Bot().ProcessUpdate(telebot.Update{
		Callback: &telebot.Callback{
			ID:     "cb",
			Unique: unique,
			Data:   "\f" + callbackData,
			Sender: &telebot.User{ID: e.userID},
			Message: &telebot.Message{
				ID:   2,
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
