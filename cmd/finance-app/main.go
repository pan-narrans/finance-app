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
	// Load config
	env, err := config.LoadEnvironment()
	if err != nil {
		log.Fatalf("Fail load config: %v", err)
	}

	// Services & Domain
	rules, err := config.LoadMappings(filepath.Join(env.ConfigRoot, "mappings.json"))
	if err != nil {
		log.Fatalf("Fail load mappings: %v", err)
	}
	conf, err := config.LoadConfig(filepath.Join(env.ConfigRoot, "config.json"))
	if err != nil {
		log.Fatalf("Fail load mappings: %v", err)
	}

	// Secondary Adapters
	ledgerPath := filepath.Join(env.LedgerRoot, env.LedgerFile)
	repo := ledger.NewTransactionFileRepository(ledgerPath, conf.LedgerAlignment)

	mappingService := domain.NewMappingService(rules, conf)

	parserFactory := excel.NewParserFactory(mappingService, conf)

	transactionService := app.NewTransactionService(repo)
	importService := app.NewImportService(transactionService, parserFactory)
	textParserService := app.NewTextParserService(mappingService, conf)

	// Primary Adapter
	bot, err := telegram.NewTelegramAdapter(
		env.TelegramToken,
		env.TelegramUserIDs,
		transactionService,
		textParserService,
		importService,
		mappingService,
		conf,
	)

	if err != nil {
		log.Fatalf("Fail init bot: %v", err)
	}

	// Start
	bot.Start()
}
