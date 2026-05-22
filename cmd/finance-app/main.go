package main

import (
	"log"
	"path/filepath"
	"time"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram"
	"github.com/a-perez/finance-app/internal/adapters/secondary/excel"
	"github.com/a-perez/finance-app/internal/adapters/secondary/ledger"
	"github.com/a-perez/finance-app/internal/app"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

func main() {
	// Environment
	env, err := config.LoadEnvironment()
	if err != nil {
		log.Fatalf("Fail load environment: %v", err)
	}

	// Config Manager (Live Reload)
	configPath := filepath.Join(env.ConfigRoot, "config.json")
	mappingsPath := filepath.Join(env.ConfigRoot, "mappings.json")

	// Domain constructor for Config Manager
	mappingServiceConstructor := func(data domain.MappingData) ports.MappingProvider {
		return domain.NewMappingService(data)
	}

	configManager, err := config.NewManager(configPath, mappingsPath, mappingServiceConstructor)
	if err != nil {
		log.Fatalf("Fail init config manager: %v", err)
	}
	configManager.Watch()
	defer configManager.Close()

	// Secondary Adapters
	ledgerPath := filepath.Join(env.LedgerRoot, env.LedgerFile)
	ledgerFormatter := ledger.NewLedgerFormatter()
	repo := ledger.NewTransactionFileRepository(ledgerPath, configManager, ledgerFormatter)
	parserFactory := excel.NewParserFactory(configManager)

	// App Layer
	transactionService := app.NewTransactionService(repo)
	importService := app.NewImportService(transactionService, parserFactory)
	transactionParserService := app.NewTransactionParserService(configManager)

	// Primary Adapter
	bot, err := telegram.NewTelegramAdapter(
		telebot.Settings{
			Token:  env.TelegramToken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		},
		env.TelegramUserIDs,
		transactionService,
		transactionParserService,
		importService,
		configManager,
		ledgerFormatter,
	)

	if err != nil {
		log.Fatalf("Fail init bot: %v", err)
	}

	// Start
	bot.Start()
}
