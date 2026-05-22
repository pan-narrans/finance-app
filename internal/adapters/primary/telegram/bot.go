package telegram

import (
	"log"

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
*/
type TelegramAdapter struct {
	teleBot             *telebot.Bot
	allowedIDs          map[int64]struct{}
	transactionUseCase  ports.TransactionUseCase
	transactionParserUC ports.TransactionParserUseCase
	importUseCase       ports.ImportUseCase
	configUseCase       ports.ConfigurationUseCase
	formatter           ports.TransactionFormatter
	sessionManager      *SessionManager
	ui                  *UI
}

/*
NewTelegramAdapter creates and initializes a TelegramAdapter with its dependencies.
*/
func NewTelegramAdapter(
	settings telebot.Settings,
	allowedIDs []int64,
	txUC ports.TransactionUseCase,
	parserUC ports.TransactionParserUseCase,
	importUC ports.ImportUseCase,
	configUC ports.ConfigurationUseCase,
	formatter ports.TransactionFormatter,
) (*TelegramAdapter, error) {
	bot, err := telebot.NewBot(settings)
	if err != nil {
		return nil, err
	}

	allowedMap := make(map[int64]struct{})
	for _, id := range allowedIDs {
		allowedMap[id] = struct{}{}
	}

	return &TelegramAdapter{
		teleBot:             bot,
		allowedIDs:          allowedMap,
		transactionUseCase:  txUC,
		transactionParserUC: parserUC,
		importUseCase:       importUC,
		configUseCase:       configUC,
		formatter:           formatter,
		sessionManager:      NewSessionManager(),
		ui:                  NewUI(),
	}, nil
}

/*
Start registers the bot's handlers and begins polling for updates.
*/
func (a *TelegramAdapter) Start() {
	// Middleware: Auth
	a.teleBot.Use(
		func(next telebot.HandlerFunc) telebot.HandlerFunc {
			return func(c telebot.Context) error {
				if _, ok := a.allowedIDs[c.Sender().ID]; !ok {
					log.Printf("Unauthorized access attempt from User ID: %d", c.Sender().ID)
					return nil
				}
				return next(c)
			}
		},
	)

	a.teleBot.Handle(
		"/start", func(c telebot.Context) error {
			return c.Send("Welcome to Finance App Bot! Send me an amount and description (e.g., '12.50 dinner') or upload a bank file.")
		},
	)

	// Message Handlers
	a.teleBot.Handle(telebot.OnText, a.handleText)
	a.teleBot.Handle(telebot.OnDocument, a.handleDocument)

	// Callback routing using centralized constants
	a.teleBot.Handle("\f"+CallbackConfirm, a.handleConfirm)
	a.teleBot.Handle("\f"+CallbackDiscard, a.handleDiscard)
	a.teleBot.Handle("\f"+CallbackEditAcc, a.handleEditRequest)
	a.teleBot.Handle("\f"+CallbackSelectAcc, a.handleAccountSelect)
	a.teleBot.Handle("\f"+CallbackCancelEdit, a.handleCancelEdit)
	a.teleBot.Handle("\f"+CallbackCreateAcc, a.handleCreateAcc)
	a.teleBot.Handle("\f"+CallbackSelectParent, a.handleSelectParent)
	a.teleBot.Handle("\f"+CallbackAddSubAcc, a.handleAddSubAcc)
	a.teleBot.Handle("\f"+CallbackDoneAcc, a.handleDoneAcc)

	log.Printf("Bot started as @%s", a.teleBot.Me.Username)
	a.teleBot.Start()
}
