package telegram

import (
	"fmt"
	"strings"

	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"gopkg.in/telebot.v3"
)

// Callback constants to remove magic strings from handlers.
const (
	CallbackConfirm    = "confirm"
	CallbackDiscard    = "discard"
	CallbackEditAcc    = "edit_acc"
	CallbackSelectAcc  = "select_acc"
	CallbackCancelEdit = "cancel_edit"
)

/*
UI provides helpers for building Telegram-specific message layouts and keyboards.
*/
type UI struct {
	alignment int
}

/*
NewUI creates a new UI helper instance.
*/
func NewUI(alignment int) *UI {
	return &UI{alignment: alignment}
}

/*
BuildDraftMessage creates the text and keyboard for a transaction draft.
*/
func (u *UI) BuildDraftMessage(tx domain.Transaction, mappingProvider config.MappingProvider) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}

	btnConfirm := selector.Data("Confirm ✅", CallbackConfirm)
	btnEditTarget := selector.Data("Edit Target ✏️", CallbackEditAcc, "0")
	btnEditSource := selector.Data("Edit Source ✏️", CallbackEditAcc, "1")
	btnDiscard := selector.Data("Discard ❌", CallbackDiscard)

	rows := []telebot.Row{
		selector.Row(btnConfirm),
		selector.Row(btnEditTarget, btnEditSource),
		selector.Row(btnDiscard),
	}

	targetAccount := tx.Postings[0].Account
	msgSuffix := ""
	if strings.HasSuffix(targetAccount, ":Unknown") {
		msgSuffix = "\n\nUnknown account. Suggestions:"
		suggestions := mappingProvider.SearchAccounts(tx.Description, 5)
		for _, suggestion := range suggestions {
			btn := selector.Data(suggestion, CallbackSelectAcc, suggestion)
			rows = append(rows, selector.Row(btn))
		}
	}

	selector.Inline(rows...)

	formatted := tx.Format(u.alignment)
	msg := fmt.Sprintf("Draft Transaction:\n<pre>%s</pre>%s", formatted, msgSuffix)

	return msg, selector
}

/*
BuildEditPrompt creates the text and keyboard for an account search prompt.
*/
func (u *UI) BuildEditPrompt(isSource bool) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}
	btnCancel := selector.Data("Cancel 🔙", CallbackCancelEdit)
	selector.Inline(selector.Row(btnCancel))

	accountType := "target"
	if isSource {
		accountType = "source"
	}

	return fmt.Sprintf("Send text to search for a %s account.", accountType), selector
}

/*
BuildSearchResults creates the text and keyboard for account search results.
*/
func (u *UI) BuildSearchResults(query string, results []string) (string, *telebot.ReplyMarkup) {
	selector := &telebot.ReplyMarkup{}
	rows := make([]telebot.Row, 0)

	for _, result := range results {
		btn := selector.Data(result, CallbackSelectAcc, result)
		rows = append(rows, selector.Row(btn))
	}

	// Option to use exact input
	btnExact := selector.Data(fmt.Sprintf("Use exactly: %s", query), CallbackSelectAcc, query)
	rows = append(rows, selector.Row(btnExact))

	btnCancel := selector.Data("Cancel 🔙", CallbackCancelEdit)
	rows = append(rows, selector.Row(btnCancel))

	selector.Inline(rows...)

	return fmt.Sprintf("Search results for '%s':", query), selector
}
