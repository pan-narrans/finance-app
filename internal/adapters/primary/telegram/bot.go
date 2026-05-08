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
	mappingSvc     *domain.MappingService
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
		sessions:       make(map[int64]*userSession),
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

	// Handle Callbacks
	adapter.teleBot.Handle("\fconfirm", adapter.handleConfirm)
	adapter.teleBot.Handle("\fdiscard", adapter.handleDiscard)
	adapter.teleBot.Handle("\fedit_acc", adapter.handleEditRequest)
	adapter.teleBot.Handle("\fselect_acc", adapter.handleAccountSelect)
	adapter.teleBot.Handle("\fcancel_edit", adapter.handleCancelEdit)

	log.Printf("Bot started as @%s", adapter.teleBot.Me.Username)
	adapter.teleBot.Start()
}

func (adapter *TelegramAdapter) handleText(c telebot.Context) error {
	userID := c.Sender().ID
	adapter.mu.Lock()
	session, exists := adapter.sessions[userID]
	adapter.mu.Unlock()

	// If awaiting search query
	if exists && session.state == StateAwaitingQuery {
		return adapter.handleSearchQuery(c, session)
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
	cleanDescription := adapter.mappingSvc.CleanDescription(description)
	targetAccount := adapter.mappingSvc.ResolveAccount(cleanDescription, amount)

	// Auto-pick if Unknown
	if strings.HasSuffix(targetAccount, ":Unknown") {
		matches := adapter.mappingSvc.SearchAccounts(cleanDescription, 1)
		if len(matches) > 0 {
			targetAccount = matches[0]
		}
	}

	// Add Metadata
	metadata := make(map[string]string)
	metadata["Origin"] = "Bot"
	metadata["ID"] = adapter.hashID(fmt.Sprintf("%d", time.Now().UnixNano()))

	// Create a draft transaction
	tx := domain.Transaction{
		Date:        time.Now(),
		Status:      domain.StatusPending,
		Description: cleanDescription,
		Metadata:    metadata,
		Postings: []domain.Posting{
			{Account: targetAccount, Amount: &amount, Currency: "EUR"},
			{Account: "Assets:Cash", Amount: nil},
		},
	}
	tx.Code = tx.GenerateCode()

	// Update session
	adapter.mu.Lock()
	adapter.sessions[userID] = &userSession{draft: tx, state: StateNone}
	adapter.mu.Unlock()

	return adapter.sendDraftMessage(c, tx)
}

func (adapter *TelegramAdapter) sendDraftMessage(c telebot.Context, tx domain.Transaction) error {
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
		matches := adapter.mappingSvc.SearchAccounts(tx.Description, 5)
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

func (adapter *TelegramAdapter) handleEditRequest(c telebot.Context) error {
	userID := c.Sender().ID
	adapter.mu.Lock()
	session, ok := adapter.sessions[userID]
	if ok {
		session.state = StateAwaitingQuery
	}
	adapter.mu.Unlock()

	if !ok {
		return c.Respond(&telebot.CallbackResponse{Text: "Session expired."})
	}

	selector := &telebot.ReplyMarkup{}
	btnCancel := selector.Data("Cancel 🔙", "cancel_edit")
	selector.Inline(selector.Row(btnCancel))

	return c.Edit("Send text to search for an account.", selector)
}

func (adapter *TelegramAdapter) handleCancelEdit(c telebot.Context) error {
	userID := c.Sender().ID
	adapter.mu.Lock()
	session, ok := adapter.sessions[userID]
	if ok {
		session.state = StateNone
	}
	adapter.mu.Unlock()

	if !ok {
		return c.Edit("Session expired.")
	}

	return adapter.sendDraftMessage(c, session.draft)
}

func (adapter *TelegramAdapter) handleSearchQuery(c telebot.Context, session *userSession) error {
	query := c.Text()
	matches := adapter.mappingSvc.SearchAccounts(query, 8)

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

func (adapter *TelegramAdapter) handleAccountSelect(c telebot.Context) error {
	userID := c.Sender().ID
	newAccount := c.Data()

	adapter.mu.Lock()
	session, ok := adapter.sessions[userID]
	if ok {
		session.draft.Postings[0].Account = newAccount
		session.state = StateNone
	}
	adapter.mu.Unlock()

	if !ok {
		return c.Respond(&telebot.CallbackResponse{Text: "Session expired."})
	}

	c.Respond(&telebot.CallbackResponse{Text: "Account updated."})
	return adapter.sendDraftMessage(c, session.draft)
}

func (adapter *TelegramAdapter) handleConfirm(c telebot.Context) error {
	userID := c.Sender().ID
	adapter.mu.Lock()
	session, ok := adapter.sessions[userID]
	if ok {
		delete(adapter.sessions, userID)
	}
	adapter.mu.Unlock()

	if !ok {
		return c.Edit("Session expired. Please send the transaction again.")
	}

	if err := adapter.transactionUC.Add(session.draft); err != nil {
		return c.Edit(fmt.Sprintf("Error saving transaction: %v", err))
	}

	formatted := session.draft.Format()
	return c.Edit(fmt.Sprintf("Transaction saved! ✅\n<pre>%s</pre>", formatted), telebot.ModeHTML)
}

func (adapter *TelegramAdapter) handleDiscard(c telebot.Context) error {
	userID := c.Sender().ID
	adapter.mu.Lock()
	delete(adapter.sessions, userID)
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
