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

	"github.com/a-perez/finance-app/internal/adapters/secondary/excel"
	"github.com/a-perez/finance-app/internal/app"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

var entryRegex = regexp.MustCompile(`^(\d+([.,]\d+)?)\s+(.+)$`)

// Bot handles Telegram interactions.
type Bot struct {
	teleBot        *telebot.Bot
	allowedIDs     map[int64]struct{}
	transactionUC  ports.TransactionUseCase
	importService  *app.ImportService
	parserFactory  *excel.ParserFactory
	mappingSvc     *domain.MappingService
	ledgerFilePath string

	// Simple session storage for drafts
	mu     sync.Mutex
	drafts map[int64]domain.Transaction
}

// NewBot creates a new Telegram bot instance.
func NewBot(
	token string,
	allowedIDs []int64,
	txUC ports.TransactionUseCase,
	importSvc *app.ImportService,
	factory *excel.ParserFactory,
	mappingSvc *domain.MappingService,
	ledgerPath string,
) (*Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	allowedMap := make(map[int64]struct{})
	for _, id := range allowedIDs {
		allowedMap[id] = struct{}{}
	}

	return &Bot{
		teleBot:        b,
		allowedIDs:     allowedMap,
		transactionUC:  txUC,
		importService:  importSvc,
		parserFactory:  factory,
		mappingSvc:     mappingSvc,
		ledgerFilePath: ledgerPath,
		drafts:         make(map[int64]domain.Transaction),
	}, nil
}

// Start initializes the bot handlers and starts polling.
func (b *Bot) Start() {
	// Middleware: Auth
	b.teleBot.Use(
		func(next telebot.HandlerFunc) telebot.HandlerFunc {
			return func(c telebot.Context) error {
				if _, ok := b.allowedIDs[c.Sender().ID]; !ok {
					log.Printf("Unauthorized access attempt from User ID: %d", c.Sender().ID)
					return nil
				}
				return next(c)
			}
		},
	)

	b.teleBot.Handle(
		"/start", func(c telebot.Context) error {
			return c.Send("Welcome to Finance App Bot! Send me an amount and description (e.g., '12.50 dinner') or upload a bank file.")
		},
	)

	// Handle Manual Text Entries
	b.teleBot.Handle(telebot.OnText, b.handleText)

	// Handle File Uploads
	b.teleBot.Handle(telebot.OnDocument, b.handleDocument)

	// Handle Confirmation Callbacks
	b.teleBot.Handle("\fconfirm", b.handleConfirm)
	b.teleBot.Handle("\fdiscard", b.handleDiscard)

	log.Printf("Bot started as @%s", b.teleBot.Me.Username)
	b.teleBot.Start()
}

func (b *Bot) handleText(c telebot.Context) error {
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
	cleanDescription := b.mappingSvc.CleanDescription(description)
	targetAccount := b.mappingSvc.ResolveAccount(cleanDescription, amount)

	// Add Metadata
	metadata := make(map[string]string)
	metadata["Origin"] = "Bot"
	metadata["ID"] = b.hashID(fmt.Sprintf("%d", time.Now().UnixNano()))

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
	b.mu.Lock()
	b.drafts[c.Sender().ID] = tx
	b.mu.Unlock()

	// Inline keyboard for confirmation
	selector := &telebot.ReplyMarkup{}
	btnConfirm := selector.Data("Confirm ✅", "confirm")
	btnDiscard := selector.Data("Discard ❌", "discard")
	selector.Inline(selector.Row(btnConfirm, btnDiscard))

	// Use HTML mode for better safety with Ledger special characters
	formatted := tx.Format()
	return c.Send(fmt.Sprintf("Draft Transaction:\n<pre>%s</pre>\nConfirm?", formatted), selector, telebot.ModeHTML)
}

func (b *Bot) handleConfirm(c telebot.Context) error {
	b.mu.Lock()
	tx, ok := b.drafts[c.Sender().ID]
	delete(b.drafts, c.Sender().ID)
	b.mu.Unlock()

	if !ok {
		return c.Edit("Session expired. Please send the transaction again.")
	}

	if err := b.transactionUC.Add(tx); err != nil {
		return c.Edit(fmt.Sprintf("Error saving transaction: %v", err))
	}

	formatted := tx.Format()
	return c.Edit(fmt.Sprintf("Transaction saved! ✅\n<pre>%s</pre>", formatted), telebot.ModeHTML)
}

/*
hashID returns an 8-character MD5 hash of the provided string.
Used for generating stable external IDs for bot transactions.
*/
func (b *Bot) hashID(data string) string {
	result := ""
	if data != "" {
		hasher := md5.New()
		hasher.Write([]byte(data))
		result = fmt.Sprintf("%x", hasher.Sum(nil))[:8]
	}
	return result
}

func (b *Bot) handleDiscard(c telebot.Context) error {
	b.mu.Lock()
	delete(b.drafts, c.Sender().ID)
	b.mu.Unlock()
	return c.Edit("Transaction discarded. ❌")
}

func (b *Bot) handleDocument(c telebot.Context) error {
	doc := c.Message().Document

	// Create a temporary file to save the download
	tmpFile := doc.FileName
	err := b.teleBot.Download(&doc.File, tmpFile)
	if err != nil {
		return c.Send("Failed to download file.")
	}
	defer os.Remove(tmpFile)

	parser, err := b.parserFactory.GetParser(tmpFile)
	if err != nil {
		return c.Send(fmt.Sprintf("Unsupported file: %v", err))
	}

	summary, err := b.importService.Import(parser, tmpFile)
	if err != nil {
		return c.Send(fmt.Sprintf("Import failed: %v", err))
	}

	response := fmt.Sprintf(
		"Import Complete!\nTotal: %d\nAdded: %d\nUpdated: %d\nFailed: %d",
		summary.Total, summary.Added, summary.Updated, summary.Failed,
	)

	return c.Send(response)
}
