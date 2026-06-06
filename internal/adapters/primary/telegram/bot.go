package telegram

import (
	"fmt"
	"log"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
	"gopkg.in/telebot.v3"
)

/*
TelegramAdapter handles interactions between users and the system via Telegram.
It implements the driving adapter pattern within the Hexagonal Architecture.

Decomposition:
  - handlers.go: Logic for processing incoming text and documents.
  - callbacks.go: Logic for processing interactive button clicks.
  - session.go: Thread-safe user session management.
  - ui.go: Layout and keyboard construction.
  - webapp_handlers.go: Logic for the Telegram Mini App API.
*/
type TelegramAdapter struct {
	teleBot             *telebot.Bot
	allowedIDs          map[int64]struct{}
	transactionUseCase  ports.TransactionUseCase
	transactionParserUC ports.TransactionParserUseCase
	importUseCase       ports.ImportUseCase
	reportUseCase       ports.ReportUseCase
	configUseCase       ports.ConfigurationUseCase
	formatter           ports.TransactionFormatter
	sessionManager      *SessionManager
	ui                  *UI
	botToken            string
	webAppBaseURL       string
	webAppServer        *WebAppServer
}

/*
TelegramConfig holds all infrastructure-specific configuration for the Telegram Adapter.
*/
type TelegramConfig struct {
	Settings      telebot.Settings
	AllowedIDs    []int64
	BotToken      string
	WebAppBaseURL string
	HTTPPort      int
}

/*
NewTelegramAdapter creates and initializes a TelegramAdapter with its dependencies.
*/
func NewTelegramAdapter(
	cfg TelegramConfig,
	txUC ports.TransactionUseCase,
	parserUC ports.TransactionParserUseCase,
	importUC ports.ImportUseCase,
	reportUC ports.ReportUseCase,
	configUC ports.ConfigurationUseCase,
	formatter ports.TransactionFormatter,
) (*TelegramAdapter, error) {
	if cfg.WebAppBaseURL == "" {
		return nil, fmt.Errorf("WEBAPP_BASE_URL is required")
	}
	if !strings.HasPrefix(cfg.WebAppBaseURL, "https://") {
		return nil, fmt.Errorf("WEBAPP_BASE_URL must start with https:// (got: %s)", cfg.WebAppBaseURL)
	}

	bot, err := telebot.NewBot(cfg.Settings)
	if err != nil {
		return nil, err
	}

	allowedMap := make(map[int64]struct{})
	for _, id := range cfg.AllowedIDs {
		allowedMap[id] = struct{}{}
	}

	sessionManager := NewSessionManager()
	adapter := &TelegramAdapter{
		teleBot:             bot,
		allowedIDs:          allowedMap,
		transactionUseCase:  txUC,
		transactionParserUC: parserUC,
		importUseCase:       importUC,
		reportUseCase:       reportUC,
		configUseCase:       configUC,
		formatter:           formatter,
		sessionManager:      sessionManager,
		ui:                  NewUI(cfg.WebAppBaseURL),
		botToken:            cfg.BotToken,
		webAppBaseURL:       cfg.WebAppBaseURL,
	}

	adapter.webAppServer = NewWebAppServer(cfg.HTTPPort, cfg.BotToken, configUC, sessionManager, adapter)

	return adapter, nil
}

/*
Start registers the bot's handlers and begins polling for updates.
*/
func (a *TelegramAdapter) Start() {
	// Middleware: Auth
	a.teleBot.Use(
		func(next telebot.HandlerFunc) telebot.HandlerFunc {
			return func(c telebot.Context) error {
				chatID := c.Chat().ID
				senderID := c.Sender().ID
				_, chatAllowed := a.allowedIDs[chatID]
				_, senderAllowed := a.allowedIDs[senderID]

				if !chatAllowed && !senderAllowed {
					log.Printf("Unauthorized access attempt from Chat ID: %d, Sender ID: %d", chatID, senderID)
					return nil
				}
				return next(c)
			}
		},
	)

	a.teleBot.Handle(
		"/start", func(c telebot.Context) error {
			return c.Send(MsgWelcome)
		},
	)

	a.teleBot.Handle(
		"/report", a.handleReport,
	)

	// Message Handlers
	a.teleBot.Handle(telebot.OnText, a.handleText)
	a.teleBot.Handle(telebot.OnDocument, a.handleDocument)

	// Callback routing
	// telebot v3 Handle() with Btn pointers only matches exact Data matches.
	// For handlers requiring payload data, we use manual prefix routing on OnCallback.
	a.teleBot.Handle(&telebot.Btn{Unique: CallbackConfirm}, a.handleConfirm)
	a.teleBot.Handle(&telebot.Btn{Unique: CallbackDiscard}, a.handleDiscard)
	a.teleBot.Handle(&telebot.Btn{Unique: CallbackCancelEdit}, a.handleCancelEdit)
	a.teleBot.Handle(&telebot.Btn{Unique: CallbackCreateAcc}, a.handleCreateAcc)
	a.teleBot.Handle(&telebot.Btn{Unique: CallbackAddSubAcc}, a.handleAddSubAcc)
	a.teleBot.Handle(&telebot.Btn{Unique: CallbackDoneAcc}, a.handleDoneAcc)
	a.teleBot.Handle(&telebot.Btn{Unique: CallbackCancelImport}, a.handleCancelImport)
	a.teleBot.Handle(&telebot.Btn{Unique: CallbackAcceptAll}, a.handleAcceptAll)

	a.teleBot.Handle(
		telebot.OnCallback, func(c telebot.Context) error {
			data := c.Callback().Data
			switch {
			case strings.HasPrefix(data, "\f"+CallbackEditAcc):
				return a.handleEditRequest(c)
			case strings.HasPrefix(data, "\f"+CallbackSelectAcc):
				return a.handleAccountSelect(c)
			case strings.HasPrefix(data, "\f"+CallbackSelectParent):
				return a.handleSelectParent(c)
			}
			return nil
		},
	)

	// Set Commands Menu for Autocomplete
	commands := []telebot.Command{
		{Text: "start", Description: "Show welcome message"},
		{Text: "report", Description: "View monthly summary (optional: 'last')"},
	}
	if err := a.teleBot.SetCommands(commands); err != nil {
		log.Printf("Warning: failed to set bot commands: %v", err)
	}

	log.Printf("Bot started as @%s", a.teleBot.Me.Username)

	// Start HTTP Server for WebApp
	go func() {
		if err := a.webAppServer.Start(); err != nil {
			log.Printf("WebApp server error: %v", err)
		}
	}()

	a.teleBot.Start()
}

/*
RefreshDraftMessage updates the existing draft message for a user.
Satisfies the MessageRefresher interface for the WebAppServer.
*/
func (a *TelegramAdapter) RefreshDraftMessage(userID int64) error {
	return a.refreshDraftMessage(userID)
}
