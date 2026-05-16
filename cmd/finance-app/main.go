package main

import (
	"log"
	"path/filepath"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram"
	"github.com/a-perez/finance-app/internal/adapters/secondary/excel"
	"github.com/a-perez/finance-app/internal/adapters/secondary/ledger"
	"github.com/a-perez/finance-app/internal/app"
	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
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
	mappingServiceConstructor := func(data config.MappingData) config.MappingProvider {
		return domain.NewMappingService(data)
	}

	configManager, err := config.NewManager(configPath, mappingsPath, mappingServiceConstructor)
	if err != nil {
		log.Fatalf("Fail init config manager: %v", err)
	}
	configManager.Watch()
	defer configManager.Close()

	// Initial snapshot for static bootstrap
	conf := configManager.Get().Settings

	// Secondary Adapters
	ledgerPath := filepath.Join(env.LedgerRoot, env.LedgerFile)
	repo := ledger.NewTransactionFileRepository(ledgerPath, conf.LedgerAlignment)
	parserFactory := excel.NewParserFactory(configManager)

	// App Layer
	transactionService := app.NewTransactionService(repo)
	importService := app.NewImportService(transactionService, parserFactory)
	textParserService := app.NewTextParserService(configManager)

	// Primary Adapter
	bot, err := telegram.NewTelegramAdapter(
		env.TelegramToken,
		env.TelegramUserIDs,
		transactionService,
		textParserService,
		importService,
		configManager,
	)

	if err != nil {
		log.Fatalf("Fail init bot: %v", err)
	}

	// Start
	bot.Start()
}
