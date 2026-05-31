package telegram

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

func (a *TelegramAdapter) handleReport(c telebot.Context) error {
	period := "this month"
	args := c.Args()
	if len(args) > 0 && args[0] == "last" {
		period = "last month"
	}

	sections, err := a.reportUseCase.GetMonthlyReport(period)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to generate report: %v", err))
	}

	if len(sections) == 0 {
		return c.Send(fmt.Sprintf("No data for %s.", period))
	}

	for _, section := range sections {
		msg := fmt.Sprintf("<b>%s %s</b>\n<pre>%s</pre>", section.Title, section.DateRange, section.Content)
		if err := c.Send(msg, telebot.ModeHTML); err != nil {
			return err
		}
	}

	return nil
}

/*
handleText processes all incoming text messages.
It handles routing between search queries, account creation inputs, and new transaction entries.
*/
func (a *TelegramAdapter) handleText(c telebot.Context) error {
	if !a.isTriggered(c) {
		return nil
	}

	text := c.Text()
	userID := c.Sender().ID
	session, exists := a.sessionManager.Get(userID)

	// 1. Handle Commands
	if strings.HasPrefix(text, "/") {
		parts := strings.Fields(text)
		if len(parts) > 1 {
			// Command with arguments (e.g. /transaction 10 steam)
			text = strings.Join(parts[1:], " ")
		} else if parts[0] == "/transaction" || strings.HasPrefix(parts[0], "/transaction@") {
			// Bare command - start interactive flow
			a.sessionManager.Set(userID, &UserSession{
				State: StateAwaitingTransactionInput,
			})

			// Use ForceReply to ensure the reply is delivered to the bot in groups
			// and InputFieldPlaceholder to guide the user.
			selector := &telebot.ReplyMarkup{
				ForceReply:  true,
				Selective:   true,
				Placeholder: "e.g. Cash 10 steam",
			}
			return c.Send(MsgPromptTransaction, selector, telebot.ModeHTML)
		} else {
			return nil
		}
	}

	// 2. Handle State-based inputs
	if exists && session.State != StateNone {
		switch session.State {
		case StateAwaitingTransactionInput:
			// Process as transaction text and proceed
			a.sessionManager.Update(userID, func(s *UserSession) {
				s.State = StateNone
			})
			// fallthrough to cleaning logic
		case StateAwaitingQuery:
			return a.handleSearchQuery(c)
		case StateCreatingAccountChild:
			return a.handleChildInput(c)
		case StateCreatingAccountParent, StateCreatingAccountReview:
			return c.Send(MsgUseButtons, telebot.ModeHTML)
		}
	}

	// 3. Clean mentions and formatting
	text = a.getCleanedText(c)

	// 4. Strip leading @ if it survived getCleanedText (e.g. misspelled mention)
	if strings.HasPrefix(text, "@") {
		fields := strings.Fields(text)
		if len(fields) > 1 {
			text = strings.Join(fields[1:], " ")
		}
	}

	// 5. Otherwise, treat as a new transaction entry
	tx, err := a.transactionParserUC.ParseText(text, "Telegram")
	if err != nil {
		return c.Send(err.Error())
	}

	// Capture source keyword for potential mapping update
	sourceKeyword := a.transactionParserUC.GuessSource(text)

	// Store in session
	a.sessionManager.Set(userID, &UserSession{
		Draft:                 tx,
		State:                 StateNone,
		OriginalSourceKeyword: sourceKeyword,
	})

	return a.sendDraftMessage(c, tx)
}

/*
handleDocument processes uploaded files (e.g., bank statements).
It downloads the file to a temporary location and triggers the import use case.
*/
func (a *TelegramAdapter) handleDocument(c telebot.Context) error {
	if !a.isTriggered(c) {
		return nil
	}
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

	if len(summary.Pending) > 0 {
		userID := c.Sender().ID
		firstPending := summary.Pending[0]
		a.sessionManager.Set(userID, &UserSession{
			Draft:                 firstPending,
			PendingQueue:          summary.Pending[1:],
			OriginalSourceKeyword: a.transactionParserUC.GuessSource(firstPending.Description),
		})

		response += fmt.Sprintf("\n\n<b>%d transactions need review.</b>", len(summary.Pending))
		c.Send(response, telebot.ModeHTML)
		return a.sendDraftMessage(c, firstPending)
	}

	return c.Send(response)
}

/*
handleSearchQuery processes text input when the user is searching for an account.
If the query contains a colon, it's treated as a direct account path selection.
*/
func (a *TelegramAdapter) handleSearchQuery(c telebot.Context) error {
	query := a.getCleanedText(c)

	// Direct Path Override
	if strings.Contains(query, ":") {
		return a.handleAccountSelect(c)
	}

	results := a.configUseCase.Get().Mappings.SearchAccounts(query, 8)

	msg, selector := a.ui.BuildSearchResults(query, results)
	return c.Send(msg, selector, telebot.ModeHTML)
}

/*
handleChildInput processes text input when the user is providing a sub-account name.
*/
func (a *TelegramAdapter) handleChildInput(c telebot.Context) error {
	userID := c.Sender().ID
	child := a.getCleanedText(c)

	session, ok := a.sessionManager.Get(userID)
	if !ok {
		return c.Send(MsgSessionExpired + " Please start over.")
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

/*
isTriggered checks if the message should be processed.
In private chats, it always returns true.
In groups, it returns true if:
  - It's a command
  - The bot is mentioned
  - The user has an active session state
  - It's a reply to one of the bot's messages
*/
func (a *TelegramAdapter) isTriggered(c telebot.Context) bool {
	msg := c.Message()
	if msg == nil {
		return false
	}

	// Always trigger in private chats
	if c.Chat().Type == telebot.ChatPrivate || c.Chat().Type == "private" || c.Chat().ID > 0 {
		return true
	}

	// Trigger on commands
	if strings.HasPrefix(c.Text(), "/") {
		return true
	}

	// Trigger on mentions (case-insensitive)
	username := a.teleBot.Me.Username
	if username != "" {
		lowerText := strings.ToLower(c.Text())
		lowerMention := "@" + strings.ToLower(username)
		if strings.Contains(lowerText, lowerMention) {
			return true
		}
	} else if strings.Contains(c.Text(), "@") {
		// Fallback for cases where username might be delayed
		return true
	}

	// Trigger on active session state (important for group flows)
	userID := c.Sender().ID
	if session, exists := a.sessionManager.Get(userID); exists && session.State != StateNone {
		return true
	}

	// Trigger on replies to bot's messages
	if msg.IsReply() && msg.ReplyTo.Sender.ID == a.teleBot.Me.ID {
		return true
	}

	return false
}

/*
sendDraftMessage helper sends or edits the transaction preview with confirmation buttons.
*/
func (a *TelegramAdapter) sendDraftMessage(c telebot.Context, tx domain.Transaction) error {
	appConfig := a.configUseCase.Get()
	userID := c.Sender().ID
	session, _ := a.sessionManager.Get(userID)

	var msg string
	var selector *telebot.ReplyMarkup

	// Check if this is part of an import review flow (Origin is not Telegram)
	isImportReview := session != nil && (len(session.PendingQueue) > 0 || session.Draft.Metadata.Origin != "Telegram")

	if isImportReview {
		msg, selector = a.ui.BuildImportReviewMessage(tx, len(session.PendingQueue), appConfig.Mappings, appConfig.Settings, a.formatter)
	} else {
		msg, selector = a.ui.BuildDraftMessage(tx, appConfig.Mappings, appConfig.Settings, a.formatter)
	}

	var sentMsg *telebot.Message
	var err error

	if c.Callback() != nil {
		sentMsg, err = a.teleBot.Edit(c.Message(), msg, selector, telebot.ModeHTML)
	} else {
		sentMsg, err = a.teleBot.Send(c.Chat(), msg, selector, telebot.ModeHTML)
	}

	if err == nil && sentMsg != nil {
		a.sessionManager.Update(userID, func(s *UserSession) {
			s.LastMessageID = sentMsg.ID
			s.LastChatID = sentMsg.Chat.ID
		})
	}

	return err
}

/*
refreshDraftMessage updates the existing draft message for a user.
Used primarily for asynchronous updates (like from the Mini App).
*/
func (a *TelegramAdapter) refreshDraftMessage(userID int64) error {
	session, ok := a.sessionManager.Get(userID)
	if !ok || session.LastMessageID == 0 {
		return fmt.Errorf("no active session or message to refresh")
	}

	appConfig := a.configUseCase.Get()
	tx := session.Draft

	var msg string
	var selector *telebot.ReplyMarkup

	isImportReview := len(session.PendingQueue) > 0 || tx.Metadata.Origin != "Telegram"

	if isImportReview {
		msg, selector = a.ui.BuildImportReviewMessage(tx, len(session.PendingQueue), appConfig.Mappings, appConfig.Settings, a.formatter)
	} else {
		msg, selector = a.ui.BuildDraftMessage(tx, appConfig.Mappings, appConfig.Settings, a.formatter)
	}

	editable := &telebot.Message{
		ID:   session.LastMessageID,
		Chat: &telebot.Chat{ID: session.LastChatID},
	}

	_, err := a.teleBot.Edit(editable, msg, selector, telebot.ModeHTML)
	return err
}
