package telegram

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a-perez/finance-app/internal/app"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

var entryRegex = regexp.MustCompile(`^(\d+([.,]\d+)?)\s+(.+)$`)

// TelegramAdapter handles Telegram interactions.
type TelegramAdapter struct {
	teleBot        *telebot.Bot
	allowedIDs     map[int64]struct{}
	transactionUC  ports.TransactionUseCase
	importService  *app.ImportService
	mappingSvc     *domain.MappingService
	ledgerFilePath string

	// Simple session storage for drafts
	mu     sync.Mutex
	drafts map[int64]domain.Transaction
}

// NewTelegramAdapter creates a new Telegram adapter instance.
func NewTelegramAdapter(
	token string,
	allowedIDs []int64,
	txUC ports.TransactionUseCase,
	importSvc *app.ImportService,
	mappingSvc *domain.MappingService,
	ledgerPath string,
) (*TelegramAdapter, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	allowedMap := make(map[int64]struct{})
	for _, id := range allowedIDs {
		allowedMap[id] = struct{}{}
	}

	return &TelegramAdapter{
		teleBot:        bot,
		allowedIDs:     allowedMap,
		transactionUC:  txUC,
		importService:  importSvc,
		mappingSvc:     mappingSvc,
		ledgerFilePath: ledgerPath,
		drafts:         make(map[int64]domain.Transaction),
	}, nil
}

// Start initializes the bot handlers and starts polling.
func (adapter *TelegramAdapter) Start() {
	// Middleware: Auth
	adapter.teleBot.Use(
		func(next telebot.HandlerFunc) telebot.HandlerFunc {
			return func(c telebot.Context) error {
				if _, ok := adapter.allowedIDs[c.Sender().ID]; !ok {
					log.Printf("Unauthorized access attempt from User ID: %d", c.Sender().ID)
					return nil
				}
				return next(c)
			}
		},
	)

	adapter.teleBot.Handle(
		"/start", func(c telebot.Context) error {
			return c.Send("Welcome to Finance App Bot! Send me an amount and description (e.g., '12.50 dinner') or upload a bank file.")
		},
	)

	// Handle Manual Text Entries
	adapter.teleBot.Handle(telebot.OnText, adapter.handleText)

	// Handle File Uploads
	adapter.teleBot.Handle(telebot.OnDocument, adapter.handleDocument)

	// Handle Confirmation Callbacks
	adapter.teleBot.Handle("\fconfirm", adapter.handleConfirm)
	adapter.teleBot.Handle("\fdiscard", adapter.handleDiscard)

	log.Printf("Bot started as @%s", adapter.teleBot.Me.Username)
	adapter.teleBot.Start()
}

func (adapter *TelegramAdapter) handleText(c telebot.Context) error {
	text := c.Text()
	matches := entryRegex.FindStringSubmatch(text)
	if len(matches) < 4 {
		return c.Send("Format not recognized. Use: 'amount description' (e.g., '10 coffee')")
	}

	amountStr := strings.Replace(matches[1], ",", ".", 1)
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return c.Send("Invalid amount format.")
	}

	description := matches[3]
	cleanDescription := adapter.mappingSvc.CleanDescription(description)
	targetAccount := adapter.mappingSvc.ResolveAccount(cleanDescription, amount)

	// Add Metadata
	metadata := make(map[string]string)
	metadata["Origin"] = "Bot"
	metadata["ID"] = adapter.hashID(fmt.Sprintf("%d", time.Now().UnixNano()))

	// Create a draft transaction for confirmation
	tx := domain.Transaction{
		Date:        time.Now(),
		Status:      domain.StatusPending,
		Description: cleanDescription,
		Metadata:    metadata,
		Postings: []domain.Posting{
			{Account: targetAccount, Amount: &amount, Currency: "EUR"},
			{Account: "Assets:Cash", Amount: nil}, // Balanced
		},
	}
	tx.Code = tx.GenerateCode()

	// Store draft in session
	adapter.mu.Lock()
	adapter.drafts[c.Sender().ID] = tx
	adapter.mu.Unlock()

	// Inline keyboard for confirmation
	selector := &telebot.ReplyMarkup{}
	btnConfirm := selector.Data("Confirm ✅", "confirm")
	btnDiscard := selector.Data("Discard ❌", "discard")
	selector.Inline(selector.Row(btnConfirm, btnDiscard))

	// Use HTML mode for better safety with Ledger special characters
	formatted := tx.Format()
	return c.Send(fmt.Sprintf("Draft Transaction:\n<pre>%s</pre>\nConfirm?", formatted), selector, telebot.ModeHTML)
}

func (adapter *TelegramAdapter) handleConfirm(c telebot.Context) error {
	adapter.mu.Lock()
	tx, ok := adapter.drafts[c.Sender().ID]
	delete(adapter.drafts, c.Sender().ID)
	adapter.mu.Unlock()

	if !ok {
		return c.Edit("Session expired. Please send the transaction again.")
	}

	if err := adapter.transactionUC.Add(tx); err != nil {
		return c.Edit(fmt.Sprintf("Error saving transaction: %v", err))
	}

	formatted := tx.Format()
	return c.Edit(fmt.Sprintf("Transaction saved! ✅\n<pre>%s</pre>", formatted), telebot.ModeHTML)
}

/*
hashID returns an 8-character MD5 hash of the provided string.
Used for generating stable external IDs for bot transactions.
*/
func (adapter *TelegramAdapter) hashID(data string) string {
	result := ""
	if data != "" {
		hasher := md5.New()
		hasher.Write([]byte(data))
		result = fmt.Sprintf("%x", hasher.Sum(nil))[:8]
	}
	return result
}

func (adapter *TelegramAdapter) handleDiscard(c telebot.Context) error {
	adapter.mu.Lock()
	delete(adapter.drafts, c.Sender().ID)
	adapter.mu.Unlock()
	return c.Edit("Transaction discarded. ❌")
}

func (adapter *TelegramAdapter) handleDocument(c telebot.Context) error {
	doc := c.Message().Document

	// Create a temporary file to save the download
	tmpFile := doc.FileName
	err := adapter.teleBot.Download(&doc.File, tmpFile)
	if err != nil {
		return c.Send("Failed to download file.")
	}
	defer os.Remove(tmpFile)

	summary, err := adapter.importService.Import(tmpFile)
	if err != nil {
		return c.Send(fmt.Sprintf("Import failed: %v", err))
	}

	response := fmt.Sprintf(
		"Import Complete!\nTotal: %d\nAdded: %d\nUpdated: %d\nFailed: %d",
		summary.Total, summary.Added, summary.Updated, summary.Failed,
	)

	return c.Send(response)
}
