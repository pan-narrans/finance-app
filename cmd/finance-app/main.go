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
	mappingServiceConstructor := func(data domain.MappingData, discoveredAccounts []string) ports.MappingProvider {
		return domain.NewMappingService(data, discoveredAccounts)
	}

	configManager, err := config.NewManager(configPath, mappingsPath, mappingServiceConstructor)
	if err != nil {
		log.Fatalf("Failed to initialize config manager: %v", err)
	}

	// Bootstrap authorized users from environment if not present in config file
	if len(configManager.Get().Settings.TelegramUserIDs) == 0 && len(env.TelegramUserIDs) > 0 {
		log.Printf("Bootstrapping authorized users from environment variable...")
		settings := configManager.Get().Settings
		settings.TelegramUserIDs = env.TelegramUserIDs
		configManager.ReloadWithData(settings, domain.MappingData{}) // Note: mappings will be reloaded from file soon
	}

	configManager.Watch()
	defer configManager.Close()

	// Secondary Adapters
	ledgerPath := filepath.Join(env.LedgerRoot, env.LedgerFile)
	ledgerFormatter := ledger.NewLedgerFormatter()
	repo, err := ledger.NewTransactionFileRepository(ledgerPath, configManager, ledgerFormatter)
	if err != nil {
		log.Fatalf("Failed to initialize ledger repository: %v", err)
	}
	if err := configManager.SetRepository(repo); err != nil {
		log.Fatalf("Failed to set repository: %v", err)
	}
	parserFactory := excel.NewParserFactory(configManager)

	// App Layer
	transactionService := app.NewTransactionService(repo)
	importService := app.NewImportService(transactionService, parserFactory)
	transactionParserService := app.NewTransactionParserService(configManager)
	reportService := app.NewReportService(repo, configManager)

	// Primary Adapter
	tgConfig := telegram.TelegramConfig{
		Settings: telebot.Settings{
			Token:  env.TelegramToken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		},
		AllowedIDs:    env.TelegramUserIDs,
		BotToken:      env.TelegramToken,
		WebAppBaseURL: env.WebAppBaseURL,
		HTTPPort:      env.HTTPPort,
	}

	bot, err := telegram.NewTelegramAdapter(
		tgConfig,
		transactionService,
		transactionParserService,
		importService,
		reportService,
		configManager,
		ledgerFormatter,
	)

	if err != nil {
		log.Fatalf("Fail init bot: %v", err)
	}

	// Start
	bot.Start()
}
