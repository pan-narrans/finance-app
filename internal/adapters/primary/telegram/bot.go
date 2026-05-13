package telegram

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a-perez/finance-app/internal/app"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

/*
 * TODO Refactor the whole file.
 * - Too many magic strings.
 * - Too many responsibilities.
 * - Unable to extend functionality.
 * - Extract business logic to app or domain.
 */

var entryRegex = regexp.MustCompile(`^(\d+([.,]\d+)?)\s+(.+)$`)

type SearchState string

const (
	StateNone          SearchState = ""
	StateAwaitingQuery SearchState = "awaiting_query"
)

type userSession struct {
	draft domain.Transaction
	state SearchState
}

// TelegramAdapter handles Telegram interactions.
type TelegramAdapter struct {
	teleBot        *telebot.Bot
	allowedIDs     map[int64]struct{}
	transactionUC  ports.TransactionUseCase
	importService  *app.ImportService
	mappingService *domain.MappingService
	cfg            config.Config
	ledgerFilePath string

	// Simple session storage for drafts and state
	mu       sync.Mutex
	sessions map[int64]*userSession
}

// NewTelegramAdapter creates a new Telegram adapter instance.
func NewTelegramAdapter(
	token string,
	allowedIDs []int64,
	txUC ports.TransactionUseCase,
	importService *app.ImportService,
	mappingService *domain.MappingService,
	ledgerPath string,
	cfg config.Config,
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
		importService:  importService,
		mappingService: mappingService,
		ledgerFilePath: ledgerPath,
		sessions:       make(map[int64]*userSession),
		cfg:            cfg,
	}, nil
}

// Start initializes the bot handlers and starts polling.
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

	// Handle Manual Text Entries
	a.teleBot.Handle(telebot.OnText, a.handleText)

	// Handle File Uploads
	a.teleBot.Handle(telebot.OnDocument, a.handleDocument)

	// Handle Callbacks
	a.teleBot.Handle("\fconfirm", a.handleConfirm)
	a.teleBot.Handle("\fdiscard", a.handleDiscard)
	a.teleBot.Handle("\fedit_acc", a.handleEditRequest)
	a.teleBot.Handle("\fselect_acc", a.handleAccountSelect)
	a.teleBot.Handle("\fcancel_edit", a.handleCancelEdit)

	log.Printf("Bot started as @%s", a.teleBot.Me.Username)
	a.teleBot.Start()
}

func (a *TelegramAdapter) handleText(c telebot.Context) error {
	userID := c.Sender().ID
	a.mu.Lock()
	session, exists := a.sessions[userID]
	a.mu.Unlock()

	// If awaiting search query
	if exists && session.state == StateAwaitingQuery {
		return a.handleSearchQuery(c, session)
	}

	// Else handle as new transaction entry
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
	cleanDescription := a.mappingService.CleanDescription(description)
	targetAccount := a.mappingService.ResolveAccount(cleanDescription, amount)

	// Auto-pick if Unknown
	if strings.HasSuffix(targetAccount, ":Unknown") {
		matches := a.mappingService.SearchAccounts(cleanDescription, 1)
		if len(matches) > 0 {
			targetAccount = matches[0]
		}
	}

	// Add Metadata
	metadata := domain.Metadata{
		Origin: "Bot",
		ID:     a.hashID(fmt.Sprintf("%d", time.Now().UnixNano())),
	}

	// Create a draft transaction
	tx := domain.Transaction{
		Date:        time.Now(),
		Status:      domain.StatusPending,
		Description: cleanDescription,
		Metadata:    metadata,
		Postings: []domain.Posting{
			{Account: targetAccount, Amount: &amount, Currency: a.cfg.DefaultCurrency},
			{Account: a.cfg.DefaultBotAccount, Amount: nil},
		},
	}
	tx.Code = tx.GenerateCode()

	// Update session
	a.mu.Lock()
	a.sessions[userID] = &userSession{draft: tx, state: StateNone}
	a.mu.Unlock()

	return a.sendDraftMessage(c, tx)
}

func (a *TelegramAdapter) sendDraftMessage(c telebot.Context, tx domain.Transaction) error {
	selector := &telebot.ReplyMarkup{}
	btnConfirm := selector.Data("Confirm ✅", "confirm")
	btnEdit := selector.Data("Edit Account ✏️", "edit_acc")
	btnDiscard := selector.Data("Discard ❌", "discard")

	rows := []telebot.Row{selector.Row(btnConfirm, btnEdit, btnDiscard)}

	// Auto-suggestions if still Unknown
	targetAccount := tx.Postings[0].Account
	msgSuffix := ""
	if strings.HasSuffix(targetAccount, ":Unknown") {
		msgSuffix = "\n\nUnknown account. Suggestions:"
		matches := a.mappingService.SearchAccounts(tx.Description, 5)
		for _, match := range matches {
			btn := selector.Data(match, "select_acc", match)
			rows = append(rows, selector.Row(btn))
		}
	}

	selector.Inline(rows...)

	formatted := tx.Format()
	msg := fmt.Sprintf("Draft Transaction:\n<pre>%s</pre>%s", formatted, msgSuffix)

	if c.Callback() != nil {
		return c.Edit(msg, selector, telebot.ModeHTML)
	}
	return c.Send(msg, selector, telebot.ModeHTML)
}

func (a *TelegramAdapter) handleEditRequest(c telebot.Context) error {
	userID := c.Sender().ID
	a.mu.Lock()
	session, ok := a.sessions[userID]
	if ok {
		session.state = StateAwaitingQuery
	}
	a.mu.Unlock()

	if !ok {
		return c.Respond(&telebot.CallbackResponse{Text: "Session expired."})
	}

	selector := &telebot.ReplyMarkup{}
	btnCancel := selector.Data("Cancel 🔙", "cancel_edit")
	selector.Inline(selector.Row(btnCancel))

	return c.Edit("Send text to search for an account.", selector)
}

func (a *TelegramAdapter) handleCancelEdit(c telebot.Context) error {
	userID := c.Sender().ID
	a.mu.Lock()
	session, ok := a.sessions[userID]
	if ok {
		session.state = StateNone
	}
	a.mu.Unlock()

	if !ok {
		return c.Edit("Session expired.")
	}

	return a.sendDraftMessage(c, session.draft)
}

func (a *TelegramAdapter) handleSearchQuery(c telebot.Context, session *userSession) error {
	query := c.Text()
	matches := a.mappingService.SearchAccounts(query, 8)

	selector := &telebot.ReplyMarkup{}
	rows := make([]telebot.Row, 0)

	for _, match := range matches {
		btn := selector.Data(match, "select_acc", match)
		rows = append(rows, selector.Row(btn))
	}

	// Option to use exact input
	btnExact := selector.Data(fmt.Sprintf("Use exactly: %s", query), "select_acc", query)
	rows = append(rows, selector.Row(btnExact))

	btnCancel := selector.Data("Cancel 🔙", "cancel_edit")
	rows = append(rows, selector.Row(btnCancel))

	selector.Inline(rows...)

	return c.Send(fmt.Sprintf("Search results for '%s':", query), selector)
}

func (a *TelegramAdapter) handleAccountSelect(c telebot.Context) error {
	userID := c.Sender().ID
	newAccount := c.Data()

	a.mu.Lock()
	session, ok := a.sessions[userID]
	if ok {
		session.draft.Postings[0].Account = newAccount
		session.state = StateNone
	}
	a.mu.Unlock()

	if !ok {
		return c.Respond(&telebot.CallbackResponse{Text: "Session expired."})
	}

	c.Respond(&telebot.CallbackResponse{Text: "Account updated."})
	return a.sendDraftMessage(c, session.draft)
}

func (a *TelegramAdapter) handleConfirm(c telebot.Context) error {
	userID := c.Sender().ID
	a.mu.Lock()
	session, ok := a.sessions[userID]
	if ok {
		delete(a.sessions, userID)
	}
	a.mu.Unlock()

	if !ok {
		return c.Edit("Session expired. Please send the transaction again.")
	}

	if err := a.transactionUC.Add(session.draft); err != nil {
		return c.Edit(fmt.Sprintf("Error saving transaction: %v", err))
	}

	formatted := session.draft.Format()
	return c.Edit(fmt.Sprintf("Transaction saved! ✅\n<pre>%s</pre>", formatted), telebot.ModeHTML)
}

func (a *TelegramAdapter) handleDiscard(c telebot.Context) error {
	userID := c.Sender().ID
	a.mu.Lock()
	delete(a.sessions, userID)
	a.mu.Unlock()
	return c.Edit("Transaction discarded. ❌")
}

func (a *TelegramAdapter) handleDocument(c telebot.Context) error {
	doc := c.Message().Document

	// Create a temporary file to save the download in a writable directory
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, doc.FileName)

	err := a.teleBot.Download(&doc.File, tmpFile)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to download file: %v", err))
	}
	defer os.Remove(tmpFile)

	summary, err := a.importService.Import(tmpFile)
	if err != nil {
		return c.Send(fmt.Sprintf("Import failed: %v", err))
	}

	response := fmt.Sprintf(
		"Import Complete!\nTotal: %d\nAdded: %d\nUpdated: %d\nFailed: %d",
		summary.Total, summary.Added, summary.Updated, summary.Failed,
	)

	return c.Send(response)
}

/*
hashID returns an 8-character MD5 hash of the provided string.
Used for generating stable external IDs for bot transactions.
*/
func (a *TelegramAdapter) hashID(data string) string {
	result := ""
	if data != "" {
		hasher := md5.New()
		hasher.Write([]byte(data))
		result = fmt.Sprintf("%x", hasher.Sum(nil))[:8]
	}
	return result
}
