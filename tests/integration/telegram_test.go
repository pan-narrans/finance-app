package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram"
	"github.com/a-perez/finance-app/internal/adapters/secondary/ledger"
	"github.com/a-perez/finance-app/internal/app"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/telebot.v3"
)

type TestPoller struct {
	updates chan telebot.Update
}

func (p *TestPoller) Poll(b *telebot.Bot, dest chan telebot.Update, stop chan struct{}) {
	for {
		select {
		case u := <-p.updates:
			dest <- u
		case <-stop:
			return
		}
	}
}

type testEnv struct {
	adapter       *telegram.TelegramAdapter
	poller        *TestPoller
	ledgerPath    string
	configManager *config.Manager
	userID        int64
	tmpDir        string
}

func setupTestEnv(t *testing.T) *testEnv {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"ok":true,"result":{"message_id":1}}`)
	}))
	t.Cleanup(server.Close)

	tmpDir, err := os.MkdirTemp("", "bot-integration")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	ledgerPath := filepath.Join(tmpDir, "test.ledger")
	configPath := filepath.Join(tmpDir, "config.json")
	mappingsPath := filepath.Join(tmpDir, "mappings.json")

	// Base config with known mappings for tests
	configJSON := `{
		"ledger_alignment":40,
		"default_asset_account": "Assets:Cash",
		"default_currency": "EUR",
		"default_income_account": "Income:Unknown",
		"default_expense_account": "Expenses:Unknown"
	}`
	mappingsJSON := `{
		"accounts": {
			"DINNER": "Expenses:Food"
		},
		"sources": {
			"visa": "Assets:Bank:Visa"
		}
	}`
	require.NoError(t, os.WriteFile(configPath, []byte(configJSON), 0644))
	require.NoError(t, os.WriteFile(mappingsPath, []byte(mappingsJSON), 0644))

	configManager, err := config.NewManager(configPath, mappingsPath, func(data domain.MappingData) ports.MappingProvider {
		return domain.NewMappingService(data)
	})
	require.NoError(t, err)

	repo := ledger.NewTransactionFileRepository(ledgerPath, configManager, ledger.NewLedgerFormatter())
	txService := app.NewTransactionService(repo)
	parserService := app.NewTextParserService(configManager)

	poller := &TestPoller{updates: make(chan telebot.Update, 10)}
	settings := telebot.Settings{
		URL:    server.URL,
		Token:  "fake-token",
		Poller: poller,
	}

	userID := int64(12345)
	adapter, err := telegram.NewTelegramAdapter(
		settings,
		[]int64{userID},
		txService,
		parserService,
		nil,
		configManager,
		ledger.NewLedgerFormatter(),
	)
	require.NoError(t, err)

	go adapter.Start()
	time.Sleep(200 * time.Millisecond)

	return &testEnv{
		adapter:       adapter,
		poller:        poller,
		ledgerPath:    ledgerPath,
		configManager: configManager,
		userID:        userID,
		tmpDir:        tmpDir,
	}
}

func (e *testEnv) sendText(text string) {
	e.poller.updates <- telebot.Update{
		Message: &telebot.Message{
			ID:     1,
			Text:   text,
			Sender: &telebot.User{ID: e.userID},
			Chat:   &telebot.Chat{ID: e.userID},
		},
	}
	time.Sleep(200 * time.Millisecond)
}

func (e *testEnv) sendCallback(data string) {
	e.poller.updates <- telebot.Update{
		Callback: &telebot.Callback{
			ID:     "cb",
			Unique: data,
			Data:   "\f" + data,
			Sender: &telebot.User{ID: e.userID},
			Message: &telebot.Message{
				ID:   2,
				Chat: &telebot.Chat{ID: e.userID},
			},
		},
	}
	time.Sleep(200 * time.Millisecond)
}

func (e *testEnv) sendCallbackWithRawData(data string) {
	e.poller.updates <- telebot.Update{
		Callback: &telebot.Callback{
			ID:     "cb",
			Sender: &telebot.User{ID: e.userID},
			Data:   data,
			Message: &telebot.Message{
				ID:   2,
				Chat: &telebot.Chat{ID: e.userID},
			},
		},
	}
	time.Sleep(200 * time.Millisecond)
}

func TestTelegramIntegration_HappyPaths(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("1. Simple Input + Mapping", func(t *testing.T) {
		env.sendText("10 Dinner")
		env.sendCallback(telegram.CallbackConfirm)

		content, _ := os.ReadFile(env.ledgerPath)
		assert.Contains(t, string(content), "Expenses:Food")
		assert.Contains(t, string(content), "10.00")
		os.Truncate(env.ledgerPath, 0)
	})

	t.Run("3. Source + Mapping", func(t *testing.T) {
		env.sendText("Visa 20 Dinner")
		env.sendCallback(telegram.CallbackConfirm)

		content, _ := os.ReadFile(env.ledgerPath)
		assert.Contains(t, string(content), "Expenses:Food")
		assert.Contains(t, string(content), "Assets:Bank:Visa")
		assert.Contains(t, string(content), "20.00")
		os.Truncate(env.ledgerPath, 0)
	})
}

func TestTelegramIntegration_AccountCreation(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("Creation Flow - Combined", func(t *testing.T) {
		// 1. Send transaction
		env.sendText("50 NewGadget")

		// 2. Edit Expense Account (Target)
		env.sendCallback(telegram.CallbackEditAcc + "0")
		env.sendText("Gadgets") // Search query
		env.sendCallback(telegram.CallbackCreateAcc)
		env.sendCallback(telegram.CallbackSelectParent + "Expenses")
		env.sendText("Tech")
		env.sendCallback(telegram.CallbackDoneAcc)

		// 3. Edit Source Account
		env.sendCallback(telegram.CallbackEditAcc + "1")
		env.sendText("Bank") // Search query
		env.sendCallback(telegram.CallbackCreateAcc)
		env.sendCallback(telegram.CallbackSelectParent + "Assets")
		env.sendText("Revolut")
		env.sendCallback(telegram.CallbackDoneAcc)

		// 4. Confirm
		env.sendCallback(telegram.CallbackConfirm)

		content, _ := os.ReadFile(env.ledgerPath)
		output := string(content)
		assert.Contains(t, output, "Expenses:Tech")
		assert.Contains(t, output, "Assets:Revolut")
		assert.Contains(t, output, "50.00")
	})
}

func TestTelegramIntegration_EdgeCases(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("Unauthorized User", func(t *testing.T) {
		env.poller.updates <- telebot.Update{
			Message: &telebot.Message{
				Text:   "10 Hack",
				Sender: &telebot.User{ID: 999},
				Chat:   &telebot.Chat{ID: 999},
			},
		}
		time.Sleep(200 * time.Millisecond)

		_, err := os.Stat(env.ledgerPath)
		assert.True(t, os.IsNotExist(err) || env.isLedgerEmpty())
	})
}

func (e *testEnv) isLedgerEmpty() bool {
	content, err := os.ReadFile(e.ledgerPath)
	if err != nil {
		return true
	}
	return len(strings.TrimSpace(string(content))) == 0
}

func TestTelegramIntegration_MappingPersistence(t *testing.T) {
	env := setupTestEnv(t)
	mappingsPath := filepath.Join(env.tmpDir, "mappings.json")

	t.Run("Persistence after Selection", func(t *testing.T) {
		// 1. Send transaction that results in Unknown
		env.sendText("10 Coffee")

		// 2. Select an account from suggestions or manual
		env.sendCallback(telegram.CallbackSelectAcc + "Expenses:Drinks")

		// 3. Confirm
		env.sendCallback(telegram.CallbackConfirm)

		// 4. Verify Mappings File
		content, _ := os.ReadFile(mappingsPath)
		assert.Contains(t, string(content), `"COFFEE": "Expenses:Drinks"`)
	})

	t.Run("Persistence after Creation", func(t *testing.T) {
		// 1. Send transaction
		env.sendText("25 Internet")

		// 2. Create new account
		env.sendCallback(telegram.CallbackEditAcc + "0")
		env.sendText("Web")
		env.sendCallback(telegram.CallbackCreateAcc)
		env.sendCallback(telegram.CallbackSelectParent + "Expenses")
		env.sendText("Utilities")
		env.sendCallback(telegram.CallbackDoneAcc)

		// 3. Confirm
		env.sendCallback(telegram.CallbackConfirm)

		// 4. Verify Mappings File
		content, _ := os.ReadFile(mappingsPath)
		assert.Contains(t, string(content), `"INTERNET": "Expenses:Utilities"`)
	})

	t.Run("Persistence of Source Override", func(t *testing.T) {
		// 1. Send transaction with new source
		env.sendText("Cash 15 Beer")

		// 2. Edit Source Account
		env.sendCallback(telegram.CallbackEditAcc + "1")
		env.sendCallback(telegram.CallbackSelectAcc + "Assets:Cash:Personal")

		// 3. Confirm
		env.sendCallback(telegram.CallbackConfirm)

		// 4. Verify Mappings File
		content, _ := os.ReadFile(mappingsPath)
		assert.Contains(t, string(content), `"cash": "Assets:Cash:Personal"`)
	})
}
