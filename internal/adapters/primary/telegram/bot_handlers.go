package telegram

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

/*
handleText processes all incoming text messages.
It handles routing between search queries, account creation inputs, and new transaction entries.
*/
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
			return c.Send(MsgUseButtons, telebot.ModeHTML)
		}
	}

	// 2. Otherwise, treat as a new transaction entry
	text := c.Text()
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
	query := c.Text()

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
	child := c.Text()

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

	if c.Callback() != nil {
		return c.Edit(msg, selector, telebot.ModeHTML)
	}
	return c.Send(msg, selector, telebot.ModeHTML)
}
