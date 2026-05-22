package telegram

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

//TODO missing godoc
//TODO file too big

/*
TelegramAdapter handles interactions between users and the system via Telegram.
It implements the driving adapter pattern within the Hexagonal Architecture.
*/
type TelegramAdapter struct {
	teleBot             *telebot.Bot
	allowedIDs          map[int64]struct{}
	transactionUC       ports.TransactionUseCase
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
		transactionUC:       txUC,
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

	// Routing logic
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
func (a *TelegramAdapter) handleText(c telebot.Context) error {
	userID := c.Sender().ID
	session, exists := a.sessionManager.Get(userID)

	if exists {
		switch session.State {
		case StateAwaitingQuery:
			return a.handleSearchQuery(c)
		case StateCreatingAccountChild:
			return a.handleChildInput(c)
		case StateCreatingAccountParent, StateCreatingAccountReview:
			return c.Send("Please use the buttons provided to continue or click Cancel.", telebot.ModeHTML)
		}
	}

	// 2. Otherwise, treat as a new transaction entry
	text := c.Text()
	tx, err := a.transactionParserUC.ParseText(text, "Telegram")
	if err != nil {
		return c.Send(err.Error())
	}

	// Capture source keyword for potential mapping update
	sourceKeyword := ""
	words := strings.Fields(text)
	if len(words) > 0 {
		// Simple heuristic: if first word is alphabetic and followed by a number, it's likely the source
		if matched, _ := regexp.MatchString(`^[a-zA-Z]+$`, words[0]); matched && len(words) > 1 {
			if matchedAmt, _ := regexp.MatchString(`^\d`, words[1]); matchedAmt {
				sourceKeyword = strings.ToLower(words[0])
			}
		}
	}

	// Store in session
	a.sessionManager.Set(userID, &UserSession{
		Draft:                 tx,
		State:                 StateNone,
		OriginalSourceKeyword: sourceKeyword,
	})

	return a.sendDraftMessage(c, tx)
}

func (a *TelegramAdapter) sendDraftMessage(c telebot.Context, tx domain.Transaction) error {
	appConfig := a.configUseCase.Get()
	msg, selector := a.ui.BuildDraftMessage(tx, appConfig.Mappings, appConfig.Settings, a.formatter)

	if c.Callback() != nil {
		return c.Edit(msg, selector, telebot.ModeHTML)
	}
	return c.Send(msg, selector, telebot.ModeHTML)
}

func (a *TelegramAdapter) handleEditRequest(c telebot.Context) error {
	userID := c.Sender().ID
	postingIndex, _ := strconv.Atoi(c.Data())

	session, ok := a.sessionManager.Get(userID)
	if !ok {
		return c.Respond(&telebot.CallbackResponse{Text: "Session expired."})
	}

	a.sessionManager.Update(userID, func(s *UserSession) {
		s.State = StateAwaitingQuery
		s.EditingPosting = postingIndex
	})

	results := a.configUseCase.Get().Mappings.SearchAccounts(session.Draft.Description, 5)

	msg, selector := a.ui.BuildEditPrompt(postingIndex == 1, results)
	return c.Edit(msg, selector, telebot.ModeHTML)
}

func (a *TelegramAdapter) handleCancelEdit(c telebot.Context) error {
	userID := c.Sender().ID
	session, ok := a.sessionManager.Get(userID)
	if !ok {
		return c.Edit("Session expired.")
	}

	a.sessionManager.Update(userID, func(s *UserSession) {
		s.State = StateNone
	})

	return a.sendDraftMessage(c, session.Draft)
}

func (a *TelegramAdapter) handleSearchQuery(c telebot.Context) error {
	query := c.Text()

	// Direct Path Override
	if strings.Contains(query, ":") {
		return a.handleAccountSelect(c)
	}

	results := a.configUseCase.Get().Mappings.SearchAccounts(query, 8)

	msg, selector := a.ui.BuildSearchResults(query, results)
	return c.Send(msg, selector, telebot.ModeHTML)
}

func (a *TelegramAdapter) handleAccountSelect(c telebot.Context) error {
	userID := c.Sender().ID
	newAccount := c.Data()
	if newAccount == "" {
		newAccount = c.Text()
	}

	session, ok := a.sessionManager.Get(userID)
	if !ok {
		if c.Callback() != nil {
			return c.Respond(&telebot.CallbackResponse{Text: "Session expired."})
		}
		return c.Send("Session expired.")
	}

	// Format and detect if it's a manual override
	formattedAccount := domain.FormatAccountPath(newAccount)

	a.sessionManager.Update(userID, func(s *UserSession) {
		if len(s.Draft.Postings) > s.EditingPosting {
			s.Draft.Postings[s.EditingPosting].Account = formattedAccount
		}
		s.State = StateNone
		if s.EditingPosting == 0 {
			s.TargetOverridden = true
		} else if s.EditingPosting == 1 {
			s.SourceOverridden = true
		}
	})

	if c.Callback() != nil {
		c.Respond(&telebot.CallbackResponse{Text: "Account updated."})
	}
	return a.sendDraftMessage(c, session.Draft)
}

func (a *TelegramAdapter) handleConfirm(c telebot.Context) error {
	userID := c.Sender().ID
	session, ok := a.sessionManager.Get(userID)

	if !ok {
		return c.Edit("Session expired. Please send the transaction again.")
	}

	if err := a.transactionUC.Add(session.Draft); err != nil {
		return c.Edit(fmt.Sprintf("Error saving transaction: %v", err))
	}

	// Persist mappings if overridden
	if session.TargetOverridden || session.SourceOverridden {
		err := a.configUseCase.UpdateMapping(func(data *domain.MappingData) {
			if session.TargetOverridden {
				key := strings.ToUpper(session.Draft.Description)
				data.Accounts[key] = session.Draft.Postings[0].Account
			}
			if session.SourceOverridden && session.OriginalSourceKeyword != "" {
				key := strings.ToLower(session.OriginalSourceKeyword)
				data.Sources[key] = session.Draft.Postings[1].Account
			}
		})
		if err != nil {
			log.Printf("Error saving mappings: %v", err)
		}
	}

	a.sessionManager.Delete(userID)

	appConfig := a.configUseCase.Get()
	formatted := a.formatter.FormatTransaction(session.Draft, appConfig.Settings.LedgerAlignment)
	return c.Edit(fmt.Sprintf("Transaction saved! ✅\n<pre>%s</pre>", formatted), telebot.ModeHTML)
}

func (a *TelegramAdapter) handleCreateAcc(c telebot.Context) error {
	userID := c.Sender().ID
	_, ok := a.sessionManager.Get(userID)
	if !ok {
		return c.Edit("Session expired. Please start over.")
	}

	a.sessionManager.Update(userID, func(s *UserSession) {
		s.State = StateCreatingAccountParent
		s.NewAccountPath = ""
	})

	msg, selector := a.ui.BuildAccountParentSelector(a.configUseCase.Get().Settings.RootAccounts)
	return c.Edit(msg, selector, telebot.ModeHTML)
}

func (a *TelegramAdapter) handleSelectParent(c telebot.Context) error {
	userID := c.Sender().ID
	parent := c.Data()

	_, ok := a.sessionManager.Get(userID)
	if !ok {
		return c.Edit("Session expired. Please start over.")
	}

	a.sessionManager.Update(userID, func(s *UserSession) {
		s.State = StateCreatingAccountChild
		s.NewAccountPath = parent
	})

	msg, selector := a.ui.BuildAccountChildPrompt(parent)
	return c.Edit(msg, selector, telebot.ModeHTML)
}

func (a *TelegramAdapter) handleChildInput(c telebot.Context) error {
	userID := c.Sender().ID
	child := c.Text()

	session, ok := a.sessionManager.Get(userID)
	if !ok {
		return c.Send("Session expired. Please start over.")
	}
	newPath := session.NewAccountPath + ":" + child
	formattedPath := domain.FormatAccountPath(newPath)

	a.sessionManager.Update(userID, func(s *UserSession) {
		s.State = StateCreatingAccountReview
		s.NewAccountPath = formattedPath
	})

	msg, selector := a.ui.BuildAccountReview(formattedPath)
	return c.Send(msg, selector, telebot.ModeHTML)
}

func (a *TelegramAdapter) handleAddSubAcc(c telebot.Context) error {
	userID := c.Sender().ID
	session, ok := a.sessionManager.Get(userID)
	if !ok {
		return c.Edit("Session expired.")
	}

	a.sessionManager.Update(userID, func(s *UserSession) {
		s.State = StateCreatingAccountChild
	})

	msg, selector := a.ui.BuildAccountChildPrompt(session.NewAccountPath)
	return c.Edit(msg, selector, telebot.ModeHTML)
}

func (a *TelegramAdapter) handleDoneAcc(c telebot.Context) error {
	userID := c.Sender().ID
	session, ok := a.sessionManager.Get(userID)
	if !ok {
		return c.Edit("Session expired.")
	}

	formattedPath := domain.FormatAccountPath(session.NewAccountPath)

	a.sessionManager.Update(userID, func(s *UserSession) {
		if len(s.Draft.Postings) > s.EditingPosting {
			s.Draft.Postings[s.EditingPosting].Account = formattedPath
		}
		s.State = StateNone
		if s.EditingPosting == 0 {
			s.TargetOverridden = true
		} else if s.EditingPosting == 1 {
			s.SourceOverridden = true
		}
	})

	c.Respond(&telebot.CallbackResponse{Text: "Account created and selected."})
	return a.sendDraftMessage(c, session.Draft)
}

func (a *TelegramAdapter) handleDiscard(c telebot.Context) error {
	userID := c.Sender().ID
	a.sessionManager.Delete(userID)
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

	summary, err := a.importUseCase.Import(tmpFile)
	if err != nil {
		return c.Send(fmt.Sprintf("Import failed: %v", err))
	}

	response := fmt.Sprintf(
		"Import Complete!\nTotal: %d\nAdded: %d\nUpdated: %d\nFailed: %d",
		summary.Total, summary.Added, summary.Updated, summary.Failed,
	)

	return c.Send(response)
}
