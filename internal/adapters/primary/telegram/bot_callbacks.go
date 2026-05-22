package telegram

import (
	"fmt"
	"log"
	"strconv"

	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

/*
handleConfirm persists the current transaction draft and learned mappings.
*/
func (a *TelegramAdapter) handleConfirm(c telebot.Context) error {
	userID := c.Sender().ID
	session, ok := a.sessionManager.Get(userID)

	if !ok {
		return c.Edit("Session expired. Please send the transaction again.")
	}

	if err := a.transactionUseCase.Add(session.Draft); err != nil {
		return c.Edit(fmt.Sprintf("Error saving transaction: %v", err))
	}

	// Persist mappings if overridden
	err := a.configUseCase.LearnMapping(
		session.Draft,
		session.TargetOverridden,
		session.SourceOverridden,
		session.OriginalSourceKeyword,
	)
	if err != nil {
		log.Printf("Error saving mappings: %v", err)
	}

	a.sessionManager.Delete(userID)

	appConfig := a.configUseCase.Get()
	formatted := a.formatter.FormatTransaction(session.Draft, appConfig.Settings.LedgerAlignment)
	return c.Edit(fmt.Sprintf("Transaction saved! ✅\n<pre>%s</pre>", formatted), telebot.ModeHTML)
}

/*
handleDiscard removes the current session without saving.
*/
func (a *TelegramAdapter) handleDiscard(c telebot.Context) error {
	userID := c.Sender().ID
	a.sessionManager.Delete(userID)
	return c.Edit("Transaction discarded. ❌")
}

/*
handleEditRequest transitions the session to search state for a specific posting.
*/
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

/*
handleAccountSelect applies a selected account path to the current draft posting.
*/
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

/*
handleCancelEdit returns the user to the draft preview.
*/
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

/*
handleCreateAcc starts the guided account creation flow.
*/
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

/*
handleSelectParent captures the root account and prompts for the first sub-account.
*/
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

/*
handleAddSubAcc allows extending an existing constructed path with more nesting.
*/
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

/*
handleDoneAcc finalizes the guided creation and assigns the new path to the draft.
*/
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
